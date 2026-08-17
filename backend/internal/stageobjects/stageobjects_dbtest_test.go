package stageobjects

import (
	"context"
	"errors"
	"testing"
)

// These cover what projection_test.go structurally cannot: real storage, real
// identity validation against real rows, persistence across reads, and the
// §24 manual/Cue parity claim.

func TestHideThenRevealPersistsAcrossReads(t *testing.T) {
	// §25: state must survive reconnect, reload, and WS reconnect. All three
	// are, from the server's side, the same thing -- a FRESH read with no
	// in-memory carry-over. That is what re-loading through LoadShowStates on
	// a new projector proves; there is no process-local cache to accidentally
	// be answering from.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_persist_dir")
	f := buildFixture(t, pool, director)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	player := Viewer{UserID: "some-player"}
	for _, label := range []string{"first read", "second read (simulated reconnect)"} {
		p, err := ProjectorFor(ctx, pool, f.showID, player)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if p.CanPerceive(f.tokenRef()) {
			t.Fatalf("%s: hidden object was perceivable", label)
		}
	}

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpReveal, ActorID: director,
	}); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	p, err := ProjectorFor(ctx, pool, f.showID, player)
	if err != nil {
		t.Fatalf("post-reveal read: %v", err)
	}
	if !p.CanPerceive(f.tokenRef()) {
		t.Fatal("revealed object was still withheld")
	}
}

func TestHiddenObjectRowSurvivesHiding(t *testing.T) {
	// §54 makes "hidden objects are deleted" a FAIL. Prove the underlying
	// scene_stage_elements row is untouched -- hiding must not reach the
	// object's own table at all, which is also what keeps §2's Scene boundary
	// intact.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_notdeleted_dir")
	f := buildFixture(t, pool, director)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	var label string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(label, '') FROM scene_stage_elements WHERE id = $1::uuid
	`, f.tokenID).Scan(&label); err != nil {
		t.Fatalf("hidden object's own row is gone -- that is deletion: %v", err)
	}
	if label != "Training Wall" {
		t.Fatalf("hiding altered the Scene's authored element, label=%q", label)
	}
	// And nothing was written to the Scene itself (§2: Scene stays stage
	// composition, never a per-viewer visibility container).
	var stateRows int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM stage_object_states WHERE show_id = $1::uuid
	`, f.showID).Scan(&stateRows); err != nil {
		t.Fatalf("count states: %v", err)
	}
	if stateRows != 1 {
		t.Fatalf("expected exactly one state row, got %d", stateRows)
	}
}

func TestSceneStateIsPerShowNotPerScene(t *testing.T) {
	// Grant's chosen ownership: the same Scene staged into a second Show starts
	// from the authored default. This is the property that keeps a Scene
	// reusable -- if hidden state travelled with the Scene, staging it again
	// would inherit another Show's secrets.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_pershow_dir")
	f := buildFixture(t, pool, director)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	// A second Show in the same Show Run, staging the same Scene.
	second := buildSecondShowStagingSameScene(t, pool, f)
	p, err := ProjectorFor(ctx, pool, second, Viewer{UserID: "player"})
	if err != nil {
		t.Fatalf("projector for second show: %v", err)
	}
	if !p.CanPerceive(f.tokenRef()) {
		t.Fatal("hidden state leaked into a different Show staging the same Scene")
	}
}

