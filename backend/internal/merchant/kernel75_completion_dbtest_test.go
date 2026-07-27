package merchant

import (
	"context"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/recognition"
	"victory/backend/internal/showings"
	"victory/backend/internal/storysofar"
	"victory/backend/internal/tutorial"
)

// TestKernel75TutorialCompletionEndToEnd is the closed-loop proof for Kernel
// 75's completion half: Ra's Leave opens the gate but does not end the
// tutorial, Continue is a separate retry-safe verb, Story So Far is
// generated deterministically from real recorded play and stays private, and
// one-time Player recognition survives a replay with a second Character.
//
// Throwaway venue for the same reason kernel74_tutorial_dbtest_test.go uses
// one: `go test ./...` runs packages concurrently against one shared
// victory_test database, and holding a session open on the real catharsis
// row would race another package's most-recent-session lookup.
func TestKernel75TutorialCompletionEndToEnd(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	suffix := time.Now().UTC().Format("150405.000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("lookup location: %v", err)
	}

	mustUser := func(handle string) string {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text`, handle).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", handle, err)
		}
		return id
	}
	directorUserID := mustUser("k75_director_" + suffix)
	playerAUserID := mustUser("k75_player_a_" + suffix)
	playerBUserID := mustUser("k75_player_b_" + suffix)
	outsiderUserID := mustUser("k75_outsider_" + suffix)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role) VALUES ($1, $2, 'director')
	`, locationID, directorUserID); err != nil {
		t.Fatalf("grant director role: %v", err)
	}

	var productionID, showRunID, showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K75 Production "+suffix, "k75-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K75 Run "+suffix, "k75-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k75-show-"+suffix, "K75 Show", directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}

	for _, r := range []struct{ user, role string }{
		{playerAUserID, "player"}, {playerBUserID, "player"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id) VALUES ($1, $2, $3, $4)
		`, showRunID, r.user, r.role, directorUserID); err != nil {
			t.Fatalf("insert roster row: %v", err)
		}
	}

	// Player A's Character carries a full workbook_context and Face Sheet
	// history, so the reflection has real archetype and attribute data to
	// draw on -- the S5.4 Example A shape.
	mustCharacter := func(owner, name, workbookContext string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO character_cards (owner_user_id, location_id, name, pronouns, workbook_context)
			VALUES ($1, $2, $3, 'they/them', $4::jsonb) RETURNING id::text
		`, owner, locationID, name, workbookContext).Scan(&id); err != nil {
			t.Fatalf("insert character %s: %v", name, err)
		}
		return id
	}
	richContext := `{
		"chapter2": {"attributes": {"Empathy": 8, "Awareness": 6, "Craft": 4}},
		"chapter3": {"archetype_key": "Observer", "archetype_title": "The Observer",
		             "primary_attribute": "Empathy", "secondary_attribute": "Awareness",
		             "key_skill": "Insight", "confirmed": true}
	}`
	characterAID := mustCharacter(playerAUserID, "K75 Trang", richContext)
	// A SECOND Character for Player A: proves per-Character replay (S7.3)
	// and that switching Character never shows the first one's reflection.
	characterA2ID := mustCharacter(playerAUserID, "K75 Trang Understudy", `{}`)
	// Player B's Character is deliberately sparse -- no archetype, no
	// attributes, no history. This is what a pre-Kernel-75 Character looks
	// like, and its story must still be complete and well-formed.
	characterBID := mustCharacter(playerBUserID, "K75 Mara", `{}`)

	if _, err := pool.Exec(ctx, `
		INSERT INTO character_workbook_entries (character_card_id, author_user_id, page_key, entry_type, title, body, stage_number, sort_order)
		VALUES ($1, $2, 'history', 'chapter2_stage', 'Stage 2: Childhood',
		        'You learned early to notice who was left outside the group, and Empathy came with it.', 2, 0)
	`, characterAID, playerAUserID); err != nil {
		t.Fatalf("insert face sheet line: %v", err)
	}

	setSelectedCharacter := func(user, character string) {
		if _, err := pool.Exec(ctx, `
			UPDATE show_run_roster_members SET character_card_id = $2 WHERE show_run_id = $1 AND user_id = $3
		`, showRunID, character, user); err != nil {
			t.Fatalf("select character: %v", err)
		}
	}
	setSelectedCharacter(playerAUserID, characterAID)
	setSelectedCharacter(playerBUserID, characterBID)

	venueSlug := "k75-stage-" + suffix
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		for _, id := range []string{characterAID, characterA2ID, characterBID} {
			_, _ = pool.Exec(cctx, `DELETE FROM character_story_events WHERE character_card_id = $1`, id)
			_, _ = pool.Exec(cctx, `DELETE FROM character_interaction_attempts WHERE character_card_id = $1`, id)
			_, _ = pool.Exec(cctx, `DELETE FROM character_workbook_entries WHERE character_card_id = $1`, id)
		}
		_, _ = pool.Exec(cctx, `DELETE FROM player_recognition_grants WHERE user_id IN ($1, $2)`, playerAUserID, playerBUserID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_local_projections WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_tutorial_progress WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_freeform_submissions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_dialogue_topic_views WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM messages WHERE session_id IN (SELECT id FROM sessions WHERE show_id = $1)`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM sessions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `
			DELETE FROM stage_element_bindings WHERE participant_interaction_id IN (
				SELECT id FROM participant_interactions WHERE show_scene_placement_id IN (
					SELECT id FROM show_scene_placements WHERE show_id = $1))
		`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_interactions WHERE show_scene_placement_id IN (SELECT id FROM show_scene_placements WHERE show_id = $1)`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_scene_placements WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM venues WHERE slug = $1`, venueSlug)
		_, _ = pool.Exec(cctx, `DELETE FROM character_cards WHERE id IN ($1, $2, $3)`, characterAID, characterA2ID, characterBID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_run_roster_members WHERE show_run_id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM shows WHERE id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_runs WHERE id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(cctx, `DELETE FROM location_memberships WHERE user_id = $1`, directorUserID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2, $3, $4)`,
			directorUserID, playerAUserID, playerBUserID, outsiderUserID)
	})

	prep, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("PrepareLockedCourtyardOpening: %v", err)
	}
	if !prep.Ready {
		t.Fatalf("expected Ready=true, gaps=%v", prep.Gaps)
	}

	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = $2 WHERE id = $1`, showID, prep.PlacementID); err != nil {
		t.Fatalf("set current placement: %v", err)
	}
	var lotID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM lots WHERE location_id = $1 AND slug = 'main-lot'`, locationID).Scan(&lotID); err != nil {
		t.Fatalf("lookup lot: %v", err)
	}
	var venueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public)
		VALUES ($1, 'K75 Stage', $2, 'plaza',
		  '{"participant_interactions_enabled": true, "participant_local_projection_enabled": true}'::jsonb, FALSE)
		RETURNING id::text
	`, lotID, venueSlug).Scan(&venueID); err != nil {
		t.Fatalf("insert fixture venue: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE show_scene_placements SET venue_id = $2 WHERE id = $1`, prep.PlacementID, venueID); err != nil {
		t.Fatalf("override placement venue: %v", err)
	}
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id) VALUES ($1, 'rehearsal', $2) RETURNING id::text
	`, venueID, showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := showings.EnsureForSession(ctx, pool, sessionID, directorUserID); err != nil {
		t.Fatalf("ensure showing: %v", err)
	}
	for _, p := range []struct{ user, role string }{
		{playerAUserID, "cast"}, {playerBUserID, "cast"}, {directorUserID, "director"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role) VALUES ($1, $2, $3::location_role)
		`, sessionID, p.user, p.role); err != nil {
			t.Fatalf("insert session participant: %v", err)
		}
	}

	// --- Player A plays the whole tail, including a stance and a Haggle ----

	if _, err := CompleteKessaIntro(ctx, pool, playerAUserID, prep.InteractionID); err != nil {
		t.Fatalf("CompleteKessaIntro: %v", err)
	}

	// Kernel 75 S5.1: these now leave a durable, Character-keyed row.
	if _, err := AttemptStance(ctx, pool, playerAUserID, prep.InteractionID, "insight"); err != nil {
		t.Fatalf("AttemptStance: %v", err)
	}
	if _, err := AttemptHaggle(ctx, pool, playerAUserID, prep.InteractionID); err != nil {
		t.Fatalf("AttemptHaggle: %v", err)
	}
	var attemptCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM character_interaction_attempts WHERE character_card_id = $1 AND show_id = $2
	`, characterAID, showID).Scan(&attemptCount); err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if attemptCount != 2 {
		t.Fatalf("expected 2 durable attempt rows (one stance, one haggle), got %d", attemptCount)
	}
	// The ephemeral actions rows must still be written -- Showing Review
	// reads them, and Kernel 75 adds a record rather than replacing one.
	var actionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM actions
		WHERE session_id = $1 AND type = 'game/event'
		  AND payload->>'event_kind' IN ('interaction/stance_attempted', 'interaction/haggle_attempted')
	`, sessionID).Scan(&actionCount); err != nil {
		t.Fatalf("count action rows: %v", err)
	}
	if actionCount < 2 {
		t.Fatalf("K73's ephemeral actions log must still be written, got %d rows", actionCount)
	}

	intention := "I study the hinges and test whether the door can be lifted instead of forced."
	if _, _, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, intention, "k75-key-1"); err != nil {
		t.Fatalf("SubmitFreeform: %v", err)
	}

	if _, err := OpenDialogue(ctx, pool, playerAUserID, prep.RaInteractionID); err != nil {
		t.Fatalf("OpenDialogue: %v", err)
	}
	for _, topic := range []string{"why-looking", "crown-bet"} {
		if _, err := ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, topic); err != nil {
			t.Fatalf("ReadTopic(%s): %v", topic, err)
		}
	}

	// --- S3.2: Continue is refused before the gate is open -----------------

	if _, err := CompleteTutorial(ctx, pool, playerBUserID, prep.RaInteractionID); err == nil {
		t.Fatal("Continue before leaving Ra must be refused, not merely hidden in the UI")
	} else if err.Error() != "milestone_required" {
		t.Fatalf("expected milestone_required, got %q", err.Error())
	}

	// --- S3.1: Leave opens the gate but does NOT complete the tutorial -----

	leave, err := LeaveDialogue(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("LeaveDialogue: %v", err)
	}
	if len(leave.State.ClosingBeats) == 0 {
		t.Fatal("Leave must return closing beats (or the closing_narration fallback)")
	}
	participationA := tutorial.Participation{UserID: playerAUserID, CharacterCardID: characterAID, ShowID: showID}
	gateOpen, err := tutorial.HasMilestone(ctx, pool, participationA, tutorial.MilestoneTutorialGateOpened)
	if err != nil {
		t.Fatalf("HasMilestone(gate): %v", err)
	}
	if !gateOpen {
		t.Fatal("Leave must record tutorial_gate_opened")
	}
	completedTooEarly, err := tutorial.HasMilestone(ctx, pool, participationA, tutorial.MilestoneTutorialCompleted)
	if err != nil {
		t.Fatalf("HasMilestone(completed): %v", err)
	}
	if completedTooEarly {
		t.Fatal("S3.1: leaving Ra must NOT complete the tutorial -- Continue is a separate Player act")
	}
	var storyBeforeContinue int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM character_story_events WHERE character_card_id = $1`, characterAID).Scan(&storyBeforeContinue); err != nil {
		t.Fatalf("count story before continue: %v", err)
	}
	if storyBeforeContinue != 0 {
		t.Fatalf("no Story So Far may exist before Continue, got %d rows", storyBeforeContinue)
	}

	// --- S3.2: Continue, and its idempotency -------------------------------

	completion, err := CompleteTutorial(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("CompleteTutorial: %v", err)
	}
	if completion.Headline == "" || len(completion.WaitingCopy) == 0 {
		t.Fatal("the completion payload must carry server-authored copy")
	}
	if len(completion.StoryEvents) == 0 {
		t.Fatal("Continue must generate Story So Far")
	}
	// NewlyGranted must be TRUE on the press that actually earned it.
	//
	// Regression guard: the first implementation granted correctly but then
	// rebuilt the payload with LoadTutorialCompletion, which is a pure read
	// and always reports NewlyGranted=false -- so the recognition beat never
	// rendered for anyone. The golden-journey proof on live data caught it;
	// an assertion that merely tolerated "either flag or grant" did not.
	if !completion.Recognition.NewlyGranted {
		t.Fatal("S7.3: the first tutorial completion must report newly_granted=true")
	}
	if completion.Recognition.Grant == nil {
		t.Fatal("a newly granted recognition must carry its grant")
	}
	if completion.Recognition.Grant.Label == "" {
		t.Fatal("the grant must carry an operator-facing label")
	}

	firstCount := len(completion.StoryEvents)

	// A double-clicked Continue must converge: no second milestone, no
	// duplicated story, no second recognition grant.
	for i := 0; i < 3; i++ {
		if _, err := CompleteTutorial(ctx, pool, playerAUserID, prep.RaInteractionID); err != nil {
			t.Fatalf("repeat CompleteTutorial %d: %v", i, err)
		}
	}
	var afterRetries int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM character_story_events WHERE character_card_id = $1`, characterAID).Scan(&afterRetries); err != nil {
		t.Fatalf("count story after retries: %v", err)
	}
	if afterRetries != firstCount {
		t.Fatalf("retried Continue duplicated a Player's history: %d -> %d rows", firstCount, afterRetries)
	}
	var milestoneRows int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM participant_tutorial_progress
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3 AND milestone_key = $4
	`, playerAUserID, characterAID, showID, tutorial.MilestoneTutorialCompleted).Scan(&milestoneRows); err != nil {
		t.Fatalf("count completion milestone: %v", err)
	}
	if milestoneRows != 1 {
		t.Fatalf("expected exactly 1 completion milestone row, got %d", milestoneRows)
	}

	// --- S5.1/S5.2: the reflection uses real recorded data ------------------

	joined := ""
	for _, e := range completion.StoryEvents {
		joined += e.Summary + "\n"
	}
	if !strings.Contains(joined, intention) {
		t.Fatalf("the door intention must appear verbatim in the reflection.\ngot:\n%s", joined)
	}
	if !strings.Contains(joined, "Empathy") {
		t.Fatalf("the archetype-selected lens (Empathy) must appear.\ngot:\n%s", joined)
	}
	if !strings.Contains(joined, "left outside the group") {
		t.Fatalf("the matching Face Sheet line must be quoted.\ngot:\n%s", joined)
	}
	if !strings.Contains(strings.ToLower(joined), "kessa") {
		t.Fatalf("the Kessa beat must appear.\ngot:\n%s", joined)
	}

	// --- S6.3: private by default ------------------------------------------

	for _, e := range completion.StoryEvents {
		if e.VisibilityState != storysofar.VisibilityPrivate {
			t.Fatalf("generated entries must be private by default, %q was %q", e.EventType, e.VisibilityState)
		}
		if !e.Generated() {
			t.Fatalf("generated entries must have no author, %q claimed %q", e.EventType, e.AuthoredByUserID)
		}
	}

	// --- S12: a non-owner sees only shared entries -------------------------

	asOwner, err := storysofar.ListForCharacter(ctx, pool, characterAID, true)
	if err != nil {
		t.Fatalf("ListForCharacter(owner): %v", err)
	}
	asOther, err := storysofar.ListForCharacter(ctx, pool, characterAID, false)
	if err != nil {
		t.Fatalf("ListForCharacter(non-owner): %v", err)
	}
	if len(asOwner) == 0 {
		t.Fatal("the owner must see their own history")
	}
	if len(asOther) != 0 {
		t.Fatalf("a non-owner must see no private entries, got %d", len(asOther))
	}

	// Revealing one entry shares exactly that one.
	revealed, err := storysofar.SetVisibility(ctx, pool, playerAUserID, asOwner[0].ID, storysofar.VisibilityTable)
	if err != nil {
		t.Fatalf("SetVisibility: %v", err)
	}
	if revealed.VisibilityState != storysofar.VisibilityTable {
		t.Fatalf("expected the entry to be shared, got %q", revealed.VisibilityState)
	}
	asOther, err = storysofar.ListForCharacter(ctx, pool, characterAID, false)
	if err != nil {
		t.Fatalf("ListForCharacter(non-owner, after reveal): %v", err)
	}
	if len(asOther) != 1 {
		t.Fatalf("expected exactly the one revealed entry, got %d", len(asOther))
	}

	// S12: another Player -- including one with a Character in the same
	// Show Run -- cannot flip someone else's visibility.
	if _, err := storysofar.SetVisibility(ctx, pool, playerBUserID, asOwner[1].ID, storysofar.VisibilityTable); err == nil {
		t.Fatal("a different Player must not be able to publish someone else's reflection")
	}
	// Nor can a Director with location authority, who passes CanEditCard.
	if _, err := storysofar.SetVisibility(ctx, pool, directorUserID, asOwner[1].ID, storysofar.VisibilityTable); err == nil {
		t.Fatal("a location Director must not be able to publish a Player's private reflection")
	}

	// --- S7.3: recognition is once per Player, milestones are per Character -

	setSelectedCharacter(playerAUserID, characterA2ID)

	// The second Character has none of the first's progress.
	participationA2 := tutorial.Participation{UserID: playerAUserID, CharacterCardID: characterA2ID, ShowID: showID}
	carried, err := tutorial.HasMilestone(ctx, pool, participationA2, tutorial.MilestoneTutorialCompleted)
	if err != nil {
		t.Fatalf("HasMilestone(second character): %v", err)
	}
	if carried {
		t.Fatal("S5.2: switching Character must not carry the first Character's completion")
	}
	var secondCharacterStory int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM character_story_events WHERE character_card_id = $1`, characterA2ID).Scan(&secondCharacterStory); err != nil {
		t.Fatalf("count second character story: %v", err)
	}
	if secondCharacterStory != 0 {
		t.Fatal("switching Character must never show the first Character's reflection")
	}

	// Re-granting for the same Player reports newly_granted=false.
	replay, err := recognition.GrantFirstTutorialCompleted(ctx, pool, playerAUserID, characterA2ID, showID)
	if err != nil {
		t.Fatalf("GrantFirstTutorialCompleted(replay): %v", err)
	}
	if replay.NewlyGranted {
		t.Fatal("S7.3: one-time Player recognition must not be awarded twice")
	}
	var grantRows int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM player_recognition_grants WHERE user_id = $1`, playerAUserID).Scan(&grantRows); err != nil {
		t.Fatalf("count grants: %v", err)
	}
	if grantRows != 1 {
		t.Fatalf("expected exactly 1 recognition grant for this Player, got %d", grantRows)
	}

	// A different Player earns their own, independently.
	otherGrant, err := recognition.GrantFirstTutorialCompleted(ctx, pool, playerBUserID, characterBID, showID)
	if err != nil {
		t.Fatalf("GrantFirstTutorialCompleted(player B): %v", err)
	}
	if !otherGrant.NewlyGranted {
		t.Fatal("a different Player must earn their own first-completion recognition")
	}

	// --- S5.2: a sparse Character still gets a complete, honest story ------

	sparseRef := storysofar.Ref{
		CharacterCardID: characterBID,
		OwnerUserID:     playerBUserID,
		ShowRunID:       showRunID,
		ShowID:          showID,
	}
	sparseInputs, err := storysofar.LoadInputs(ctx, pool, sparseRef, storyRules(), time.Now().UTC())
	if err != nil {
		t.Fatalf("LoadInputs(sparse): %v", err)
	}
	// Player B never played, so there is nothing to reflect on yet -- and
	// that must be an empty story rather than a story full of blanks.
	sparseDrafts := storysofar.Generate(sparseInputs, storyRules())
	for _, d := range sparseDrafts {
		if strings.TrimSpace(d.Summary) == "" {
			t.Fatalf("clause %q rendered an empty summary", d.EventType)
		}
		if strings.Contains(d.Summary, "%!") || strings.Contains(d.Summary, "<nil>") {
			t.Fatalf("clause %q rendered a formatting artifact: %s", d.EventType, d.Summary)
		}
	}

	// --- S4.3: the reopen path records and grants nothing -------------------

	setSelectedCharacter(playerAUserID, characterAID)
	counts := func() (story, grants int) {
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM character_story_events WHERE character_card_id = $1`, characterAID).Scan(&story); err != nil {
			t.Fatalf("count story: %v", err)
		}
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM player_recognition_grants WHERE user_id = $1`, playerAUserID).Scan(&grants); err != nil {
			t.Fatalf("count grants: %v", err)
		}
		return story, grants
	}
	storyBefore, grantsBefore := counts()

	reopened, err := LoadTutorialCompletion(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("LoadTutorialCompletion: %v", err)
	}
	if len(reopened.StoryEvents) != storyBefore {
		t.Fatalf("reopening must show the same story, got %d of %d", len(reopened.StoryEvents), storyBefore)
	}
	if reopened.Recognition.NewlyGranted {
		t.Fatal("reopening the Program must not re-award recognition")
	}
	storyAfter, grantsAfter := counts()
	if storyAfter != storyBefore || grantsAfter != grantsBefore {
		t.Fatalf("the reopen path wrote something: story %d->%d, grants %d->%d",
			storyBefore, storyAfter, grantsBefore, grantsAfter)
	}

	// --- S12: an outsider cannot reach any of it ----------------------------

	if _, err := CompleteTutorial(ctx, pool, outsiderUserID, prep.RaInteractionID); err == nil {
		t.Fatal("a non-roster user must not be able to complete someone's tutorial")
	}
	if _, err := LoadTutorialCompletion(ctx, pool, outsiderUserID, prep.RaInteractionID); err == nil {
		t.Fatal("a non-roster user must not be able to read a completion payload")
	}
	if _, err := CompleteTutorial(ctx, pool, "", prep.RaInteractionID); err == nil {
		t.Fatal("an anonymous caller must be refused")
	}
}

// TestKernel75TrailerFaceCarriesNoStoryEvents proves S6.3 and S12
// structurally: a Character's private play history must never reach the
// Player-level Trailer Face.
//
// The proof is that no code path exists -- playerprofile reads only
// player_profile_* tables, and character_story_events is Character-scoped
// and joined by none of them. This test is the regression guard that keeps
// it that way, because the failure mode of a future refactor here is a
// privacy leak rather than a broken build.
func TestKernel75TrailerFaceCarriesNoStoryEvents(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var refs int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.view_column_usage
		WHERE table_name = 'character_story_events'
	`).Scan(&refs); err != nil {
		t.Fatalf("inspect view usage: %v", err)
	}
	if refs != 0 {
		t.Fatalf("character_story_events must not be exposed through any view, found %d references", refs)
	}
}
