package cues

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/stageobjects"
)

// Kernel 90 §39. These execute REAL Cues through the real GO press
// (ExecuteCue) against real rows -- the four actions Kernel 70 deferred for
// want of a canonical object identity.
//
// What they are careful NOT to do is re-test stageobjects' own semantics.
// Whether a hidden object is withheld from a Player is proven in that
// package; what matters here is that a Cue reaches the same state path, with
// the same validation, leaving the same evidence.

// sceneTokenForPlacement adds a durable composition token to the Scene behind a
// placement, so a Cue has a real canonical target to name.
func sceneTokenForPlacement(t *testing.T, pool *pgxpool.Pool, placementID string) string {
	t.Helper()
	ctx := context.Background()
	var sceneID string
	if err := pool.QueryRow(ctx, `
		SELECT scene_id::text FROM show_scene_placements WHERE id = $1
	`, placementID).Scan(&sceneID); err != nil {
		t.Fatalf("resolve scene for placement: %v", err)
	}
	var tokenID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO scene_stage_elements (scene_id, kind, label)
		VALUES ($1::uuid, 'token', 'Cue Target Token')
		RETURNING id::text
	`, sceneID).Scan(&tokenID); err != nil {
		t.Fatalf("insert scene token: %v", err)
	}
	return tokenID
}

func boundInteractionForPlacement(t *testing.T, pool *pgxpool.Pool, placementID, tokenID, actorUserID string) string {
	t.Helper()
	ctx := context.Background()
	var interactionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO participant_interactions (
			show_scene_placement_id, internal_name, stage_button_label,
			interaction_type, configuration_json, enabled, sort_order, created_by_user_id
		)
		VALUES ($1::uuid, $2, 'Open the cabinet', 'open_equip_mode', '{}'::jsonb, TRUE, 0, $3::uuid)
		RETURNING id::text
	`, placementID, "k90-cue-interaction-"+testSuffix(t), actorUserID).Scan(&interactionID); err != nil {
		t.Fatalf("insert participant interaction: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO stage_element_bindings (
			scene_stage_element_id, binding_type, participant_interaction_id, created_by_user_id
		)
		VALUES ($1::uuid, 'participant_interaction', $2::uuid, $3::uuid)
	`, tokenID, interactionID, actorUserID); err != nil {
		t.Fatalf("bind interaction: %v", err)
	}
	return interactionID
}

func loadCanonicalState(t *testing.T, pool *pgxpool.Pool, showID string, ref stageobjects.Ref) (stageobjects.State, bool) {
	t.Helper()
	states, err := stageobjects.LoadShowStates(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load canonical states: %v", err)
	}
	st, ok := states[ref]
	return st, ok
}

func TestCueHideAndRevealObjectExecute(t *testing.T) {
	// §39: reveal_object and hide_object execute, persist state, and resolve
	// their target by canonical object ref.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_hide_producer")
	f := buildCueFixture(t, pool, producer)
	tokenID := sceneTokenForPlacement(t, pool, f.placementID)
	ref := stageobjects.Ref{Kind: stageobjects.KindSceneStageElement, ID: tokenID}

	hideCue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Hide the token",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: tokenID,
		}}},
	})
	if err != nil {
		t.Fatalf("create hide cue: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, hideCue.ID, newIdempotencyKey(t))
	if err != nil || result.Status != "succeeded" {
		t.Fatalf("execute hide cue: status=%q err=%v results=%+v", result.Status, err, result.ActionResults)
	}
	st, ok := loadCanonicalState(t, pool, f.showID, ref)
	if !ok || !st.Hidden() {
		t.Fatalf("hide_object did not persist canonical hidden state (found=%v state=%+v)", ok, st)
	}

	revealCue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Reveal the token",
		Actions: []CueAction{{Type: ActionTypeRevealObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: tokenID,
		}}},
	})
	if err != nil {
		t.Fatalf("create reveal cue: %v", err)
	}
	result, err = ExecuteCue(context.Background(), pool, producer, revealCue.ID, newIdempotencyKey(t))
	if err != nil || result.Status != "succeeded" {
		t.Fatalf("execute reveal cue: status=%q err=%v", result.Status, err)
	}
	st, ok = loadCanonicalState(t, pool, f.showID, ref)
	if !ok || st.Hidden() {
		t.Fatalf("reveal_object did not clear hidden state (state=%+v)", st)
	}
}

func TestCueAndManualControlConvergeOnOneRow(t *testing.T) {
	// §24/§52: "Cue and manual controls call the same canonical state path."
	//
	// Asserted on the row identity, not just the outcome: a Cue must not create
	// a parallel state record that agrees today and drifts later. This is the
	// §53 PARTIAL trigger "Cues still use separate state", made falsifiable.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_parity_producer")
	f := buildCueFixture(t, pool, producer)
	tokenID := sceneTokenForPlacement(t, pool, f.placementID)
	ref := stageobjects.Ref{Kind: stageobjects.KindSceneStageElement, ID: tokenID}

	// Manual hide first, through the same function the HTTP handler calls.
	manual, err := stageobjects.ApplyMutation(context.Background(), pool, stageobjects.Mutation{
		ShowID: f.showID, Ref: ref, Op: stageobjects.OpHide, ActorID: producer,
	})
	if err != nil {
		t.Fatalf("manual hide: %v", err)
	}

	// Then a Cue reveal.
	cue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Cue reveal after manual hide",
		Actions: []CueAction{{Type: ActionTypeRevealObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: tokenID,
		}}},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	if result, err := ExecuteCue(context.Background(), pool, producer, cue.ID, newIdempotencyKey(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute cue: status=%q err=%v", result.Status, err)
	}

	st, ok := loadCanonicalState(t, pool, f.showID, ref)
	if !ok {
		t.Fatal("state row vanished")
	}
	if st.ID != manual.ID {
		t.Fatalf("the Cue wrote a different row than the manual control (%s vs %s)", st.ID, manual.ID)
	}
	if st.Hidden() {
		t.Fatal("the Cue's reveal did not take effect on the shared row")
	}
	// Exactly one row for this object, from both paths combined.
	var rows int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM stage_object_states WHERE show_id = $1::uuid AND object_id = $2::uuid
	`, f.showID, tokenID).Scan(&rows); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 canonical row across manual+Cue, got %d", rows)
	}
}