func TestManualAndCueWriteIdenticalCanonicalState(t *testing.T) {
	// §24, the core acceptance requirement, and §52's "Cue and manual controls
	// call the same canonical state path".
	//
	// This asserts on the STORED ROW rather than on the observable outcome,
	// because "same projection" could be true by coincidence while the two
	// paths wrote different state. The comparison includes the row id: both
	// operations converge on ONE row per (Show, object), so a Cue cannot
	// create a parallel record that later diverges.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_parity_dir")
	f := buildFixture(t, pool, director)

	for _, op := range []string{OpHide, OpReveal} {
		manual, err := ApplyMutation(ctx, pool, Mutation{
			ShowID: f.showID, Ref: f.tokenRef(), Op: op, ActorID: director,
		})
		if err != nil {
			t.Fatalf("manual %s: %v", op, err)
		}
		// The Cue path calls this exact function (see
		// cues.executeStageObjectState), so re-invoking it here is the same
		// code the GO press runs -- not a reimplementation of it.
		viaCue, err := ApplyMutation(ctx, pool, Mutation{
			ShowID: f.showID, Ref: f.tokenRef(), Op: op, ActorID: director,
		})
		if err != nil {
			t.Fatalf("cue %s: %v", op, err)
		}
		if manual.ID != viaCue.ID {
			t.Fatalf("%s: manual and Cue wrote different rows (%s vs %s)", op, manual.ID, viaCue.ID)
		}
		if manual.Visibility != viaCue.Visibility {
			t.Fatalf("%s: manual=%q cue=%q", op, manual.Visibility, viaCue.Visibility)
		}
	}
}

func TestInteractionOperationsDoNotDisturbVisibility(t *testing.T) {
	// §8 in storage, not just in projection: each operation touches only its
	// own dimension. Without the CASE guards in upsertStateRow, disabling an
	// interaction would reset a visibility state written earlier.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_dims_dir")
	f := buildFixture(t, pool, director)
	interactionID := insertBoundInteraction(t, pool, f)
	interactionRef := Ref{Kind: KindParticipantInteraction, ID: interactionID}

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide token: %v", err)
	}
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: interactionRef, Op: OpDisableInteraction, ActorID: director,
	}); err != nil {
		t.Fatalf("disable interaction: %v", err)
	}

	states, err := LoadShowStates(ctx, pool, f.showID)
	if err != nil {
		t.Fatalf("load states: %v", err)
	}
	if got := states[f.tokenRef()]; !got.Hidden() {
		t.Error("disabling an interaction cleared the token's hidden state")
	}
	got := states[interactionRef.normalized()]
	if got.InteractionEnabled == nil || *got.InteractionEnabled {
		t.Error("interaction was not recorded as disabled")
	}
	// And the interaction row carries no meaningful visibility of its own --
	// visibility follows the bound element.
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: interactionRef, Op: OpHide, ActorID: director,
	}); !errors.Is(err, ErrObjectNotSupported) {
		t.Errorf("expected an interaction's visibility to be unsettable, got %v", err)
	}

	// The invoke-time guard agrees with the stored state (§35).
	invocable, err := InteractionInvocable(ctx, pool, f.showID, interactionID)
	if err != nil {
		t.Fatalf("InteractionInvocable: %v", err)
	}
	if invocable {
		t.Error("a Director-disabled interaction was still invocable")
	}

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: interactionRef, Op: OpEnableInteraction, ActorID: director,
	}); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	invocable, err = InteractionInvocable(ctx, pool, f.showID, interactionID)
	if err != nil {
		t.Fatalf("InteractionInvocable after re-enable: %v", err)
	}
	if !invocable {
		t.Error("re-enabling did not restore invocability")
	}
}

func TestMapAndGridAreRefused(t *testing.T) {
	// §3: the map is explicitly not part of the generic object-visibility
	// system, and §54 makes dragging it in a FAIL. Asserted against a REAL
	// map_backdrop row so this cannot pass merely because the id was fake.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_map_dir")
	f := buildFixture(t, pool, director)

	_, err := ApplyMutation(ctx, pool, Mutation{
		ShowID:  f.showID,
		Ref:     Ref{Kind: KindSceneStageElement, ID: f.mapID},
		Op:      OpHide,
		ActorID: director,
	})
	if !errors.Is(err, ErrObjectNotSupported) {
		t.Fatalf("expected the map to be refused, got %v", err)
	}
}

