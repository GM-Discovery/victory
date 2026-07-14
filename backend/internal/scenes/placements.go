package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

const placementColumns = `
	id::text, show_id::text, scene_id::text, venue_id::text, sort_order, status,
	COALESCE(audience_title_override, ''), COALESCE(audience_summary_override, ''),
	COALESCE(director_notes_override, ''), config_json::text, created_by_user_id::text,
	created_at, updated_at, archived_at
`

func scanPlacement(row pgx.Row) (ShowScenePlacement, error) {
	var p ShowScenePlacement
	var venueID, createdByUserID *string
	var configText string
	if err := row.Scan(
		&p.ID, &p.ShowID, &p.SceneID, &venueID, &p.SortOrder, &p.Status,
		&p.AudienceTitleOverride, &p.AudienceSummaryOverride, &p.DirectorNotesOverride,
		&configText, &createdByUserID, &p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShowScenePlacement{}, errors.New("placement_not_found")
		}
		return ShowScenePlacement{}, err
	}
	p.VenueID = venueID
	p.CreatedByUserID = createdByUserID
	if configText != "" {
		p.ConfigJSON = json.RawMessage(configText)
	}
	return p, nil
}

// showRunForShow loads a Show and its parent Show Run in one place, the
// same two-step lookup shows.CreateShow/UpdateShow already perform for
// their own authority checks.
func showRunForShow(ctx context.Context, pool *pgxpool.Pool, showID string) (shows.Show, showruns.ShowRun, error) {
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return shows.Show{}, showruns.ShowRun{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return shows.Show{}, showruns.ShowRun{}, err
	}
	return s, sr, nil
}

