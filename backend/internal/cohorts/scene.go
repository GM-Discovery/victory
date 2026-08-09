package cohorts

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/scenes"
)

// ActivateSceneForCohort makes the given Show Scene Placement this cohort's
// current Scene -- mirroring shows.SetCurrentScenePlacement's own
// validation (placement belongs to this exact Show, not archived/retired)
// but writing to show_cohorts instead of shows, so only this cohort's
// members are affected (kernel-85 S5.3: "leave other cohort Scene
// projections untouched").
func ActivateSceneForCohort(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, cohortID, placementID string) (Cohort, error) {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return Cohort{}, err
	}
	c, err := LoadCohortByID(ctx, pool, cohortID)
	if err != nil {
		return Cohort{}, err
	}
	if c.ShowID != showID {
		return Cohort{}, errors.New("cohort_show_mismatch")
	}
	if c.ArchivedAt != nil {
		return Cohort{}, errors.New("cohort_archived")
	}

	placementID = strings.TrimSpace(placementID)
	if placementID == "" {
		return Cohort{}, errors.New("placement_id_required")
	}
	p, err := scenes.LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return Cohort{}, err
	}
	if p.ShowID != showID {
		return Cohort{}, errors.New("placement_show_mismatch")
	}
	if p.ArchivedAt != nil || p.Status == "archived" || p.Status == "retired" {
		return Cohort{}, errors.New("placement_not_eligible")
	}

	row := pool.QueryRow(ctx, `
		UPDATE show_cohorts SET current_show_scene_placement_id = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING `+cohortColumns, cohortID, placementID)
	return scanCohort(row)
}

// ClearSceneForCohort explicitly clears a cohort's current-Scene pointer,
// returning its members to whatever the Show's own global current Scene is
// (the pre-Kernel-85 fallback path in world.LoadVenueSnapshot).
func ClearSceneForCohort(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, cohortID string) (Cohort, error) {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return Cohort{}, err
	}
	c, err := LoadCohortByID(ctx, pool, cohortID)
	if err != nil {
		return Cohort{}, err
	}
	if c.ShowID != showID {
		return Cohort{}, errors.New("cohort_show_mismatch")
	}
	row := pool.QueryRow(ctx, `
		UPDATE show_cohorts SET current_show_scene_placement_id = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING `+cohortColumns, cohortID)
	return scanCohort(row)
}

// ResolveCurrentPlacementForViewer answers "does this viewer belong to a
// cohort with its own current Scene right now" -- the one read
// world.LoadVenueSnapshot needs to decide whether to substitute the
// viewer's cohort Scene for the Show's shared one. Returns "" for both
// values when the viewer is Ungrouped or their cohort has no Scene set yet
// (both fall through to the unchanged Show-global path, kernel-85 S5.6).
func ResolveCurrentPlacementForViewer(ctx context.Context, pool *pgxpool.Pool, showID, userID string) (placementID, cohortID string, err error) {
	showID = strings.TrimSpace(showID)
	userID = strings.TrimSpace(userID)
	if showID == "" || userID == "" {
		return "", "", nil
	}
	var placement *string
	var cid string
	err = pool.QueryRow(ctx, `
		SELECT c.current_show_scene_placement_id::text, c.id::text
		FROM show_cohort_assignments a
		JOIN show_cohorts c ON c.id = a.cohort_id
		WHERE a.show_id = $1 AND a.user_id = $2 AND c.archived_at IS NULL
	`, showID, userID).Scan(&placement, &cid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil
		}
		return "", "", err
	}
	if placement == nil {
		return "", cid, nil
	}
	return *placement, cid, nil
}