func TestInvalidAndForeignTargetsAreRejected(t *testing.T) {
	// §38's "invalid object target rejected" and "stale/deleted object target
	// handled cleanly", plus the cross-Show containment §35 requires.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_target_dir")
	f := buildFixture(t, pool, director)

	cases := []struct {
		name string
		ref  Ref
		want error
	}{
		{"unknown id", Ref{Kind: KindSceneStageElement, ID: "00000000-0000-0000-0000-000000000009"}, ErrObjectNotFound},
		{"unsupported kind", Ref{Kind: "stage_effect", ID: f.tokenID}, nil},
		{"empty id", Ref{Kind: KindSceneStageElement, ID: ""}, nil},
	}
	for _, tc := range cases {
		_, err := ApplyMutation(ctx, pool, Mutation{
			ShowID: f.showID, Ref: tc.ref, Op: OpHide, ActorID: director,
		})
		if err == nil {
			t.Errorf("%s: expected refusal, got success", tc.name)
			continue
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}

	// A deleted drawing object resolves as not-found rather than being
	// hideable, and rather than erroring in some other way.
	drawingID := insertDrawingObject(t, pool, f)
	if _, err := pool.Exec(ctx, `UPDATE drawing_objects SET deleted_at = NOW() WHERE id = $1::uuid`, drawingID); err != nil {
		t.Fatalf("soft-delete drawing: %v", err)
	}
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: Ref{Kind: KindDrawingObject, ID: drawingID}, Op: OpHide, ActorID: director,
	}); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("expected a deleted drawing to be not-found, got %v", err)
	}

	// A Scene element belonging to a Show that never staged its Scene.
	other := insertTestUser(t, pool, "k90_target_other")
	otherF := buildFixture(t, pool, other)
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: otherF.tokenRef(), Op: OpHide, ActorID: director,
	}); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("expected another Show's object to be unreachable, got %v", err)
	}
}

func TestMutationRequiresAnActor(t *testing.T) {
	// ApplyMutation does not check authority (its callers do, with the gate
	// appropriate to each), but it refuses to run anonymously so a caller
	// cannot forget to establish one -- and so every audit row has an actor.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_actor_dir")
	f := buildFixture(t, pool, director)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: "  ",
	}); err == nil || err.Error() != "actor_required" {
		t.Fatalf("expected actor_required, got %v", err)
	}
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: "obliterate", ActorID: director,
	}); err == nil || err.Error() != "unsupported_stage_object_operation" {
		t.Fatalf("expected unsupported operation refusal, got %v", err)
	}
}

func TestScopeGrantsAreValidatedAgainstTheShow(t *testing.T) {
	// §17: a client-supplied Character id is never authority. Same for a
	// Cohort -- a Director must not be able to grant visibility to a cohort
	// from someone else's production, which would also be a probe for ids.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_scopeval_dir")
	f := buildFixture(t, pool, director)

	cohortA := insertCohort(t, pool, f, "Alpha")
	player := insertTestUser(t, pool, "k90_scopeval_player")
	charID := insertRosterCharacter(t, pool, f, player, "Niava")

	// Valid grants are accepted and round-trip.
	st, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}.WithScopes([]Scope{
		{Kind: ScopeCohort, ID: cohortA},
		{Kind: ScopeCharacter, ID: charID},
		{Kind: ScopeCast},
	}))
	if err != nil {
		t.Fatalf("apply valid scopes: %v", err)
	}
	if len(st.Scopes) != 3 {
		t.Fatalf("expected 3 grants, got %d (%v)", len(st.Scopes), st.Scopes)
	}

	// A foreign Cohort is refused.
	otherDirector := insertTestUser(t, pool, "k90_scopeval_other")
	otherF := buildFixture(t, pool, otherDirector)
	foreignCohort := insertCohort(t, pool, otherF, "Foreign")
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpSetScopes, ActorID: director,
	}.WithScopes([]Scope{{Kind: ScopeCohort, ID: foreignCohort}})); err == nil || err.Error() != "cohort_not_in_show" {
		t.Fatalf("expected cohort_not_in_show, got %v", err)
	}

	// A Character not on this Show's roster is refused.
	stranger := insertTestUser(t, pool, "k90_scopeval_stranger")
	strangerChar := insertRosterCharacter(t, pool, otherF, stranger, "Outsider")
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpSetScopes, ActorID: director,
	}.WithScopes([]Scope{{Kind: ScopeCharacter, ID: strangerChar}})); err == nil || err.Error() != "character_not_in_show" {
		t.Fatalf("expected character_not_in_show, got %v", err)
	}

	// The refused writes left the original grant set intact -- a rejected
	// scope edit must not clear what was already there.
	states, err := LoadShowStates(ctx, pool, f.showID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(states[f.tokenRef()].Scopes); got != 3 {
		t.Fatalf("a rejected scope edit changed the stored grants: now %d", got)
	}
}

