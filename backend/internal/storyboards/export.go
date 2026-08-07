package storyboards

// JSON export (spec 11). Synchronous, in-memory -- following
// ewrite.HandleLibraryExport's pattern rather than the async job/polling
// machinery in identity/account_export.go, since a single board's JSON is
// small and bounded and building polling infrastructure for it would
// itself be unrequested scope beyond what Kernel 80 needs.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ExportFormat = "victory-storyboard"
const ExportFormatVersion = 1

type ExportBoard struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	OwnerUserID     string     `json:"owner_user_id"`
	OwnerHandle     string     `json:"owner_handle,omitempty"`
	Mode            string     `json:"mode"`
	TemplateVersion *int       `json:"template_version,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

type ExportGrant struct {
	UserID      string    `json:"user_id"`
	UserHandle  string    `json:"user_handle,omitempty"`
	GrantedRole string    `json:"granted_role"`
	GrantedAt   time.Time `json:"granted_at"`
}

// ExportEwriteLink is a reference only -- publication ID and section
// anchor -- never the linked publication's resolved content (spec 11.4's
// "no ... inaccessible eWrite content" is satisfied by construction: the
// export never embeds eWrite content at all, hidden or not).
type ExportEwriteLink struct {
	PublicationID string `json:"publication_id"`
	SectionAnchor string `json:"section_anchor,omitempty"`
}

type ExportCard struct {
	StoryboardCard
	EwriteLink *ExportEwriteLink `json:"ewrite_link,omitempty"`
}

type ExportDocument struct {
	Format           string                    `json:"format"`
	FormatVersion    int                       `json:"format_version"`
	ExportedAt       time.Time                 `json:"exported_at"`
	Board            ExportBoard               `json:"board"`
	VisibilityGrants []ExportGrant             `json:"visibility_grants"`
	Columns          []StoryboardColumn        `json:"columns"`
	Bands            []StoryboardBand          `json:"bands"`
	Rows             []StoryboardRow           `json:"rows"`
	Cards            []ExportCard              `json:"cards"`
	ReferenceFields  []ReferenceFieldWithItems `json:"reference_fields"`
	IntegritySHA256  string                    `json:"integrity_sha256"`
}

// BuildBoardExport requires CanExportBoard (owner or Director+, spec
// 5.3-5.5 -- Crew never exports). Hidden cards are included only when the
// exporter's own tier authorizes seeing them -- reuses
// projectBoardSnapshotForTier rather than reimplementing the filter.
func BuildBoardExport(ctx context.Context, pool *pgxpool.Pool, userID, boardID string) (*ExportDocument, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return nil, err
	}
	if !tierAtLeastDirector(tier) {
		return nil, ErrNotAuthorized
	}

	snap, err := projectBoardSnapshotForTier(ctx, pool, tier, board)
	if err != nil {
		return nil, err
	}

	grantRows, err := listGrantsRaw(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	grants := make([]ExportGrant, 0, len(grantRows))
	for _, g := range grantRows {
		grants = append(grants, ExportGrant{
			UserID: g.UserID, UserHandle: g.UserHandle, GrantedRole: g.GrantedRole, GrantedAt: g.CreatedAt,
		})
	}

	cardIDs := make([]string, 0, len(snap.Cards))
	for _, c := range snap.Cards {
		cardIDs = append(cardIDs, c.ID)
	}
	links, err := storyboardCardEwriteLinks(ctx, pool, cardIDs)
	if err != nil {
		return nil, err
	}
	cards := make([]ExportCard, 0, len(snap.Cards))
	for _, c := range snap.Cards {
		ec := ExportCard{StoryboardCard: c}
		if link, ok := links[c.ID]; ok {
			ec.EwriteLink = &link
		}
		cards = append(cards, ec)
	}

	doc := &ExportDocument{
		Format:        ExportFormat,
		FormatVersion: ExportFormatVersion,
		ExportedAt:    time.Now().UTC(),
		Board: ExportBoard{
			ID: board.ID, Title: board.Title, Description: board.Description,
			OwnerUserID: board.OwnerUserID, OwnerHandle: board.OwnerHandle,
			Mode: board.Mode, TemplateVersion: board.TemplateVersion,
			CreatedAt: board.CreatedAt, UpdatedAt: board.UpdatedAt, ArchivedAt: board.ArchivedAt,
		},
		VisibilityGrants: grants,
		Columns:          snap.Columns,
		Bands:            snap.Bands,
		Rows:             snap.Rows,
		Cards:            cards,
		ReferenceFields:  snap.ReferenceFields,
	}

	canonical, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	doc.IntegritySHA256 = hex.EncodeToString(sum[:])
	return doc, nil
}

// storyboardCardEwriteLinks resolves the object_type='storyboard_card' rows
// added by migration 091, batched in one query -- same shape as
// ewrite.RuleLinksForEquipmentItems.
func storyboardCardEwriteLinks(ctx context.Context, pool *pgxpool.Pool, cardIDs []string) (map[string]ExportEwriteLink, error) {
	out := map[string]ExportEwriteLink{}
	if len(cardIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.storyboard_card_id::text, ol.publication_id::text, COALESCE(s.anchor, '')
		FROM ewrite_object_links ol
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'storyboard_card' AND ol.storyboard_card_id = ANY($1)
	`, cardIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cardID string
		var link ExportEwriteLink
		if err := rows.Scan(&cardID, &link.PublicationID, &link.SectionAnchor); err != nil {
			return nil, err
		}
		out[cardID] = link
	}
	return out, rows.Err()
}

// HandleBoardExport serves GET /api/storyboards/{board_id}/export.
func HandleBoardExport(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		doc, err := BuildBoardExport(ctx, pool, userID, boardID)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="storyboard-`+boardID+`.json"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doc)
	}
}