// CreatePlacement authority-checks the actor against the Show's parent Show
// Run location, validates the target Scene is reusable at the same
// location as the Show (a Scene may be staged in any Show at the same
// Victory location regardless of source Production, Kernel 70 SS3.1,
// correcting Kernel 69 SS1.4's Production-exclusive rule; cross-location
// placement remains rejected), and rejects staging an archived Scene.
func CreatePlacement(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string, in CreatePlacementInput) (ShowScenePlacement, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return ShowScenePlacement{}, errors.New("not_authenticated")
	}
	sceneID := strings.TrimSpace(in.SceneID)
	if sceneID == "" {
		return ShowScenePlacement{}, errors.New("scene_id_required")
	}

	_, sr, err := showRunForShow(ctx, pool, showID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	if !ok {
		return ShowScenePlacement{}, errors.New("not_authorized")
	}

	scene, err := LoadSceneByID(ctx, pool, sceneID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	if scene.LocationID != sr.LocationID {
		return ShowScenePlacement{}, errors.New("scene_location_mismatch")
	}
	if scene.Status == "archived" {
		return ShowScenePlacement{}, errors.New("scene_archived")
	}

	var venueID *string
	if v := strings.TrimSpace(in.VenueID); v != "" {
		venueID = &v
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (
			show_id, scene_id, venue_id, sort_order,
			audience_title_override, audience_summary_override, director_notes_override,
			created_by_user_id
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8)
		RETURNING `+placementColumns,
		showID, sceneID, venueID, in.SortOrder,
		in.AudienceTitleOverride, in.AudienceSummaryOverride, in.DirectorNotesOverride,
		actorUserID)
	placement, err := scanPlacement(row)
	if err != nil && strings.Contains(err.Error(), "show_scene_placements_show_id_scene_id_key") {
		return ShowScenePlacement{}, errors.New("scene_already_placed_in_show")
	}
	return placement, err
}

// LoadPlacementByID returns the raw row with no authority check.
func LoadPlacementByID(ctx context.Context, pool *pgxpool.Pool, placementID string) (ShowScenePlacement, error) {
	placementID = strings.TrimSpace(placementID)
	if placementID == "" {
		return ShowScenePlacement{}, errors.New("placement_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+placementColumns+` FROM show_scene_placements WHERE id = $1`, placementID)
	return scanPlacement(row)
}

// UpdatePlacement authority-checks against the placement's Show's parent
// Show Run location, then applies only the fields present in the patch.
// This never mutates the underlying Scene (Kernel 69 SS1.3, SS4.2).
func UpdatePlacement(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string, patch UpdatePlacementPatch) (ShowScenePlacement, error) {
	p, err := LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	_, sr, err := showRunForShow(ctx, pool, p.ShowID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return ShowScenePlacement{}, err
	}
	if !ok {
		return ShowScenePlacement{}, errors.New("not_authorized")
	}

	venueID := p.VenueID
	if patch.VenueID != nil {
		if strings.TrimSpace(*patch.VenueID) == "" {
			venueID = nil
		} else {
			venueID = patch.VenueID
		}
	}
	sortOrder := p.SortOrder
	if patch.SortOrder != nil {
		sortOrder = *patch.SortOrder
	}
	audienceTitleOverride := p.AudienceTitleOverride
	if patch.AudienceTitleOverride != nil {
		audienceTitleOverride = *patch.AudienceTitleOverride
	}
	audienceSummaryOverride := p.AudienceSummaryOverride
	if patch.AudienceSummaryOverride != nil {
		audienceSummaryOverride = *patch.AudienceSummaryOverride
	}
	directorNotesOverride := p.DirectorNotesOverride
	if patch.DirectorNotesOverride != nil {
		directorNotesOverride = *patch.DirectorNotesOverride
	}
	status := p.Status
	if patch.Status != nil {
		status = *patch.Status
	}

	archivedAt := "NULL"
	if status == "archived" {
		archivedAt = "NOW()"
	}

	row := pool.QueryRow(ctx, `
		UPDATE show_scene_placements
		SET venue_id = $2, sort_order = $3, status = $4,
		    audience_title_override = NULLIF($5, ''), audience_summary_override = NULLIF($6, ''),
		    director_notes_override = NULLIF($7, ''), archived_at = `+archivedAt+`, updated_at = NOW()
		WHERE id = $1
		RETURNING `+placementColumns,
		placementID, venueID, sortOrder, status,
		audienceTitleOverride, audienceSummaryOverride, directorNotesOverride)
	updated, err := scanPlacement(row)
	if err != nil {
		return ShowScenePlacement{}, err
	}

	// A Show's persistent current-Scene pointer (Kernel 70 SS4.1) must
	// never point at an archived or retired placement. This is plain SQL
	// against the shows table rather than a call into
	// backend/internal/shows, avoiding a package import cycle (shows
	// already imports showruns, and scenes already imports shows) for a
	// narrow, well-understood side effect -- the same pattern showings
	// already uses when it writes to actions.showing_id.
	if updated.Status == "archived" || updated.Status == "retired" {
		if _, err := pool.Exec(ctx, `
			UPDATE shows SET current_show_scene_placement_id = NULL, updated_at = NOW()
			WHERE id = $1 AND current_show_scene_placement_id = $2
		`, updated.ShowID, updated.ID); err != nil {
			return ShowScenePlacement{}, err
		}
	}
	return updated, nil
}

// ArchivePlacement ("Remove Scene from Show" in the UI) is a thin wrapper
// over UpdatePlacement's status field. It never archives or otherwise
// touches the underlying reusable Scene.
func ArchivePlacement(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string) (ShowScenePlacement, error) {
	status := "archived"
	return UpdatePlacement(ctx, pool, actorUserID, placementID, UpdatePlacementPatch{Status: &status})
}

// ListPlacementsForShow returns every Show Scene Placement for one Show,
// ordered by sort_order, joined with a curated projection of each
// placement's base Scene. Callers must already have passed a Scenes
// backstage-visibility check for this Show before calling.
func ListPlacementsForShow(ctx context.Context, pool *pgxpool.Pool, showID string) ([]PlacementDetail, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id::text, p.show_id::text, p.scene_id::text, p.venue_id::text, p.sort_order, p.status,
		       COALESCE(p.audience_title_override, ''), COALESCE(p.audience_summary_override, ''),
		       COALESCE(p.director_notes_override, ''), p.config_json::text, p.created_by_user_id::text,
		       p.created_at, p.updated_at, p.archived_at,
		       s.id::text, s.slug, s.title, COALESCE(s.short_title, ''), s.default_venue_id::text,
		       COALESCE(s.audience_title, ''), COALESCE(s.audience_summary, ''),
		       COALESCE(s.player_brief, ''), COALESCE(s.director_notes, ''),
		       COALESCE(s.source_ref, ''), s.status
		FROM show_scene_placements p
		JOIN scenes s ON s.id = p.scene_id
		WHERE p.show_id = $1
		ORDER BY p.sort_order ASC, p.created_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlacementDetail
	for rows.Next() {
		var p ShowScenePlacement
		var venueID, createdByUserID, sceneDefaultVenueID *string
		var configText string
		var base ScenePlacementBase
		if err := rows.Scan(
			&p.ID, &p.ShowID, &p.SceneID, &venueID, &p.SortOrder, &p.Status,
			&p.AudienceTitleOverride, &p.AudienceSummaryOverride, &p.DirectorNotesOverride,
			&configText, &createdByUserID, &p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt,
			&base.ID, &base.Slug, &base.Title, &base.ShortTitle, &sceneDefaultVenueID,
			&base.AudienceTitle, &base.AudienceSummary, &base.PlayerBrief, &base.DirectorNotes,
			&base.SourceRef, &base.Status,
		); err != nil {
			return nil, err
		}
		p.VenueID = venueID
		p.CreatedByUserID = createdByUserID
		if configText != "" {
			p.ConfigJSON = json.RawMessage(configText)
		}
		base.DefaultVenueID = sceneDefaultVenueID
		out = append(out, PlacementDetail{Placement: p, Scene: base})
	}
	return out, rows.Err()
}

// ListAudiencePlacementsForShow returns only ready, non-archived Show Scene
// Placements for a Show, projected to the curated Audience Program shape.
// Backstage-only fields (director_notes, operator_notes, source_ref,
// config_json, created_by_user_id, internal ids) never appear here (Kernel
// 69 SS2.6).
func ListAudiencePlacementsForShow(ctx context.Context, pool *pgxpool.Pool, showID string) ([]AudienceScenePlacement, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  COALESCE(NULLIF(p.audience_title_override, ''), NULLIF(s.audience_title, ''), s.title),
		  COALESCE(NULLIF(p.audience_summary_override, ''), COALESCE(s.audience_summary, '')),
		  p.sort_order
		FROM show_scene_placements p
		JOIN scenes s ON s.id = p.scene_id
		WHERE p.show_id = $1 AND p.status = 'ready' AND p.archived_at IS NULL
		ORDER BY p.sort_order ASC, p.created_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AudienceScenePlacement
	for rows.Next() {
		var a AudienceScenePlacement
		if err := rows.Scan(&a.Title, &a.AudienceSummary, &a.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