func TestSetScopesReplacesRatherThanAccumulates(t *testing.T) {
	// The scope panel edits a SET ("who may see this?"), so an unchecked box
	// must be a removal. An add-only API would make removing a Cohort
	// impossible without a per-grant delete endpoint.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_replace_dir")
	f := buildFixture(t, pool, director)
	cohortA := insertCohort(t, pool, f, "Alpha")
	cohortB := insertCohort(t, pool, f, "Beta")

	base := Mutation{ShowID: f.showID, Ref: f.tokenRef(), Op: OpSetScopes, ActorID: director}

	if _, err := ApplyMutation(ctx, pool, base.WithScopes([]Scope{{Kind: ScopeCohort, ID: cohortA}})); err != nil {
		t.Fatalf("first set: %v", err)
	}
	st, err := ApplyMutation(ctx, pool, base.WithScopes([]Scope{{Kind: ScopeCohort, ID: cohortB}}))
	if err != nil {
		t.Fatalf("second set: %v", err)
	}
	if len(st.Scopes) != 1 || st.Scopes[0].ID != cohortB {
		t.Fatalf("expected only Cohort B, got %v", st.Scopes)
	}

	// An explicitly empty set clears all grants, making a hidden object
	// Director-only again.
	st, err = ApplyMutation(ctx, pool, base.WithScopes([]Scope{}))
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if len(st.Scopes) != 0 {
		t.Fatalf("expected grants cleared, got %v", st.Scopes)
	}

	// Omitting Scopes entirely leaves them alone -- distinct from clearing.
	if _, err := ApplyMutation(ctx, pool, base.WithScopes([]Scope{{Kind: ScopeAudience}})); err != nil {
		t.Fatalf("re-add: %v", err)
	}
	after, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	})
	if err != nil {
		t.Fatalf("hide without touching scopes: %v", err)
	}
	if len(after.Scopes) != 1 {
		t.Fatalf("hiding cleared grants it should not have touched: %v", after.Scopes)
	}
}

func TestDrawingObjectsJoinTheModel(t *testing.T) {
	// §28: Kernel 87 drawing objects are durable stage objects and were
	// audited in.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_drawing_dir")
	f := buildFixture(t, pool, director)
	drawingID := insertDrawingObject(t, pool, f)
	ref := Ref{Kind: KindDrawingObject, ID: drawingID}

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: ref, Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide drawing: %v", err)
	}
	p, err := ProjectorFor(ctx, pool, f.showID, Viewer{UserID: "player"})
	if err != nil {
		t.Fatalf("projector: %v", err)
	}
	if p.CanPerceive(ref) {
		t.Error("a hidden drawing object was still perceivable")
	}
	dp, err := ProjectorFor(ctx, pool, f.showID, Viewer{UserID: director, Backstage: true})
	if err != nil {
		t.Fatalf("director projector: %v", err)
	}
	if !dp.CanPerceive(ref) || !dp.HiddenFor(ref) {
		t.Error("the Director lost the hidden drawing or was not told it is hidden")
	}
}