func TestCueEnableAndDisableInteractionExecute(t *testing.T) {
	// §39: enable_interaction / disable_interaction execute against the same
	// canonical interaction state the manual control and the invoke-time guard
	// read.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_interaction_producer")
	f := buildCueFixture(t, pool, producer)
	tokenID := sceneTokenForPlacement(t, pool, f.placementID)
	interactionID := boundInteractionForPlacement(t, pool, f.placementID, tokenID, producer)
	ref := stageobjects.Ref{Kind: stageobjects.KindParticipantInteraction, ID: interactionID}

	disable, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Lock the cabinet",
		Actions: []CueAction{{Type: ActionTypeDisableInteraction, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindParticipantInteraction, ObjectID: interactionID,
		}}},
	})
	if err != nil {
		t.Fatalf("create disable cue: %v", err)
	}
	if result, err := ExecuteCue(context.Background(), pool, producer, disable.ID, newIdempotencyKey(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute disable cue: status=%q err=%v results=%+v", result.Status, err, result.ActionResults)
	}

	invocable, err := stageobjects.InteractionInvocable(context.Background(), pool, f.showID, interactionID)
	if err != nil {
		t.Fatalf("InteractionInvocable: %v", err)
	}
	if invocable {
		t.Error("a Cue-disabled interaction was still invocable -- disabled must actually disable")
	}
	// The object itself stays visible: disabled is not hidden (§8).
	if st, ok := loadCanonicalState(t, pool, f.showID, stageobjects.Ref{
		Kind: stageobjects.KindSceneStageElement, ID: tokenID,
	}); ok && st.Hidden() {
		t.Error("disabling an interaction hid its object")
	}

	enable, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Unlock the cabinet",
		Actions: []CueAction{{Type: ActionTypeEnableInteraction, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindParticipantInteraction, ObjectID: interactionID,
		}}},
	})
	if err != nil {
		t.Fatalf("create enable cue: %v", err)
	}
	if result, err := ExecuteCue(context.Background(), pool, producer, enable.ID, newIdempotencyKey(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute enable cue: status=%q err=%v", result.Status, err)
	}
	invocable, err = stageobjects.InteractionInvocable(context.Background(), pool, f.showID, interactionID)
	if err != nil {
		t.Fatalf("InteractionInvocable after enable: %v", err)
	}
	if !invocable {
		t.Error("re-enabling via Cue did not restore invocability")
	}

	if st, ok := loadCanonicalState(t, pool, f.showID, ref); !ok || st.InteractionEnabled == nil || !*st.InteractionEnabled {
		t.Errorf("interaction state not persisted as enabled: found=%v state=%+v", ok, st)
	}
}

