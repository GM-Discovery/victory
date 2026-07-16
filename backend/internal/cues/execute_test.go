package cues

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/shows"
)

// insertCueTestSession creates a session linked to the given Show and
// registers participantUserID as a session_participants row -- required
// by actions.StoreGameEventTrusted's identity.ResolveSessionIdentity
// lookup, which joins session_participants and errors on no rows
// otherwise. Defaults to the-cave for historical call sites; use
// insertCueTestSessionAtVenue directly to target a specific venue (Kernel
// 70A's loadActorIdentity fix means emit_game_event now works at any
// venue, not just the-cave -- see TestExecuteCueEmitGameEventWithActiveSessionSucceedsAtCatharsis).
func insertCueTestSession(t *testing.T, pool *pgxpool.Pool, showID, status, participantUserID string) string {
	t.Helper()
	return insertCueTestSessionAtVenue(t, pool, showID, status, participantUserID, "the-cave")
}

func insertCueTestSessionAtVenue(t *testing.T, pool *pgxpool.Pool, showID, status, participantUserID, venueSlug string) string {
	t.Helper()
	ctx := context.Background()
	var venueID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues WHERE slug = $1 LIMIT 1`, venueSlug).Scan(&venueID); err != nil {
		t.Fatalf("load %s venue: %v", venueSlug, err)
	}
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id) VALUES ($1, $2::session_status, $3) RETURNING id::text
	`, venueID, status, showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID) })

	if _, err := pool.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1, $2, 'producer'::location_role)
	`, sessionID, participantUserID); err != nil {
		t.Fatalf("insert session participant: %v", err)
	}
	return sessionID
}

func newIdempotencyKey(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate idempotency key: %v", err)
	}
	return hex.EncodeToString(buf)
}

func TestExecuteCueGoToSceneChangesShowCurrentPlacement(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_scene_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Go To Scene B",
		Actions: []CueAction{
			{Type: ActionTypeGoToScene, GoToScene: &GoToSceneAction{ShowScenePlacementID: f.placement2ID}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded, got %+v", result)
	}

	show, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	if show.CurrentShowScenePlacementID == nil || *show.CurrentShowScenePlacementID != f.placement2ID {
		t.Fatalf("expected current_show_scene_placement_id = %q, got %v", f.placement2ID, show.CurrentShowScenePlacementID)
	}
}

func TestExecuteCueSetShowVariable(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_var_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Set Mood Variable",
		Actions: []CueAction{
			{Type: ActionTypeSetShowVariable, SetShowVariable: &SetShowVariableAction{Key: "mood", Value: "tense"}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded, got %+v", result)
	}

	show, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	var vars map[string]any
	if err := json.Unmarshal(show.VariablesJSON, &vars); err != nil {
		t.Fatalf("unmarshal variables_json: %v", err)
	}
	if vars["mood"] != "tense" {
		t.Fatalf("expected variables_json.mood = tense, got %+v", vars)
	}
}

func TestExecuteCueEmitGameEventRequiresActiveSession(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_event_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Emit Event No Session",
		Actions: []CueAction{
			{Type: ActionTypeEmitGameEvent, EmitGameEvent: &EmitGameEventAction{EventKind: "test/kind"}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue (execution itself should not error, only the action): %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status with no active session, got %+v", result)
	}
	if len(result.ActionResults) != 1 || result.ActionResults[0].Status != "failed" {
		t.Fatalf("expected exactly one failed action result, got %+v", result.ActionResults)
	}
}

func TestExecuteCueEmitGameEventWithActiveSessionSucceeds(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_event_ok_producer")
	f := buildCueFixture(t, pool, producer)
	insertCueTestSession(t, pool, f.showID, "live", producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Emit Event With Session",
		Actions: []CueAction{
			{Type: ActionTypeEmitGameEvent, EmitGameEvent: &EmitGameEventAction{EventKind: "test/kind", Detail: map[string]any{"note": "hi"}}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded with an active session, got %+v", result)
	}
}

// TestExecuteCueEmitGameEventWithActiveSessionSucceedsAtCatharsis pins the
// Kernel 70A fix to actions.loadActorIdentity: this package's identity
// resolution was hardcoded to assume every session belonged to venue
// "the-cave", which would have made emit_game_event silently fail (wrong
// failure reason, not the intentional no_active_session_for_show) for any
// session at a different venue. Catharsis is the sole real functional
// venue for Kernel 70A -- this must succeed there, not just at the-cave.
func TestExecuteCueEmitGameEventWithActiveSessionSucceedsAtCatharsis(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_event_catharsis_producer")
	f := buildCueFixture(t, pool, producer)
	insertCueTestSessionAtVenue(t, pool, f.showID, "live", producer, "catharsis")

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Emit Event At Catharsis",
		Actions: []CueAction{
			{Type: ActionTypeEmitGameEvent, EmitGameEvent: &EmitGameEventAction{EventKind: "test/kind", Detail: map[string]any{"note": "hi"}}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded with an active Catharsis session, got %+v", result)
	}
}

func TestExecuteCuePartialFailureRecordsEachActionResult(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_partial_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Partial Failure Cue",
		Actions: []CueAction{
			{Type: ActionTypeSetShowVariable, SetShowVariable: &SetShowVariableAction{Key: "step1", Value: true}},
			{Type: ActionTypeGoToScene, GoToScene: &GoToSceneAction{ShowScenePlacementID: "00000000-0000-0000-0000-000000000000"}},
			{Type: ActionTypeSetShowVariable, SetShowVariable: &SetShowVariableAction{Key: "step3", Value: true}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "partial_failure" {
		t.Fatalf("expected partial_failure, got %+v", result)
	}
	if len(result.ActionResults) != 2 {
		t.Fatalf("expected fail-stop after action 1 (2 results: succeeded, failed), got %+v", result.ActionResults)
	}
	if result.ActionResults[0].Status != "succeeded" || result.ActionResults[1].Status != "failed" {
		t.Fatalf("expected [succeeded, failed], got %+v", result.ActionResults)
	}

	show, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	var vars map[string]any
	_ = json.Unmarshal(show.VariablesJSON, &vars)
	if vars["step1"] != true {
		t.Fatalf("expected step1 to have committed before the failing action, got %+v", vars)
	}
	if _, ok := vars["step3"]; ok {
		t.Fatalf("expected step3 to never run after a fail-stop, got %+v", vars)
	}
}

func TestExecuteCueIdempotentOnRepeatedKey(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_idem_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Idempotent Cue",
		Actions: []CueAction{
			{Type: ActionTypeSetShowVariable, SetShowVariable: &SetShowVariableAction{Key: "counter_marker", Value: "once"}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	key := newIdempotencyKey(t)

	first, err := ExecuteCue(context.Background(), pool, producer, c.ID, key)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	second, err := ExecuteCue(context.Background(), pool, producer, c.ID, key)
	if err != nil {
		t.Fatalf("second execute (same key) should replay, not error: %v", err)
	}
	if first.ExecutionID != second.ExecutionID {
		t.Fatalf("expected the same execution id on repeat, got %q vs %q", first.ExecutionID, second.ExecutionID)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM cue_executions WHERE cue_id = $1 AND idempotency_key = $2
	`, c.ID, key).Scan(&count); err != nil {
		t.Fatalf("count executions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one cue_executions row for this idempotency key, got %d", count)
	}
}

