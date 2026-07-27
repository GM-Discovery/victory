package merchant

import (
	"context"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/messages"
	"victory/backend/internal/projection"
	"victory/backend/internal/showings"
	"victory/backend/internal/shows"
	"victory/backend/internal/tutorial"
	"victory/backend/internal/world"
)

// TestKernel74TutorialTailEndToEnd is the closed-loop proof for Kernel 74's
// PASS standard: Kessa completion reveals the door for one Player and one
// Character only, a freeform intention is stored verbatim and reported to
// Directors+ without creating a GO, Ra's guided dialogue gates on required
// topics, and Leave Ra moves ONLY the caller onto a participant-local
// projection while the shared Show Scene stays exactly where it was.
//
// The throwaway venue mirrors prepare_end_to_end_test.go's reasoning:
// `go test ./...` runs packages concurrently against one shared victory_test
// database, and holding a rehearsal session open on the real catharsis row
// would race another package's most-recent-session lookup.
func TestKernel74TutorialTailEndToEnd(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	directorUserID := mustUser("k74_director_" + suffix)
	producerUserID := mustUser("k74_producer_" + suffix)
	playerAUserID := mustUser("k74_player_a_" + suffix)
	playerBUserID := mustUser("k74_player_b_" + suffix)
	audienceUserID := mustUser("k74_audience_" + suffix)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role) VALUES ($1, $2, 'director')
	`, locationID, directorUserID); err != nil {
		t.Fatalf("grant director role: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K74 Production "+suffix, "k74-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K74 Run "+suffix, "k74-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k74-show-"+suffix, "K74 Show", directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}

	for _, r := range []struct{ user, role string }{
		{playerAUserID, "player"}, {playerBUserID, "player"}, {audienceUserID, "audience"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id) VALUES ($1, $2, $3, $4)
		`, showRunID, r.user, r.role, directorUserID); err != nil {
			t.Fatalf("insert roster row for %s: %v", r.role, err)
		}
	}

	mustCharacter := func(owner, name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
		`, owner, locationID, name).Scan(&id); err != nil {
			t.Fatalf("insert character %s: %v", name, err)
		}
		return id
	}
	characterAID := mustCharacter(playerAUserID, "K74 Trang")
	characterBID := mustCharacter(playerBUserID, "K74 Kai")
	// A SECOND Character for Player A, used to prove S5.2: switching the
	// selected Character must not carry the first Character's progress.
	characterA2ID := mustCharacter(playerAUserID, "K74 Trang Understudy")

	setSelectedCharacter := func(user, character string) {
		if _, err := pool.Exec(ctx, `
			UPDATE show_run_roster_members SET character_card_id = $2 WHERE show_run_id = $1 AND user_id = $3
		`, showRunID, character, user); err != nil {
			t.Fatalf("select character: %v", err)
		}
	}
	setSelectedCharacter(playerAUserID, characterAID)
	setSelectedCharacter(playerBUserID, characterBID)

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `DELETE FROM participant_local_projections WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_tutorial_progress WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_freeform_submissions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_dialogue_topic_views WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM messages WHERE session_id IN (SELECT id FROM sessions WHERE show_id = $1)`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM sessions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM venues WHERE slug = $1`, "k74-stage-"+suffix)
		_, _ = pool.Exec(cctx, `
			DELETE FROM stage_element_bindings WHERE participant_interaction_id IN (
				SELECT id FROM participant_interactions WHERE show_scene_placement_id IN (
					SELECT id FROM show_scene_placements WHERE show_id = $1)
			)
		`, showID)
		_, _ = pool.Exec(cctx, `
			DELETE FROM scene_stage_elements WHERE scene_id IN (SELECT id FROM scenes WHERE location_id = $1 AND slug = 'courtyard')
			  AND show_scene_placement_id IS NULL AND label IN ('Kessa', 'Locked Courtyard Door')
		`, locationID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_interactions WHERE show_scene_placement_id IN (SELECT id FROM show_scene_placements WHERE show_id = $1)`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_scene_placements WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM character_cards WHERE id IN ($1, $2, $3)`, characterAID, characterBID, characterA2ID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_run_roster_members WHERE show_run_id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM shows WHERE id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_runs WHERE id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(cctx, `DELETE FROM location_memberships WHERE user_id = $1`, directorUserID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2, $3, $4, $5)`,
			directorUserID, producerUserID, playerAUserID, playerBUserID, audienceUserID)
	})

	// --- Prepare the whole tutorial tail with one idempotent action -------

	prep, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("PrepareLockedCourtyardOpening: %v", err)
	}
	if !prep.Ready {
		t.Fatalf("expected Ready=true, gaps=%v", prep.Gaps)
	}
	if prep.DoorInteractionID == "" || prep.DoorBindingID == "" || prep.RaInteractionID == "" {
		t.Fatalf("expected the Kernel 74 tail to be prepared, got %+v", prep)
	}
	if !prep.RaDialoguePacketFound || !prep.TutorialHandoffSceneFound {
		t.Fatal("expected Ra's packet and the tutorial-handoff Scene to be found (seeded by migration 066)")
	}

	// Idempotency: a second Prepare creates nothing new.
	prep2, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("second PrepareLockedCourtyardOpening: %v", err)
	}
	if prep2.DoorInteractionCreated || prep2.DoorElementCreated || prep2.DoorBindingCreated || prep2.RaInteractionCreated {
		t.Fatalf("second Prepare should have created nothing, got %+v", prep2)
	}

	// --- Live session on a throwaway venue --------------------------------

	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = $2 WHERE id = $1`, showID, prep.PlacementID); err != nil {
		t.Fatalf("set current placement: %v", err)
	}
	var lotID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM lots WHERE location_id = $1 AND slug = 'main-lot'`, locationID).Scan(&lotID); err != nil {
		t.Fatalf("lookup lot: %v", err)
	}
	venueSlug := "k74-stage-" + suffix
	var venueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public)
		VALUES ($1, 'K74 Stage', $2, 'plaza',
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
		{playerAUserID, "cast"}, {playerBUserID, "cast"},
		{directorUserID, "director"}, {producerUserID, "producer"}, {audienceUserID, "audience"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role) VALUES ($1, $2, $3::location_role)
		`, sessionID, p.user, p.role); err != nil {
			t.Fatalf("insert session participant %s: %v", p.role, err)
		}
	}

	// doorElementFor reports whether the door hotspot appears in a viewer's
	// own snapshot -- the actual S6.2 reveal test, since the gate works by
	// omitting the element entirely rather than flagging it hidden.
	doorElementFor := func(role, userID string) *world.PlacedElement {
		t.Helper()
		snap, err := world.LoadVenueSnapshot(ctx, pool, role, userID, venueSlug)
		if err != nil {
			t.Fatalf("LoadVenueSnapshot(%s): %v", role, err)
		}
		for i := range snap.Elements {
			if snap.Elements[i].Name == "Locked Courtyard Door" {
				return &snap.Elements[i]
			}
		}
		return nil
	}

	// --- S16.3: the door is hidden before Kessa completion ----------------

	if doorElementFor("cast", playerAUserID) != nil {
		t.Fatal("door hotspot must NOT be in a Player's snapshot before Kessa completion")
	}
	if doorElementFor("audience", audienceUserID) != nil {
		t.Fatal("door hotspot must never appear for an Audience viewer")
	}
	// Backstage authoring bypass: the Director positions the hotspot without
	// having played the tutorial.
	if doorElementFor("director", directorUserID) == nil {
		t.Fatal("Directors+ must see the door hotspot for authoring/preview regardless of Player progress")
	}

	// S13: the milestone gates INVOCATION as well as discovery. Knowing the
	// door's interaction id -- from another Character, a shared screen, a
	// replayed request -- must not let a Player who has not finished with
	// Kessa reach the door.
	if _, _, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, "I try the handle.", "forged-1"); err == nil {
		t.Fatal("submitting a door intention before Kessa completion must be refused server-side")
	} else if err.Error() != "milestone_required" {
		t.Fatalf("pre-Kessa submit error = %q, want milestone_required", err.Error())
	}
	if _, err := OpenFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID); err == nil {
		t.Fatal("even opening the door prompt before Kessa completion must be refused")
	}
	var strayCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM participant_freeform_submissions WHERE show_id = $1`, showID).Scan(&strayCount); err != nil {
		t.Fatalf("count stray submissions: %v", err)
	}
	if strayCount != 0 {
		t.Fatalf("a refused submission must store nothing, got %d rows", strayCount)
	}

	// --- S16.2/S16.3: Kessa completion, no purchase required --------------

	progress, err := CompleteKessaIntro(ctx, pool, playerAUserID, prep.InteractionID)
	if err != nil {
		t.Fatalf("CompleteKessaIntro: %v", err)
	}
	if len(progress) != 1 || progress[0] != tutorial.MilestoneKessaIntroCompleted {
		t.Fatalf("progress after Kessa = %v, want exactly [kessa_intro_completed]", progress)
	}
	// Idempotent.
	if _, err := CompleteKessaIntro(ctx, pool, playerAUserID, prep.InteractionID); err != nil {
		t.Fatalf("repeat CompleteKessaIntro: %v", err)
	}
	var progressRows int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM participant_tutorial_progress
		WHERE show_id = $1 AND user_id = $2 AND milestone_key = 'kessa_intro_completed'
	`, showID, playerAUserID).Scan(&progressRows); err != nil {
		t.Fatalf("count progress: %v", err)
	}
	if progressRows != 1 {
		t.Fatalf("expected exactly 1 kessa milestone row after a repeat call, got %d", progressRows)
	}

	door := doorElementFor("cast", playerAUserID)
	if door == nil {
		t.Fatal("door hotspot must appear for Player A after Kessa completion")
	}
	if door.ElementType != "scene_interaction_hotspot" {
		t.Fatalf("door element_type = %q, want scene_interaction_hotspot", door.ElementType)
	}
	binding, ok := door.Data["binding"].(map[string]any)
	if !ok || binding["participant_interaction_id"] != prep.DoorInteractionID {
		t.Fatalf("door binding = %v, want the prepared door interaction", door.Data["binding"])
	}
	if binding["interaction_type"] != InteractionTypeFreeformSubmission {
		t.Fatalf("door binding interaction_type = %v, want freeform_submission", binding["interaction_type"])
	}
	if _, hasWidth := door.Data["width"]; !hasWidth {
		t.Fatal("door hotspot must carry a normalized width so it can cover the door art")
	}

	// --- S16.3: Player B is unaffected ------------------------------------

	if doorElementFor("cast", playerBUserID) != nil {
		t.Fatal("Player A completing Kessa must not reveal the door to Player B")
	}

	// --- S5.2: switching Character does not carry progress ----------------

	setSelectedCharacter(playerAUserID, characterA2ID)
	if doorElementFor("cast", playerAUserID) != nil {
		t.Fatal("Character A's Kessa completion must not unlock the door for a different Character of the same Player")
	}
	setSelectedCharacter(playerAUserID, characterAID)
	if doorElementFor("cast", playerAUserID) == nil {
		t.Fatal("switching back to the completed Character must restore the door")
	}

	// --- S16.4/S16.5: the freeform door intention --------------------------

	if _, _, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, "   ", "empty-key"); err == nil {
		t.Fatal("expected a whitespace-only submission to be rejected")
	}
	if _, _, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, strings.Repeat("x", 5000), "long-key"); err == nil {
		t.Fatal("expected an oversized submission to be rejected")
	}

	// A complete sentence, deliberately -- S1.4 exists because "You start to
	// <this>" would be ungrammatical nonsense.
	intention := `I brace my shoulder against the oak and shove, hoping the frame is older than the door.`
	result, recipients, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, intention, "key-1")
	if err != nil {
		t.Fatalf("SubmitFreeform: %v", err)
	}
	if result.SubmittedText != intention {
		t.Fatalf("stored text = %q, want the exact submitted text %q", result.SubmittedText, intention)
	}
	if result.AlreadySubmitted {
		t.Fatal("first submission must not report already_submitted")
	}
	if result.NextInteractionID != prep.RaInteractionID {
		t.Fatalf("next interaction = %q, want Ra's interaction %q", result.NextInteractionID, prep.RaInteractionID)
	}
	// S1.4: narration and the quote are separate fields; the server never
	// concatenates authored text onto the Player's sentence.
	if strings.Contains(result.Narration, intention) || strings.Contains(result.Interruption, intention) {
		t.Fatal("authored narration must not embed the Player's text server-side")
	}

	// --- S16.6: the Directors+ note ---------------------------------------

	if len(recipients) == 0 {
		t.Fatal("expected at least one Directors+ note recipient")
	}
	directorNotes, err := messages.LoadBackstageNotesForSession(ctx, pool, sessionID, directorUserID)
	if err != nil {
		t.Fatalf("load director notes: %v", err)
	}
	if len(directorNotes) != 1 {
		t.Fatalf("director should see exactly 1 backstage note, got %d", len(directorNotes))
	}
	if !strings.Contains(directorNotes[0].Body, intention) {
		t.Fatalf("note body must quote the intention, got %q", directorNotes[0].Body)
	}
	if !strings.Contains(directorNotes[0].Subject, "Unresolved Door Intention") {
		t.Fatalf("note subject = %q, want the authored subject", directorNotes[0].Subject)
	}
	producerNotes, err := messages.LoadBackstageNotesForSession(ctx, pool, sessionID, producerUserID)
	if err != nil {
		t.Fatalf("load producer notes: %v", err)
	}
	if len(producerNotes) != 1 {
		t.Fatalf("producer should see the note too, got %d", len(producerNotes))
	}

	// S8.3 leakage: neither Player nor Audience is ever a recipient.
	for _, leaker := range []struct {
		name, userID string
	}{
		{"submitting player", playerAUserID},
		{"other player", playerBUserID},
		{"audience", audienceUserID},
	} {
		notes, err := messages.LoadBackstageNotesForSession(ctx, pool, sessionID, leaker.userID)
		if err != nil {
			t.Fatalf("load notes for %s: %v", leaker.name, err)
		}
		if len(notes) != 0 {
			t.Fatalf("%s must not receive a backstage note, got %d", leaker.name, len(notes))
		}
		if messages.IsBackstageNoteRole("cast") || messages.IsBackstageNoteRole("audience") {
			t.Fatal("cast and audience must not be backstage-note roles")
		}
	}

	// --- S5.3: retries duplicate neither the intention nor the note --------

	retry, _, err := SubmitFreeform(ctx, pool, playerAUserID, prep.DoorInteractionID, "A COMPLETELY DIFFERENT PLAN", "key-2-fresh")
	if err != nil {
		t.Fatalf("retry SubmitFreeform: %v", err)
	}
	if !retry.AlreadySubmitted {
		t.Fatal("a second submission must resolve to the original, not create a new one")
	}
	if retry.SubmittedText != intention {
		t.Fatalf("retry returned %q, want the ORIGINAL committed words", retry.SubmittedText)
	}
	var submissionCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM participant_freeform_submissions WHERE show_id = $1`, showID).Scan(&submissionCount); err != nil {
		t.Fatalf("count submissions: %v", err)
	}
	if submissionCount != 1 {
		t.Fatalf("expected exactly 1 stored submission after a retry, got %d", submissionCount)
	}
	directorNotes, err = messages.LoadBackstageNotesForSession(ctx, pool, sessionID, directorUserID)
	if err != nil {
		t.Fatalf("reload director notes: %v", err)
	}
	if len(directorNotes) != 1 {
		t.Fatalf("a retry must not create a second note, got %d", len(directorNotes))
	}

	// --- S16.9/S16.11: Ra's guided dialogue --------------------------------

	state, err := OpenDialogue(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("OpenDialogue: %v", err)
	}
	if state.NPCName != "Ra" {
		t.Fatalf("npc name = %q, want Ra", state.NPCName)
	}
	if state.CanLeave {
		t.Fatal("Leave Ra must be locked before the required topics are viewed")
	}
	topicState := func(key string) (seen, unlocked, required, found bool) {
		for _, tv := range state.Topics {
			if tv.TopicKey == key {
				return tv.Seen, tv.Unlocked, tv.Required, true
			}
		}
		return false, false, false, false
	}

	// S9.3's dependency pattern: crown-bet is locked until why-looking.
	if _, unlocked, _, found := topicState("crown-bet"); !found || unlocked {
		t.Fatal("crown-bet must start locked behind why-looking")
	}
	if _, unlocked, _, found := topicState("why-locked"); !found || !unlocked {
		t.Fatal("why-locked must be open from the start")
	}

	// S13: posting a locked topic directly is refused, and its authored
	// response never leaves the server.
	if _, err := ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, "crown-bet"); err == nil {
		t.Fatal("expected a locked topic to be refused server-side")
	}
	// S13: posting Leave without the required topics is refused.
	if _, err := LeaveDialogue(ctx, pool, playerAUserID, prep.RaInteractionID); err == nil {
		t.Fatal("expected Leave Ra to be refused before required topics are viewed")
	}

	// Ask out of default order -- who-are-you is optional and open.
	state, err = ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, "who-are-you")
	if err != nil {
		t.Fatalf("ReadTopic(who-are-you): %v", err)
	}
	if !strings.Contains(state.CurrentResponse, "Ra") {
		t.Fatalf("who-are-you response should name Ra, got %q", state.CurrentResponse)
	}

	state, err = ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, "why-looking")
	if err != nil {
		t.Fatalf("ReadTopic(why-looking): %v", err)
	}
	topicState = func(key string) (seen, unlocked, required, found bool) {
		for _, tv := range state.Topics {
			if tv.TopicKey == key {
				return tv.Seen, tv.Unlocked, tv.Required, true
			}
		}
		return false, false, false, false
	}
	if _, unlocked, _, _ := topicState("crown-bet"); !unlocked {
		t.Fatal("crown-bet must unlock once why-looking has been read")
	}
	if state.CanLeave {
		t.Fatal("Leave Ra must stay locked while crown-bet is unread")
	}

	state, err = ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, "crown-bet")
	if err != nil {
		t.Fatalf("ReadTopic(crown-bet): %v", err)
	}
	// S9.4 canonical material.
	if !strings.Contains(strings.ToLower(state.CurrentResponse), "turtle") ||
		!strings.Contains(strings.ToLower(state.CurrentResponse), "crown") {
		t.Fatalf("crown-bet response must deliver the Turtle-continent crown challenge, got %q", state.CurrentResponse)
	}
	if !state.CanLeave {
		t.Fatal("Leave Ra must unlock once both required topics are read, without every optional topic")
	}

	// S5.3: re-reading a seen topic is idempotent.
	if _, err := ReadTopic(ctx, pool, playerAUserID, prep.RaInteractionID, "crown-bet"); err != nil {
		t.Fatalf("re-read crown-bet: %v", err)
	}
	var viewCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM participant_dialogue_topic_views WHERE show_id = $1 AND user_id = $2`, showID, playerAUserID).Scan(&viewCount); err != nil {
		t.Fatalf("count topic views: %v", err)
	}
	if viewCount != 3 {
		t.Fatalf("expected 3 distinct topic views after a repeat read, got %d", viewCount)
	}

	// S9.5: progress survives a fresh open (the refresh case).
	reopened, err := OpenDialogue(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("reopen dialogue: %v", err)
	}
	if !reopened.CanLeave {
		t.Fatal("Leave Ra must remain available after a reopen -- progress is stored, not client state")
	}
	seenAfterReopen := 0
	for _, tv := range reopened.Topics {
		if tv.Seen {
			seenAfterReopen++
		}
	}
	if seenAfterReopen != 3 {
		t.Fatalf("expected 3 topics still marked seen after reopen, got %d", seenAfterReopen)
	}

	// --- S16.12/S16.13/S16.14: Leave Ra ------------------------------------

	leave, err := LeaveDialogue(ctx, pool, playerAUserID, prep.RaInteractionID)
	if err != nil {
		t.Fatalf("LeaveDialogue: %v", err)
	}
	// S10.1/S10.2: the reveal describes a concealed courtyard-side mechanism
	// and must not retcon an ordinary visible padlock.
	closing := strings.ToLower(leave.State.ClosingNarration)
	if !strings.Contains(closing, "iron plate") || !strings.Contains(closing, "recessed") {
		t.Fatalf("closing narration must reveal a concealed courtyard-side mechanism, got %q", leave.State.ClosingNarration)
	}
	if strings.Contains(closing, "padlock") {
		t.Fatal("the reveal must not contradict the door's 'no obvious lock' description")
	}
	if leave.Projection.SceneSlug != "tutorial-handoff" {
		t.Fatalf("projection scene = %q, want tutorial-handoff", leave.Projection.SceneSlug)
	}

	// --- S16.15: the shared current Scene did NOT move ---------------------

	var currentAfterLeave *string
	if err := pool.QueryRow(ctx, `SELECT current_show_scene_placement_id::text FROM shows WHERE id = $1`, showID).Scan(&currentAfterLeave); err != nil {
		t.Fatalf("reload current placement: %v", err)
	}
	if currentAfterLeave == nil || *currentAfterLeave != prep.PlacementID {
		t.Fatalf("shared current Scene must be unchanged, got %v want %v", currentAfterLeave, prep.PlacementID)
	}

	// --- S16.14/S16.16/S16.17: only the caller moved -----------------------

	snapA, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerAUserID, venueSlug)
	if err != nil {
		t.Fatalf("snapshot A after leave: %v", err)
	}
	if snapA.Session.LocalProjection == nil {
		t.Fatal("Player A must be on their own local projection after Leave Ra")
	}
	if snapA.Session.LocalProjection.SceneSlug != "tutorial-handoff" {
		t.Fatalf("A's projection = %q, want tutorial-handoff", snapA.Session.LocalProjection.SceneSlug)
	}
	// The snapshot still reports the unchanged shared Scene alongside the
	// projection -- this is what makes S16.15 assertable from the Player's
	// own payload, not just from the database.
	if snapA.Session.CurrentShowScenePlacementID != prep.PlacementID {
		t.Fatalf("A's snapshot must still report the shared Courtyard, got %q", snapA.Session.CurrentShowScenePlacementID)
	}
	if snapA.Session.LocalProjection.BackdropURL == "" {
		t.Fatal("the handoff projection must carry its authored backdrop")
	}
	for _, el := range snapA.Elements {
		if el.Name == "Kessa" {
			t.Fatal("Player A on the handoff projection must not still see the Courtyard's composition")
		}
	}

	snapB, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerBUserID, venueSlug)
	if err != nil {
		t.Fatalf("snapshot B after A's leave: %v", err)
	}
	if snapB.Session.LocalProjection != nil {
		t.Fatal("Player B must NOT have been moved by Player A leaving Ra")
	}
	foundKessaForB := false
	for _, el := range snapB.Elements {
		if el.Name == "Kessa" {
			foundKessaForB = true
		}
	}
	if !foundKessaForB {
		t.Fatal("Player B must remain on the shared Courtyard")
	}
	snapAudience, err := world.LoadVenueSnapshot(ctx, pool, "audience", audienceUserID, venueSlug)
	if err != nil {
		t.Fatalf("audience snapshot: %v", err)
	}
	if snapAudience.Session.LocalProjection != nil {
		t.Fatal("the Audience projection must be untouched by a Player's local projection")
	}

	// Persistence across "refresh": a second snapshot read still resolves it.
	snapAAgain, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerAUserID, venueSlug)
	if err != nil {
		t.Fatalf("second snapshot A: %v", err)
	}
	if snapAAgain.Session.LocalProjection == nil {
		t.Fatal("the local projection must survive a refresh -- it is a row, not client state")
	}

	// S5.3: Leave is idempotent; no second projection row.
	if _, err := LeaveDialogue(ctx, pool, playerAUserID, prep.RaInteractionID); err != nil {
		t.Fatalf("repeat LeaveDialogue: %v", err)
	}
	var activeProjections int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM participant_local_projections WHERE show_id = $1 AND cleared_at IS NULL
	`, showID).Scan(&activeProjections); err != nil {
		t.Fatalf("count projections: %v", err)
	}
	if activeProjections != 1 {
		t.Fatalf("expected exactly 1 active projection after a repeated Leave, got %d", activeProjections)
	}

	// --- S5.2 regression: switching Character leaves the projection --------
	//
	// This was a real bug found in live play. The projection was keyed on
	// (user, show) and ignored character_card_id, so a Player who finished
	// the tutorial with one Character and then switched stayed stranded on
	// the handoff map -- with a Character who had never met Kessa, never
	// tried the door, and never spoke to Ra. The original Character-switch
	// assertion above only covered the door hotspot, not the projection,
	// which is exactly how this got through.
	setSelectedCharacter(playerAUserID, characterA2ID)
	snapSwitched, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerAUserID, venueSlug)
	if err != nil {
		t.Fatalf("snapshot after Character switch: %v", err)
	}
	if snapSwitched.Session.LocalProjection != nil {
		t.Fatal("switching to a Character who never played the tutorial must return the Player to the shared stage")
	}
	foundKessaAfterSwitch := false
	for _, el := range snapSwitched.Elements {
		if el.Name == "Kessa" {
			foundKessaAfterSwitch = true
		}
	}
	if !foundKessaAfterSwitch {
		t.Fatal("the switched-to Character must see the shared Courtyard, including Kessa")
	}

	// Switching back restores the finished Character's own projection --
	// the record belongs to the Character who earned it and is not destroyed
	// by looking away.
	setSelectedCharacter(playerAUserID, characterAID)
	snapBack, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerAUserID, venueSlug)
	if err != nil {
		t.Fatalf("snapshot after switching back: %v", err)
	}
	if snapBack.Session.LocalProjection == nil {
		t.Fatal("switching back to the finished Character must restore their handoff projection")
	}

	// --- S16.18: a later shared Scene clears the projection ----------------

	// Stage a second Scene and fly to it as the Director.
	var secondSceneID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM scenes WHERE location_id = $1 AND slug = 'tutorial-handoff'`, locationID).Scan(&secondSceneID); err != nil {
		t.Fatalf("lookup a second scene: %v", err)
	}
	var secondPlacementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, venue_id, sort_order, created_by_user_id)
		VALUES ($1, $2, $3, 99, $4) RETURNING id::text
	`, showID, secondSceneID, venueID, directorUserID).Scan(&secondPlacementID); err != nil {
		t.Fatalf("insert second placement: %v", err)
	}
	if _, err := shows.SetCurrentScenePlacement(ctx, pool, directorUserID, showID, secondPlacementID); err != nil {
		t.Fatalf("Director flies a later shared Scene: %v", err)
	}

	snapACleared, err := world.LoadVenueSnapshot(ctx, pool, "cast", playerAUserID, venueSlug)
	if err != nil {
		t.Fatalf("snapshot A after shared advance: %v", err)
	}
	if snapACleared.Session.LocalProjection != nil {
		t.Fatal("advancing the shared Scene must clear the Player's local projection")
	}
	if snapACleared.Session.CurrentShowScenePlacementID != secondPlacementID {
		t.Fatalf("Player A must now resolve the Director's Scene, got %q", snapACleared.Session.CurrentShowScenePlacementID)
	}

	// --- S11.5: the backstage status list carries no denominator -----------

	list, err := tutorial.ListShowProgress(ctx, pool, showID)
	if err != nil {
		t.Fatalf("ListShowProgress: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected the backstage status list to report Player A")
	}
	foundA := false
	for _, entry := range list {
		if entry.CharacterCardID == characterAID {
			foundA = true
			if entry.Status != "Tutorial handoff entered" {
				t.Fatalf("Character A status = %q, want 'Tutorial handoff entered'", entry.Status)
			}
		}
	}
	if !foundA {
		t.Fatal("Character A must appear in the backstage status list")
	}
}

// TestKernel74AnonymousAndCapabilityRefusals covers the S13 authority floor
// that does not need the full fixture: no caller, and a venue with the
// capability flag off.
func TestKernel74AnonymousAndCapabilityRefusals(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	fakeInteraction := "00000000-0000-0000-0000-000000000000"

	for name, call := range map[string]func() error{
		"CompleteKessaIntro": func() error {
			_, err := CompleteKessaIntro(ctx, pool, "", fakeInteraction)
			return err
		},
		"SubmitFreeform": func() error {
			_, _, err := SubmitFreeform(ctx, pool, "", fakeInteraction, "anything", "k")
			return err
		},
		"OpenDialogue": func() error {
			_, err := OpenDialogue(ctx, pool, "", fakeInteraction)
			return err
		},
		"ReadTopic": func() error {
			_, err := ReadTopic(ctx, pool, "", fakeInteraction, "who-are-you")
			return err
		},
		"LeaveDialogue": func() error {
			_, err := LeaveDialogue(ctx, pool, "", fakeInteraction)
			return err
		},
	} {
		if err := call(); err == nil {
			t.Fatalf("%s must refuse an anonymous caller", name)
		} else if err.Error() != "not_authenticated" {
			t.Fatalf("%s anonymous error = %q, want not_authenticated", name, err.Error())
		}
	}

	// Fail-closed capability: an unknown venue slug is never enabled.
	enabled, err := projection.VenueLocalProjectionEnabled(ctx, pool, "no-such-venue-"+time.Now().Format("150405"))
	if err != nil {
		t.Fatalf("VenueLocalProjectionEnabled: %v", err)
	}
	if enabled {
		t.Fatal("an unknown venue must never report the local-projection capability as enabled")
	}
}

// TestKernel74MilestoneValidation proves the milestone allowlist is enforced
// in Go as well as by the CHECK constraint, so a typo in a binding fails as
// a clean error instead of hiding an element from every Player forever.
func TestKernel74MilestoneValidation(t *testing.T) {
	if tutorial.IsMilestone("kessa_intro_completedd") {
		t.Fatal("a typo'd milestone must not validate")
	}
	if !tutorial.IsMilestone(tutorial.MilestoneKessaIntroCompleted) {
		t.Fatal("the real milestone must validate")
	}
	if tutorial.IsMilestone("") {
		t.Fatal("an empty milestone must not validate as a real one")
	}
}
