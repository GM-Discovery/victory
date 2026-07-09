package playerprofile

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OwnerWorkbookView is everything the authenticated owner sees: source
// facts, ordinary event history, stage-name history, and Face preview
// (Kernel 61 §9.7). It never leaves the owner-only surface.
type OwnerWorkbookView struct {
	Workbook          Workbook               `json:"workbook"`
	StageName         string                 `json:"stage_name"`
	Facts             map[string]ProfileFact `json:"facts"`
	Events            []ProfileEvent         `json:"events"`
	StageNameHistory  []StageNameEntry       `json:"stage_name_history"`
	FacePreview       TrailerFace            `json:"face_preview"`
	EligibleFields    []ProjectedField       `json:"eligible_fields"`
	ProjectionVersion string                 `json:"projection_version"`
}

// ProjectionVersion is a durable, comparable server version for a user's
// Player Workbook / Trailer Face, derived as GREATEST() over every table
// that composes the projection -- the same no-separate-counter-table
// pattern as the Kernel 59A character projector (Kernel 61 §6.7).
func ProjectionVersion(ctx context.Context, pool *pgxpool.Pool, workbookID string) (string, error) {
	var version time.Time
	if err := pool.QueryRow(ctx, `
		SELECT GREATEST(
			w.updated_at,
			COALESCE((SELECT MAX(e.created_at) FROM player_profile_events e WHERE e.workbook_id = w.id), w.updated_at),
			COALESCE((SELECT MAX(f.effective_at) FROM player_profile_facts f WHERE f.workbook_id = w.id), w.updated_at),
			COALESCE((SELECT MAX(o.updated_at) FROM player_profile_face_overrides o WHERE o.workbook_id = w.id), w.updated_at),
			COALESCE((SELECT MAX(s.started_at) FROM player_stage_name_history s WHERE s.user_id = w.user_id), w.updated_at)
		)
		FROM player_profile_workbooks w
		WHERE w.id = $1
	`, workbookID).Scan(&version); err != nil {
		return "", err
	}
	return version.UTC().Format(time.RFC3339Nano), nil
}

func loadCurrentFacts(ctx context.Context, pool *pgxpool.Pool, workbookID string) (map[string]ProfileFact, error) {
	rows, err := pool.Query(ctx, `
		SELECT field_key, value_json, display_value, source_event_id::text, source_page_key, catalogue_version, effective_at
		FROM player_profile_facts
		WHERE workbook_id = $1
	`, workbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]ProfileFact{}
	for rows.Next() {
		var f ProfileFact
		if err := rows.Scan(&f.FieldKey, &f.ValueJSON, &f.DisplayValue, &f.SourceEventID, &f.SourcePageKey, &f.CatalogueVersion, &f.EffectiveAt); err != nil {
			return nil, err
		}
		out[f.FieldKey] = f
	}
	return out, rows.Err()
}

// ProjectOwnerWorkbook builds the full owner-only projection (Kernel 61
// §9.7). Only the authenticated owner may receive this -- callers must
// resolve userID from the session, never trust a client-supplied value.
func ProjectOwnerWorkbook(ctx context.Context, pool *pgxpool.Pool, userID string) (OwnerWorkbookView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return OwnerWorkbookView{}, errors.New("not_authenticated")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return OwnerWorkbookView{}, err
	}

	facts, err := loadCurrentFacts(ctx, pool, wb.ID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}

	events, err := loadWorkbookEvents(ctx, pool, wb.ID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}
	eventsDesc := append([]ProfileEvent(nil), events...)
	for i, j := 0, len(eventsDesc)-1; i < j; i, j = i+1, j-1 {
		eventsDesc[i], eventsDesc[j] = eventsDesc[j], eventsDesc[i]
	}

	stageHistory, err := LoadStageNameHistory(ctx, pool, userID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}
	stageName, _, err := CurrentStageName(ctx, pool, userID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}

	overrides, err := loadFaceOverrides(ctx, pool, wb.ID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}
	projected := BuildProjectedFields(cat, facts, overrides)

	// Unlike the Face preview (visible-only), the Face compiler needs every
	// eligible fact -- including currently-hidden ones -- so the owner can
	// find and re-show them. Sorted by region then priority so the compiler
	// UI can group consistently with how Face itself groups.
	eligible := append([]ProjectedField(nil), projected...)
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Region != eligible[j].Region {
			return eligible[i].Region < eligible[j].Region
		}
		if eligible[i].PriorityScore != eligible[j].PriorityScore {
			return eligible[i].PriorityScore > eligible[j].PriorityScore
		}
		return eligible[i].Label < eligible[j].Label
	})

	version, err := ProjectionVersion(ctx, pool, wb.ID)
	if err != nil {
		return OwnerWorkbookView{}, err
	}

	return OwnerWorkbookView{
		Workbook:          wb,
		StageName:         stageName,
		Facts:             facts,
		Events:            eventsDesc,
		StageNameHistory:  stageHistory,
		FacePreview:       BuildTrailerFace(version, stageName, projected),
		EligibleFields:    eligible,
		ProjectionVersion: version,
	}, nil
}

// ProjectTrailerFace builds the compiled social projection for any
// authenticated requester (Kernel 61 §8.2). It contains only stage name and
// Face-visible facts -- no email, handle, account UUID, source pages, event
// history, or stage-name ledger (AC-19).
func ProjectTrailerFace(ctx context.Context, pool *pgxpool.Pool, profileUserID string) (TrailerFace, error) {
	profileUserID = strings.TrimSpace(profileUserID)
	if profileUserID == "" {
		return TrailerFace{}, errors.New("user_id_required")
	}

	wb, err := EnsureWorkbook(ctx, pool, profileUserID)
	if err != nil {
		return TrailerFace{}, err
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return TrailerFace{}, err
	}

	facts, err := loadCurrentFacts(ctx, pool, wb.ID)
	if err != nil {
		return TrailerFace{}, err
	}
	overrides, err := loadFaceOverrides(ctx, pool, wb.ID)
	if err != nil {
		return TrailerFace{}, err
	}
	projected := BuildProjectedFields(cat, facts, overrides)

	stageName, _, err := CurrentStageName(ctx, pool, profileUserID)
	if err != nil {
		return TrailerFace{}, err
	}

	version, err := ProjectionVersion(ctx, pool, wb.ID)
	if err != nil {
		return TrailerFace{}, err
	}

	return BuildTrailerFace(version, stageName, projected), nil
}