// TestExecuteCueConcurrentSameKeyOneWins fires two simultaneous requests
// with the same idempotency key. Whichever loses the DB-level unique-index
// race must never silently double-apply the Cue's actions -- it either
// replays the winner's final result or, if genuinely concurrent, receives
// cue_execution_in_progress. Either way, exactly one cue_executions row
// exists for the key afterward.
func TestExecuteCueConcurrentSameKeyOneWins(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_race_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Race Cue",
		Actions: []CueAction{
			{Type: ActionTypeSetShowVariable, SetShowVariable: &SetShowVariableAction{Key: "race_marker", Value: "set"}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	key := newIdempotencyKey(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := ExecuteCue(context.Background(), pool, producer, c.ID, key)
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	acceptable := func(err error) bool {
		return err == nil || err.Error() == "cue_execution_in_progress"
	}
	if !acceptable(errs[0]) || !acceptable(errs[1]) {
		t.Fatalf("expected both concurrent calls to either succeed or report in-progress, got %v and %v", errs[0], errs[1])
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM cue_executions WHERE cue_id = $1 AND idempotency_key = $2
	`, c.ID, key).Scan(&count); err != nil {
		t.Fatalf("count executions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one cue_executions row despite the concurrent race, got %d", count)
	}
}

func TestExecuteCueRejectsUnauthorizedActor(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_auth_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Director Only Cue", TriggerScope: TriggerScopeDirectorCrewOnly,
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	player := insertCuesTestUser(t, pool, "cue_exec_auth_player")
	addRoster(t, pool, producer, f.showRunID, player, "player")

	if _, err := ExecuteCue(context.Background(), pool, player, c.ID, newIdempotencyKey(t)); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for a player under director_crew_only, got %v", err)
	}

	audience := insertCuesTestUser(t, pool, "cue_exec_auth_audience")
	addRoster(t, pool, producer, f.showRunID, audience, "audience")
	if _, err := ExecuteCue(context.Background(), pool, audience, c.ID, newIdempotencyKey(t)); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for audience, got %v", err)
	}
}

func TestExecuteCueGoToSceneRejectsPlacementFromAnotherShow(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_exec_cross_show_producer")
	f := buildCueFixture(t, pool, producer)
	other := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Cross Show Cue",
		Actions: []CueAction{
			{Type: ActionTypeGoToScene, GoToScene: &GoToSceneAction{ShowScenePlacementID: other.placementID}},
		},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, c.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("execute cue: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected a Cue targeting a placement from another Show to fail, got %+v", result)
	}
	if len(result.ActionResults) != 1 || result.ActionResults[0].Error != "placement_show_mismatch" {
		t.Fatalf("expected placement_show_mismatch, got %+v", result.ActionResults)
	}
}
