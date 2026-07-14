package cues

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/shows"
)

// ExecuteCue is the GO press (Kernel 70 §6.4). Authority is re-checked
// here (not just by the HTTP handler) since this is the actual
// state-mutating boundary. Execution is idempotency-keyed per press:
// repeating the same (cue, key) pair replays the original result rather
// than re-executing, and a concurrent duplicate press loses a DB-level
// unique-index race and falls back to replay too.
func ExecuteCue(ctx context.Context, pool *pgxpool.Pool, actorUserID, cueID, idempotencyKey string) (CueExecutionResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return CueExecutionResult{}, errors.New("not_authenticated")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return CueExecutionResult{}, errors.New("idempotency_key_required")
	}

	c, err := LoadCueByID(ctx, pool, cueID)
	if err != nil {
		return CueExecutionResult{}, err
	}
	if !c.Enabled {
		return CueExecutionResult{}, errors.New("cue_disabled")
	}

	canTrigger, err := CanTriggerCue(ctx, pool, actorUserID, cueID)
	if err != nil {
		return CueExecutionResult{}, err
	}
	if !canTrigger {
		return CueExecutionResult{}, errors.New("not_authorized")
	}

	showID, _, _, err := placementShowShowRunLocation(ctx, pool, c.ShowScenePlacementID)
	if err != nil {
		return CueExecutionResult{}, err
	}

	if existing, found, err := loadExecutionByIdempotencyKey(ctx, pool, cueID, idempotencyKey); err != nil {
		return CueExecutionResult{}, err
	} else if found {
		if existing.Status == "in_progress" {
			return CueExecutionResult{}, errors.New("cue_execution_in_progress")
		}
		return existing, nil
	}

	executionID, err := insertInProgressExecution(ctx, pool, cueID, showID, actorUserID, idempotencyKey)
	if err != nil {
		if strings.Contains(err.Error(), "uq_cue_executions_idempotency") {
			if existing, found, ferr := loadExecutionByIdempotencyKey(ctx, pool, cueID, idempotencyKey); ferr != nil {
				return CueExecutionResult{}, ferr
			} else if found {
				return existing, nil
			}
		}
		return CueExecutionResult{}, err
	}

	var cueActions []CueAction
	if len(c.Actions) > 0 {
		if err := json.Unmarshal(c.Actions, &cueActions); err != nil {
			return CueExecutionResult{}, err
		}
	}

	activeSessionID := activeSessionIDForShow(ctx, pool, showID)

	results := make([]ActionResult, 0, len(cueActions))
	overallStatus := "succeeded"
	anySucceeded := false

	// Fail-stop, not best-effort-continue: actions in a Cue are usually
	// causally ordered (e.g. "change scene" then "announce it"), so a
	// failure at action N does not attempt action N+1. Actions 0..N-1
	// that already committed are NOT rolled back -- each action commits
	// independently, so the recorded outcome (partial_failure with
	// per-action detail) is an honest description of real, already-visible
	// state, not a fiction papered over by an all-or-nothing rollback that
	// heterogeneous side effects (Show pointer change + game event mirror +
	// variable set) don't share a clean boundary for anyway.
	for i, action := range cueActions {
		if err := executeOneAction(ctx, pool, showID, activeSessionID, actorUserID, action); err != nil {
			results = append(results, ActionResult{ActionIndex: i, Type: action.Type, Status: "failed", Error: err.Error()})
			if anySucceeded {
				overallStatus = "partial_failure"
			} else {
				overallStatus = "failed"
			}
			break
		}
		results = append(results, ActionResult{ActionIndex: i, Type: action.Type, Status: "succeeded"})
		anySucceeded = true
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return CueExecutionResult{}, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE cue_executions
		SET status = $2, action_results = $3, completed_at = NOW()
		WHERE id = $1
	`, executionID, overallStatus, resultsJSON); err != nil {
		return CueExecutionResult{}, err
	}

	return CueExecutionResult{
		ExecutionID:   executionID,
		CueID:         cueID,
		ShowID:        showID,
		Status:        overallStatus,
		ActionResults: results,
	}, nil
}

func loadExecutionByIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, cueID, idempotencyKey string) (CueExecutionResult, bool, error) {
	var executionID, showID, status, resultsText string
	err := pool.QueryRow(ctx, `
		SELECT id::text, show_id::text, status, action_results::text
		FROM cue_executions
		WHERE cue_id = $1 AND idempotency_key = $2
	`, cueID, idempotencyKey).Scan(&executionID, &showID, &status, &resultsText)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CueExecutionResult{}, false, nil
		}
		return CueExecutionResult{}, false, err
	}
	var results []ActionResult
	if resultsText != "" {
		_ = json.Unmarshal([]byte(resultsText), &results)
	}
	return CueExecutionResult{ExecutionID: executionID, CueID: cueID, ShowID: showID, Status: status, ActionResults: results}, true, nil
}

func insertInProgressExecution(ctx context.Context, pool *pgxpool.Pool, cueID, showID, actorUserID, idempotencyKey string) (string, error) {
	var executionID string
	err := pool.QueryRow(ctx, `
		INSERT INTO cue_executions (cue_id, show_id, triggered_by_user_id, idempotency_key, status)
		VALUES ($1, $2, $3, $4, 'in_progress')
		RETURNING id::text
	`, cueID, showID, actorUserID, idempotencyKey).Scan(&executionID)
	return executionID, err
}

// activeSessionIDForShow returns the current live/rehearsal session linked
// to this Show, or "" if none. go_to_scene and set_show_variable succeed
// either way (they write directly to shows.*); the optional actions-table
// mirror row (which lets a connected client's snapshot fold and activity
// feed reflect the Cue in real time) is only written when a session
// exists, since actions.session_id is NOT NULL.
func activeSessionIDForShow(ctx context.Context, pool *pgxpool.Pool, showID string) string {
	var sessionID string
	_ = pool.QueryRow(ctx, `
		SELECT id::text FROM sessions
		WHERE show_id = $1 AND status IN ('rehearsal', 'live')
		ORDER BY started_at DESC LIMIT 1
	`, showID).Scan(&sessionID)
	return sessionID
}

func executeOneAction(ctx context.Context, pool *pgxpool.Pool, showID, activeSessionID, actorUserID string, action CueAction) error {
	switch action.Type {
	case ActionTypeGoToScene:
		if action.GoToScene == nil {
			return errors.New("go_to_scene_target_required")
		}
		return executeGoToScene(ctx, pool, showID, activeSessionID, actorUserID, *action.GoToScene)
	case ActionTypeEmitGameEvent:
		if action.EmitGameEvent == nil {
			return errors.New("emit_game_event_kind_required")
		}
		return executeEmitGameEvent(ctx, pool, activeSessionID, actorUserID, *action.EmitGameEvent)
	case ActionTypeSetShowVariable:
		if action.SetShowVariable == nil {
			return errors.New("set_show_variable_key_required")
		}
		return executeSetShowVariable(ctx, pool, showID, activeSessionID, actorUserID, *action.SetShowVariable)
	default:
		return errors.New("unknown_cue_action_type")
	}
}

// executeGoToScene sets the Show's persistent current-Scene pointer
// (Kernel 70 §6.3.1) via shows.SetCurrentScenePlacementTrusted, which
// itself validates the target belongs to this exact Show and is not
// archived/retired -- a Cue cannot target a Scene Placement from another
// Show (Kernel 70 §9).
func executeGoToScene(ctx context.Context, pool *pgxpool.Pool, showID, activeSessionID, actorUserID string, in GoToSceneAction) error {
	targetPlacementID := strings.TrimSpace(in.ShowScenePlacementID)
	if targetPlacementID == "" {
		return errors.New("go_to_scene_target_required")
	}
	if _, err := shows.SetCurrentScenePlacementTrusted(ctx, pool, showID, targetPlacementID); err != nil {
		return err
	}
	if activeSessionID != "" {
		return insertCueActionRow(ctx, pool, activeSessionID, showID, actorUserID, "cue/go_to_scene", map[string]any{
			"show_scene_placement_id": targetPlacementID,
		})
	}
	return nil
}

// executeEmitGameEvent reuses actions.StoreGameEventTrusted (Kernel 70
// §6.3.2 -- "use existing Game Event/action infrastructure where
// possible"). This action type requires an active session, since a game
// event is inherently a live-session mirror (actions.session_id NOT
// NULL) -- if none exists, the action fails and is recorded as such,
// rather than silently doing nothing.
func executeEmitGameEvent(ctx context.Context, pool *pgxpool.Pool, activeSessionID, actorUserID string, in EmitGameEventAction) error {
	if activeSessionID == "" {
		return errors.New("no_active_session_for_show")
	}
	_, err := actions.StoreGameEventTrusted(ctx, pool, actions.GameEventRequest{
		SessionID: activeSessionID,
		ActorID:   actorUserID,
		EventKind: in.EventKind,
		Detail:    in.Detail,
	})
	return err
}

// executeSetShowVariable writes the canonical show_id-scoped actions log
// row (source of truth) and the shows.variables_json materialized cache
// (Kernel 70 §4.2, §6.3.3). Unlike emit_game_event, this does not require
// an active session -- the variable itself is Show-level state, not
// session-scoped; the actions-table mirror is written only when a session
// exists to make it visible to connected clients immediately.
func executeSetShowVariable(ctx context.Context, pool *pgxpool.Pool, showID, activeSessionID, actorUserID string, in SetShowVariableAction) error {
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return errors.New("set_show_variable_key_required")
	}
	valueJSON, err := json.Marshal(in.Value)
	if err != nil {
		return err
	}
	if err := shows.SetShowVariableTrusted(ctx, pool, showID, key, valueJSON); err != nil {
		return err
	}
	if activeSessionID != "" {
		return insertCueActionRow(ctx, pool, activeSessionID, showID, actorUserID, "cue/set_show_variable", map[string]any{
			"key":   key,
			"value": in.Value,
		})
	}
	return nil
}

// insertCueActionRow writes a single actions row tagged with BOTH
// session_id and show_id (Kernel 70's chosen shape -- one row, not two,
// avoiding double-counting in world.Snapshot's combined action feed). This
// is the mechanism that lets a Show's Cue history persist and fold across
// session boundaries via world.LoadVenueSnapshot.
func insertCueActionRow(ctx context.Context, pool *pgxpool.Pool, sessionID, showID, actorUserID, actionType string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO actions (session_id, show_id, moment_id, actor_id, type, target, payload)
		VALUES (
			$1, $2,
			COALESCE((SELECT MAX(moment_id) FROM actions WHERE session_id = $1), 0) + 1,
			$3, $4, '{}'::jsonb, $5
		)
	`, sessionID, showID, actorUserID, actionType, payloadJSON)
	return err
}
