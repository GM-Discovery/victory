package thirdplace

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/playerprofile"
	"victory/backend/internal/playerrelationships"
)

const headshotColumns = `id::text, user_id::text, status, placed_at, removed_at, created_at, updated_at`

func scanHeadshot(row pgx.Row) (Headshot, error) {
	var h Headshot
	if err := row.Scan(&h.ID, &h.UserID, &h.Status, &h.PlacedAt, &h.RemovedAt, &h.CreatedAt, &h.UpdatedAt); err != nil {
		return Headshot{}, err
	}
	return h, nil
}

// LeaveHeadshot creates the caller's one active Headshot, or refreshes the
// existing one if they already have one (Kernel 65 §3.4). The upsert
// targets the partial unique index directly (`ON CONFLICT (user_id) WHERE
// removed_at IS NULL AND status = 'active'`), so "at most one active
// Headshot per account" holds even under concurrent requests, not just by
// application-level convention. `(xmax = 0)` is the standard Postgres way to
// tell an INSERT from an ON-CONFLICT UPDATE in the same RETURNING clause,
// which is how `created` is determined without a second round trip.
func LeaveHeadshot(ctx context.Context, pool *pgxpool.Pool, userID string) (Headshot, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Headshot{}, false, errors.New("not_authenticated")
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO third_place_headshots (user_id, status, placed_at, updated_at)
		VALUES ($1::uuid, 'active', NOW(), NOW())
		ON CONFLICT (user_id) WHERE removed_at IS NULL AND status = 'active'
		DO UPDATE SET updated_at = NOW()
		RETURNING `+headshotColumns+`, (xmax = 0) AS inserted
	`, userID)

	var h Headshot
	var inserted bool
	if err := row.Scan(&h.ID, &h.UserID, &h.Status, &h.PlacedAt, &h.RemovedAt, &h.CreatedAt, &h.UpdatedAt, &inserted); err != nil {
		return Headshot{}, false, err
	}
	return h, inserted, nil
}

// RemoveHeadshot closes the caller's active Headshot (if any), preserving it
// as a history row rather than deleting it (Kernel 65 §3.5, §5.2). Safe to
// call with no active Headshot -- affects zero rows, returns no error
// (Kernel 65 §10.2 "repeated DELETE is safe/idempotent").
func RemoveHeadshot(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("not_authenticated")
	}
	_, err := pool.Exec(ctx, `
		UPDATE third_place_headshots
		SET status = 'removed', removed_at = NOW(), updated_at = NOW()
		WHERE user_id = $1::uuid AND removed_at IS NULL AND status = 'active'
	`, userID)
	return err
}

// GetMyHeadshot returns the caller's active Headshot, or (nil, nil) if they
// don't have one -- not an error, since "no active Headshot" is an ordinary
// state (Kernel 65 §8.3).
func GetMyHeadshot(ctx context.Context, pool *pgxpool.Pool, userID string) (*Headshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("not_authenticated")
	}
	row := pool.QueryRow(ctx, `
		SELECT `+headshotColumns+`
		FROM third_place_headshots
		WHERE user_id = $1::uuid AND removed_at IS NULL AND status = 'active'
	`, userID)
	h, err := scanHeadshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}

// ListActiveHeadshots returns every active Headshot, most recently placed
// first (Kernel 65 §6.2's "recently placed" default). Removed Headshots are
// never included (Kernel 65 §9.5). Search/sort-by-name are left to the
// frontend to apply client-side over this list, mirroring the existing
// My People list page's convention rather than adding server-side filter
// params for a dataset this size.
func ListActiveHeadshots(ctx context.Context, pool *pgxpool.Pool) ([]Headshot, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+headshotColumns+`
		FROM third_place_headshots
		WHERE removed_at IS NULL AND status = 'active'
		ORDER BY placed_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Headshot
	for rows.Next() {
		h, err := scanHeadshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ListMyHeadshotHistory returns every placement/removal row for the caller,
// most recent first -- their own Headshot ledger, never anyone else's
// (Kernel 65 §6.1, §9.5).
func ListMyHeadshotHistory(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Headshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("not_authenticated")
	}
	rows, err := pool.Query(ctx, `
		SELECT `+headshotColumns+`
		FROM third_place_headshots
		WHERE user_id = $1::uuid
		ORDER BY placed_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Headshot
	for rows.Next() {
		h, err := scanHeadshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ProjectHeadshot builds the live, viewer-scoped public projection for one
// Headshot row (Kernel 65 §5.3). It always re-derives stage name, portrait,
// and headline facts from the owner's *current* Trailer Face -- there is no
// stored Face snapshot anywhere to go stale (Kernel 65 §3.3). Relationship
// state is computed fresh per viewer and never reveals anyone else's
// relationship with the owner (Kernel 65 §9.4).
func ProjectHeadshot(ctx context.Context, pool *pgxpool.Pool, viewerUserID string, h Headshot) (HeadshotProjection, error) {
	wb, err := playerprofile.EnsureWorkbook(ctx, pool, h.UserID)
	if err != nil {
		return HeadshotProjection{}, err
	}

	face, err := playerprofile.ProjectTrailerFace(ctx, pool, h.UserID)
	if err != nil {
		return HeadshotProjection{}, err
	}

	proj := HeadshotProjection{
		HeadshotID:                 h.ID,
		ProfileID:                  wb.ID,
		PlacedAt:                   h.PlacedAt,
		StageName:                  face.StageName,
		HeadlineFacts:              []HeadlineFact{},
		TrailerURL:                 "/venues/trailers/view.html?id=" + wb.ID,
		RelationshipStateForViewer: "none",
		IsYou:                      strings.TrimSpace(viewerUserID) == h.UserID,
	}

	for _, f := range face.Regions["identity_header"] {
		if f.FieldKey == "portrait_url" {
			proj.PortraitURL = f.DisplayValue
			break
		}
	}

	glance := face.Regions["at_a_glance"]
	limit := 3
	if len(glance) < limit {
		limit = len(glance)
	}
	for _, f := range glance[:limit] {
		proj.HeadlineFacts = append(proj.HeadlineFacts, HeadlineFact{Label: f.Label, Value: f.DisplayValue})
	}

	if !proj.IsYou {
		rel, err := playerrelationships.GetRelationshipBySubjectProfile(ctx, pool, viewerUserID, wb.ID)
		if err != nil {
			if err.Error() != "relationship_not_found" {
				return HeadshotProjection{}, err
			}
			proj.CanAddToMyPeople = true
		} else {
			proj.RelationshipStateForViewer = "exists"
			proj.CanOpenMyNotes = true
			proj.RelationshipID = rel.ID
		}
	}

	return proj, nil
}
