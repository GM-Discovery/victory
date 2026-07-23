package shows

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// placementOwnership is the minimal shape needed to validate a Show Scene
// Placement before it becomes a Show's current pointer. This package
// deliberately does not import backend/internal/scenes for this lookup --
// scenes already imports shows (scenes/placements.go's showRunForShow), so
// importing scenes back here would be a cycle. A narrow duplicated SQL
// query against show_scene_placements is the smaller, safer fix, matching
// this codebase's existing precedent of packages writing/reading a table
// they don't "own" for a narrow, well-understood side effect (e.g.
// showings writing to actions.showing_id).
type placementOwnership struct {
	ShowID     string
	Status     string
	ArchivedAt *string
}

func loadPlacementOwnership(ctx context.Context, pool *pgxpool.Pool, placementID string) (placementOwnership, error) {
	var p placementOwnership
	var archivedAt *string
	err := pool.QueryRow(ctx, `
		SELECT show_id::text, status, archived_at::text
		FROM show_scene_placements
		WHERE id = $1
	`, placementID).Scan(&p.ShowID, &p.Status, &archivedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return placementOwnership{}, errors.New("placement_not_found")
		}
		return placementOwnership{}, err
	}
	p.ArchivedAt = archivedAt
	return p, nil
}

// SetCurrentScenePlacement makes the given Show Scene Placement the Show's
// persistent current Scene (Kernel 70 SS4.1). Only Director/Producer/
// Operator authority may advance the Show's stage this way directly;
// Crew's non-destructive GO right reaches this indirectly through a Cue's
// go_to_scene action (backend/internal/cues), which performs its own
// authority check before calling this function.
func SetCurrentScenePlacement(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, placementID string) (Show, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Show{}, errors.New("not_authenticated")
	}
	placementID = strings.TrimSpace(placementID)
	if placementID == "" {
		return Show{}, errors.New("placement_id_required")
	}

	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return Show{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return Show{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Show{}, err
	}
	if !ok {
		return Show{}, errors.New("not_authorized")
	}

	if err := validatePlacementBelongsToShow(ctx, pool, showID, placementID); err != nil {
		return Show{}, err
	}

	row := pool.QueryRow(ctx, `
		UPDATE shows SET current_show_scene_placement_id = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING `+showColumns, showID, placementID)
	updated, err := scanShow(row)
	if err != nil {
		return Show{}, err
	}
	return updated, nil
}

// SetCurrentScenePlacementTrusted is SetCurrentScenePlacement without the
// Director/Producer/Operator authority re-check -- used exclusively by
// cues.ExecuteCue's go_to_scene action, whose own cues.CanTriggerCue check
// has already authorized the whole GO press (including for Crew, who are
// explicitly permitted to press GO and thereby advance the current Scene,
// Kernel 70 SS1.8/SS6.2). Placement/Show ownership is still validated.
func SetCurrentScenePlacementTrusted(ctx context.Context, pool *pgxpool.Pool, showID, placementID string) (Show, error) {
	placementID = strings.TrimSpace(placementID)
	if placementID == "" {
		return Show{}, errors.New("placement_id_required")
	}
	if err := validatePlacementBelongsToShow(ctx, pool, showID, placementID); err != nil {
		return Show{}, err
	}
	row := pool.QueryRow(ctx, `
		UPDATE shows SET current_show_scene_placement_id = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING `+showColumns, showID, placementID)
	updated, err := scanShow(row)
	if err != nil {
		return Show{}, err
	}
	return updated, nil
}

// validatePlacementBelongsToShow enforces the kernel's pointer-safety
// rules: the placement must exist, belong to this exact Show, and not be
// archived or retired.
func validatePlacementBelongsToShow(ctx context.Context, pool *pgxpool.Pool, showID, placementID string) error {
	p, err := loadPlacementOwnership(ctx, pool, placementID)
	if err != nil {
		return err
	}
	if p.ShowID != showID {
		return errors.New("placement_show_mismatch")
	}
	if p.ArchivedAt != nil || p.Status == "archived" || p.Status == "retired" {
		return errors.New("placement_not_eligible")
	}
	return nil
}

// ClearCurrentScenePlacement explicitly clears the Show's current-Scene
// pointer. Director/Producer/Operator authority only.
func ClearCurrentScenePlacement(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (Show, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Show{}, errors.New("not_authenticated")
	}
	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return Show{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return Show{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Show{}, err
	}
	if !ok {
		return Show{}, errors.New("not_authorized")
	}

	row := pool.QueryRow(ctx, `
		UPDATE shows SET current_show_scene_placement_id = NULL, updated_at = NOW()
		WHERE id = $1
		RETURNING `+showColumns, showID)
	return scanShow(row)
}

// SetShowVariableTrusted writes a single Show variable into the
// materialized variables_json cache (Kernel 70 SS4.2). Called by
// cues.ExecuteCue's set_show_variable action, which separately writes the
// canonical show_id-scoped actions log row -- this is a denormalized
// read-shortcut, not the source of truth. No authority check here -- the
// caller's Cue-trigger authority already gated the whole execution.
func SetShowVariableTrusted(ctx context.Context, pool *pgxpool.Pool, showID, key string, value []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("variable_key_required")
	}
	_, err := pool.Exec(ctx, `
		UPDATE shows
		SET variables_json = jsonb_set(COALESCE(variables_json, '{}'::jsonb), $2::text[], $3::jsonb, TRUE), updated_at = NOW()
		WHERE id = $1
	`, showID, []string{key}, string(value))
	return err
}