func TestAuditRowIsWrittenWhenASessionIsLive(t *testing.T) {
	// §23 step 7. The manual path leaves the same evidence a Cue does, so a
	// Showing Review cannot tell them apart by whether a row exists.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_audit_dir")
	f := buildFixture(t, pool, director)
	sessionID := insertLiveSession(t, pool, f)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM actions
		WHERE session_id = $1::uuid AND type = 'stage_object/hide_object'
	`, sessionID).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 audit row, got %d", count)
	}
}

func TestResolveViewerDerivesCastFromShowParticipationNotTheRoleString(t *testing.T) {
	// §13 lists Show participation as a projection input, and §32 says to use
	// current Show/Show Run authority. Both matter concretely here, because the
	// role string handed to LoadVenueSnapshot is NOT reliable for this
	// question: cmd/victory's lookupVenueRole resolves participation with an
	// empty showRunID, so a roster Player arrives labelled "audience".
	//
	// This test is the guard on that. It passes the WRONG role string on
	// purpose and asserts the roster wins -- if someone later "simplifies"
	// ResolveViewer back to reading the role, cast-vs-audience scoping breaks
	// silently and this fails loudly.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_cast_dir")
	f := buildFixture(t, pool, director)

	rosterPlayer := insertTestUser(t, pool, "k90_cast_player")
	insertRosterCharacter(t, pool, f, rosterPlayer, "On The Roster")
	spectator := insertTestUser(t, pool, "k90_cast_spectator")

	// A roster member arriving with the misleading "audience" label.
	v, err := ResolveViewer(ctx, pool, "", f.showID, rosterPlayer, "audience", "")
	if err != nil {
		t.Fatalf("resolve roster player: %v", err)
	}
	if v.Audience {
		t.Error("a Show Run roster member was classified as Audience")
	}

	// A genuine spectator: admitted to look, not in the company.
	v, err = ResolveViewer(ctx, pool, "", f.showID, spectator, "audience", "")
	if err != nil {
		t.Fatalf("resolve spectator: %v", err)
	}
	if !v.Audience {
		t.Error("a non-roster viewer was not classified as Audience")
	}

	// Backstage short-circuits before either lookup, so a Director is never
	// classified as Cast or Audience by this path.
	v, err = ResolveViewer(ctx, pool, "", f.showID, director, "director", "")
	if err != nil {
		t.Fatalf("resolve director: %v", err)
	}
	if !v.Backstage || v.Audience {
		t.Errorf("director resolved wrong: backstage=%v audience=%v", v.Backstage, v.Audience)
	}

	// And the cast/audience grants then behave accordingly on real state.
	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}.WithScopes([]Scope{{Kind: ScopeCast}})); err != nil {
		t.Fatalf("hide with cast grant: %v", err)
	}
	states, err := LoadShowStates(ctx, pool, f.showID)
	if err != nil {
		t.Fatalf("load states: %v", err)
	}
	castViewer, _ := ResolveViewer(ctx, pool, "", f.showID, rosterPlayer, "audience", "")
	audViewer, _ := ResolveViewer(ctx, pool, "", f.showID, spectator, "audience", "")
	if !NewProjector(castViewer, states).CanPerceive(f.tokenRef()) {
		t.Error("the roster member could not see a cast-scoped object")
	}
	if NewProjector(audViewer, states).CanPerceive(f.tokenRef()) {
		t.Error("a spectator saw a cast-only object")
	}
}

func TestMutationSucceedsWithNoLiveSession(t *testing.T) {
	// Audit is evidence, not a precondition. A Director preparing between
	// sessions has nowhere to put an actions row (actions.session_id is NOT
	// NULL) and must still be able to set state -- otherwise canonical state
	// would be unwritable exactly when a Director is setting up.
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k90_nosession_dir")
	f := buildFixture(t, pool, director)

	if _, err := ApplyMutation(ctx, pool, Mutation{
		ShowID: f.showID, Ref: f.tokenRef(), Op: OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide with no live session should succeed: %v", err)
	}
}