func TestCueCannotBypassTargetValidation(t *testing.T) {
	// §35: "Cue execution cannot bypass normal target validation." A Cue is
	// authored once and fired later, so the interesting cases are the ones
	// where the world changed in between -- and they must fail loudly rather
	// than appear to succeed.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_validate_producer")
	f := buildCueFixture(t, pool, producer)
	tokenID := sceneTokenForPlacement(t, pool, f.placementID)

	// A target deleted after the Cue was authored.
	cue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Hide a doomed token",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: tokenID,
		}}},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		DELETE FROM scene_stage_elements WHERE id = $1::uuid
	`, tokenID); err != nil {
		t.Fatalf("delete token: %v", err)
	}

	result, err := ExecuteCue(context.Background(), pool, producer, cue.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("ExecuteCue itself should not error, the ACTION should fail: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected the action to fail cleanly, got status=%q results=%+v", result.Status, result.ActionResults)
	}
	if len(result.ActionResults) != 1 || result.ActionResults[0].Status != "failed" {
		t.Fatalf("expected one failed action result, got %+v", result.ActionResults)
	}

	// The map cannot be reached through a Cue either (§3/§54).
	var sceneID string
	if err := pool.QueryRow(context.Background(), `
		SELECT scene_id::text FROM show_scene_placements WHERE id = $1
	`, f.placementID).Scan(&sceneID); err != nil {
		t.Fatalf("resolve scene: %v", err)
	}
	var mapID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO scene_stage_elements (scene_id, kind, label)
		VALUES ($1::uuid, 'map_backdrop', 'Cue Map') RETURNING id::text
	`, sceneID).Scan(&mapID); err != nil {
		t.Fatalf("insert map: %v", err)
	}
	mapCue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Hide the map",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: mapID,
		}}},
	})
	if err != nil {
		t.Fatalf("create map cue: %v", err)
	}
	result, err = ExecuteCue(context.Background(), pool, producer, mapCue.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("ExecuteCue errored on the map target: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("a Cue reached the map through the object system: status=%q", result.Status)
	}
}

func TestCueCannotTargetAnotherShowsObject(t *testing.T) {
	// A Cue is placement-scoped, and its Show is resolved server-side from the
	// placement -- never from the payload. So an object belonging to a
	// different Show must be unreachable even though the actor is a legitimate
	// Director of both.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_crossshow_producer")
	f := buildCueFixture(t, pool, producer)

	other := insertCuesTestUser(t, pool, "k90_cue_crossshow_other")
	otherF := buildCueFixture(t, pool, other)
	foreignToken := sceneTokenForPlacement(t, pool, otherF.placementID)

	cue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Reach across Shows",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: foreignToken,
		}}},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	result, err := ExecuteCue(context.Background(), pool, producer, cue.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("ExecuteCue errored: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("a Cue mutated another Show's object: status=%q", result.Status)
	}
	if _, ok := loadCanonicalState(t, pool, otherF.showID, stageobjects.Ref{
		Kind: stageobjects.KindSceneStageElement, ID: foreignToken,
	}); ok {
		t.Fatal("state was written into the other Show")
	}
}

func TestCueVisibilityActionRecordsAudit(t *testing.T) {
	// §39/§23 step 7: "Cue audit/evidence follows current action conventions."
	// Both rows are expected -- stageobjects' own state-change row and cues'
	// per-action mirror row -- because they answer different questions ("state
	// changed" vs "this Cue fired"), the same way go_to_scene already writes
	// its own mirror alongside the Show pointer change.
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "k90_cue_audit_producer")
	f := buildCueFixture(t, pool, producer)
	tokenID := sceneTokenForPlacement(t, pool, f.placementID)
	sessionID := insertCueLiveSession(t, pool, f.showID)

	cue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Audited hide",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: stageobjects.KindSceneStageElement, ObjectID: tokenID,
		}}},
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	if result, err := ExecuteCue(context.Background(), pool, producer, cue.ID, newIdempotencyKey(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute: status=%q err=%v", result.Status, err)
	}

	for _, actionType := range []string{"cue/hide_object", "stage_object/hide_object"} {
		var count int
		if err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM actions WHERE session_id = $1::uuid AND type = $2
		`, sessionID, actionType).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", actionType, err)
		}
		if count != 1 {
			t.Errorf("expected 1 %s audit row, got %d", actionType, count)
		}
	}
}

func insertCueLiveSession(t *testing.T, pool *pgxpool.Pool, showID string) string {
	t.Helper()
	ctx := context.Background()
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		VALUES ((SELECT id FROM venues WHERE slug = 'catharsis' LIMIT 1), 'rehearsal', $1::uuid)
		RETURNING id::text
	`, showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM actions WHERE session_id = $1::uuid`, sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM sessions WHERE id = $1::uuid`, sessionID)
	})
	return sessionID
}
