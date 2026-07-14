package cues

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

const cueColumns = `
	id::text, show_scene_placement_id::text, internal_name, COALESCE(stage_button_label, ''),
	trigger_scope, sort_order, enabled, actions::text, created_by_user_id::text,
	created_at, updated_at
`

func scanCue(row pgx.Row) (Cue, error) {
	var c Cue
	var createdByUserID *string
	var actionsText string
	if err := row.Scan(
		&c.ID, &c.ShowScenePlacementID, &c.InternalName, &c.StageButtonLabel,
		&c.TriggerScope, &c.SortOrder, &c.Enabled, &actionsText, &createdByUserID,
		&c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cue{}, errors.New("cue_not_found")
		}
		return Cue{}, err
	}
	c.CreatedByUserID = createdByUserID
	if actionsText != "" {
		c.Actions = json.RawMessage(actionsText)
	}
	return c, nil
}

// placementShowShowRunLocation resolves a Show Scene Placement's parent
// Show id, Show Run id, and Show Run location -- the same walk
// scenes/placements.go's showRunForShow already performs, extended with
// the Show id itself since Cues need it for cue_executions.show_id and
// for scoping go_to_scene/set_show_variable to the right Show.
func placementShowShowRunLocation(ctx context.Context, pool *pgxpool.Pool, placementID string) (showID, showRunID, locationID string, err error) {
	p, err := scenes.LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return "", "", "", err
	}
	s, err := shows.LoadShowByID(ctx, pool, p.ShowID)
	if err != nil {
		return "", "", "", err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return "", "", "", err
	}
	return p.ShowID, sr.ID, sr.LocationID, nil
}

// validateCueActions checks structural shape server-side: each action's
// Type must be one of the three implemented types, with its corresponding
// typed field present. Show-membership/eligibility of a go_to_scene target
// is deliberately NOT checked here -- that can change between Cue creation
// and firing, so it's re-validated fresh at execution time instead.
func validateCueActions(list []CueAction) error {
	for _, a := range list {
		switch a.Type {
		case ActionTypeGoToScene:
			if a.GoToScene == nil || strings.TrimSpace(a.GoToScene.ShowScenePlacementID) == "" {
				return errors.New("go_to_scene_target_required")
			}
		case ActionTypeEmitGameEvent:
			if a.EmitGameEvent == nil || strings.TrimSpace(a.EmitGameEvent.EventKind) == "" {
				return errors.New("emit_game_event_kind_required")
			}
		case ActionTypeSetShowVariable:
			if a.SetShowVariable == nil || strings.TrimSpace(a.SetShowVariable.Key) == "" {
				return errors.New("set_show_variable_key_required")
			}
		default:
			return errors.New("unknown_cue_action_type")
		}
	}
	return nil
}

// CreateCue authority-checks the actor against the placement's Show Run
// location via CanCrewPerformNonDestructiveEdit -- Crew may create
// non-destructive Cues (Kernel 70 §1.8).
func CreateCue(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string, in CreateCueInput) (Cue, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Cue{}, errors.New("not_authenticated")
	}
	if strings.TrimSpace(in.InternalName) == "" {
		return Cue{}, errors.New("internal_name_required")
	}

	triggerScope := strings.TrimSpace(in.TriggerScope)
	if triggerScope == "" {
		triggerScope = TriggerScopeDirectorCrewOnly
	}
	if !isValidTriggerScope(triggerScope) {
		return Cue{}, errors.New("invalid_trigger_scope")
	}
	if err := validateCueActions(in.Actions); err != nil {
		return Cue{}, err
	}

	_, _, locationID, err := placementShowShowRunLocation(ctx, pool, placementID)
	if err != nil {
		return Cue{}, err
	}
	ok, err := showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
	if err != nil {
		return Cue{}, err
	}
	if !ok {
		return Cue{}, errors.New("not_authorized")
	}

	actionsJSON, err := json.Marshal(in.Actions)
	if err != nil {
		return Cue{}, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	var stageButtonLabel *string
	if v := strings.TrimSpace(in.StageButtonLabel); v != "" {
		stageButtonLabel = &v
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO cues (
			show_scene_placement_id, internal_name, stage_button_label,
			trigger_scope, sort_order, enabled, actions, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+cueColumns,
		placementID, in.InternalName, stageButtonLabel, triggerScope, in.SortOrder, enabled, actionsJSON, actorUserID)
	return scanCue(row)
}

// LoadCueByID returns the raw row with no authority check -- callers that
// expose this to a viewer must authority-check separately.
func LoadCueByID(ctx context.Context, pool *pgxpool.Pool, cueID string) (Cue, error) {
	cueID = strings.TrimSpace(cueID)
	if cueID == "" {
		return Cue{}, errors.New("cue_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+cueColumns+` FROM cues WHERE id = $1`, cueID)
	return scanCue(row)
}

// UpdateCue authority-checks against the Cue's placement's Show Run
// location via CanCrewPerformNonDestructiveEdit, then applies only the
// fields present in the patch.
func UpdateCue(ctx context.Context, pool *pgxpool.Pool, actorUserID, cueID string, patch UpdateCuePatch) (Cue, error) {
	c, err := LoadCueByID(ctx, pool, cueID)
	if err != nil {
		return Cue{}, err
	}
	_, _, locationID, err := placementShowShowRunLocation(ctx, pool, c.ShowScenePlacementID)
	if err != nil {
		return Cue{}, err
	}
	ok, err := showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
	if err != nil {
		return Cue{}, err
	}
	if !ok {
		return Cue{}, errors.New("not_authorized")
	}

	internalName := c.InternalName
	if patch.InternalName != nil {
		internalName = *patch.InternalName
	}
	stageButtonLabel := c.StageButtonLabel
	if patch.StageButtonLabel != nil {
		stageButtonLabel = *patch.StageButtonLabel
	}
	triggerScope := c.TriggerScope
	if patch.TriggerScope != nil {
		triggerScope = strings.TrimSpace(*patch.TriggerScope)
		if !isValidTriggerScope(triggerScope) {
			return Cue{}, errors.New("invalid_trigger_scope")
		}
	}
	sortOrder := c.SortOrder
	if patch.SortOrder != nil {
		sortOrder = *patch.SortOrder
	}
	enabled := c.Enabled
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}

	actionsJSON := []byte(c.Actions)
	if patch.Actions != nil {
		if err := validateCueActions(*patch.Actions); err != nil {
			return Cue{}, err
		}
		marshaled, err := json.Marshal(*patch.Actions)
		if err != nil {
			return Cue{}, err
		}
		actionsJSON = marshaled
	}

	row := pool.QueryRow(ctx, `
		UPDATE cues
		SET internal_name = $2, stage_button_label = NULLIF($3, ''), trigger_scope = $4,
		    sort_order = $5, enabled = $6, actions = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING `+cueColumns,
		cueID, internalName, stageButtonLabel, triggerScope, sortOrder, enabled, actionsJSON)
	return scanCue(row)
}

// ListTriggerableCuesForViewer returns the curated player-facing stage
// button list (Kernel 70 §6.2, §8.2) -- only enabled Cues at this
// placement that viewerUserID can actually trigger right now, per
// CanTriggerCue (which itself hard-excludes Audience regardless of
// trigger_scope). No backstage-visibility check is required to call this
// -- it performs its own per-Cue authority check and returns nothing for
// a viewer who cannot trigger anything, which is itself the correct
// "no controls for you" response for Audience.
func ListTriggerableCuesForViewer(ctx context.Context, pool *pgxpool.Pool, viewerUserID, placementID string) ([]PlayerVisibleCue, error) {
	all, err := ListCuesForPlacement(ctx, pool, placementID)
	if err != nil {
		return nil, err
	}
	out := make([]PlayerVisibleCue, 0, len(all))
	for _, c := range all {
		if !c.Enabled {
			continue
		}
		ok, err := CanTriggerCue(ctx, pool, viewerUserID, c.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		label := strings.TrimSpace(c.StageButtonLabel)
		if label == "" {
			label = c.InternalName
		}
		out = append(out, PlayerVisibleCue{ID: c.ID, Label: label})
	}
	return out, nil
}

// ListCuesForPlacement returns every Cue attached to one Show Scene
// Placement, ordered by sort_order. Callers must already have passed a
// backstage-visibility check before calling.
func ListCuesForPlacement(ctx context.Context, pool *pgxpool.Pool, placementID string) ([]Cue, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+cueColumns+`
		FROM cues
		WHERE show_scene_placement_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, placementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Cue
	for rows.Next() {
		c, err := scanCue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
