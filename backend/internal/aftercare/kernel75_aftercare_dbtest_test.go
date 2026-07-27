package aftercare

import (
	"context"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// TestKernel75AftercareLifecycle proves S8's whole contract: partial
// submission, server-side drafts that are never mistaken for submissions,
// an explicitly confirmed Skip, and a consecutive-skip count that is
// computed rather than stored -- and therefore cannot be forged, cannot be
// decremented, and resets by construction when a Player submits.
func TestKernel75AftercareLifecycle(t *testing.T) {
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
			t.Fatalf("insert user: %v", err)
		}
		return id
	}
	ownerUserID := mustUser("k75ac_owner_" + suffix)
	directorUserID := mustUser("k75ac_director_" + suffix)

	var productionID, showRunID, showID, characterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K75AC Production "+suffix, "k75ac-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K75AC Run "+suffix, "k75ac-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k75ac-show-"+suffix, "K75AC Show", directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, ownerUserID, locationID, "K75AC Character").Scan(&characterID); err != nil {
		t.Fatalf("insert character: %v", err)
	}

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `DELETE FROM aftercare_skips WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM aftercare_response_drafts WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM aftercare_submissions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM character_cards WHERE id = $1`, characterID)
		_, _ = pool.Exec(cctx, `DELETE FROM shows WHERE id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_runs WHERE id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2)`, ownerUserID, directorUserID)
	})

	p := Participation{
		UserID:          ownerUserID,
		CharacterCardID: characterID,
		ShowID:          showID,
		ShowRunID:       showRunID,
	}

	// --- S8.1: the three locked prompts ------------------------------------

	state, err := Offer(ctx, pool, p)
	if err != nil {
		t.Fatalf("Offer: %v", err)
	}
	if len(state.Prompts) != 3 {
		t.Fatalf("expected exactly the three locked prompts, got %d", len(state.Prompts))
	}
	if state.Draft != nil || state.Submission != nil || state.Resolved {
		t.Fatal("a fresh Player has no draft, no submission, and is unresolved")
	}

	// --- S8.2: drafts persist and are NOT submissions -----------------------

	if _, err := SaveDraft(ctx, pool, p, map[string]string{"favorite_moments": "The gate opening."}); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	state, err = Offer(ctx, pool, p)
	if err != nil {
		t.Fatalf("Offer after draft: %v", err)
	}
	if state.Draft == nil || state.Draft.Responses["favorite_moments"] != "The gate opening." {
		t.Fatalf("draft must survive and round-trip, got %+v", state.Draft)
	}
	if state.Submission != nil || state.Resolved {
		t.Fatal("a draft must never count as a submission")
	}
	if state.ConsecutiveSkips != 0 {
		t.Fatal("saving a draft must not touch the skip count")
	}

	// A draft revision overwrites rather than accumulating.
	if _, err := SaveDraft(ctx, pool, p, map[string]string{"favorite_moments": "Actually, meeting Kessa."}); err != nil {
		t.Fatalf("SaveDraft revision: %v", err)
	}
	var draftRows int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM aftercare_response_drafts WHERE show_id = $1`, showID).Scan(&draftRows); err != nil {
		t.Fatalf("count drafts: %v", err)
	}
	if draftRows != 1 {
		t.Fatalf("expected exactly 1 draft row, got %d", draftRows)
	}

	// --- S8.4/S1.14: Skip requires explicit confirmation --------------------

	if err := Skip(ctx, pool, p, false); err == nil {
		t.Fatal("an unconfirmed skip must be refused, not trusted")
	} else if err.Error() != "confirmation_required" {
		t.Fatalf("expected confirmation_required, got %q", err.Error())
	}
	var skipRowsAfterRefusal int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM aftercare_skips WHERE user_id = $1`, ownerUserID).Scan(&skipRowsAfterRefusal); err != nil {
		t.Fatalf("count skips: %v", err)
	}
	if skipRowsAfterRefusal != 0 {
		t.Fatal("a refused skip must write nothing")
	}

	// --- S8.5: consecutive skips accumulate ---------------------------------

	if err := Skip(ctx, pool, p, true); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	count, err := ConsecutiveSkips(ctx, pool, ownerUserID)
	if err != nil {
		t.Fatalf("ConsecutiveSkips: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 consecutive skip, got %d", count)
	}

	if err := Skip(ctx, pool, p, true); err != nil {
		t.Fatalf("second Skip: %v", err)
	}
	count, _ = ConsecutiveSkips(ctx, pool, ownerUserID)
	if count != 2 {
		t.Fatalf("expected 2 consecutive skips, got %d", count)
	}

	// S8.5's negative list -- closing the panel, refreshing, abandoning a
	// draft -- is satisfied structurally: none of them has a route that
	// writes aftercare_skips. Offering and re-offering proves the read path
	// itself never increments.
	for i := 0; i < 3; i++ {
		if _, err := Offer(ctx, pool, p); err != nil {
			t.Fatalf("Offer loop: %v", err)
		}
	}
	if _, err := SaveDraft(ctx, pool, p, map[string]string{"next_session": "More Ra."}); err != nil {
		t.Fatalf("SaveDraft after skips: %v", err)
	}
	count, _ = ConsecutiveSkips(ctx, pool, ownerUserID)
	if count != 2 {
		t.Fatalf("reading and drafting must not change the skip count, got %d", count)
	}

	// --- S8.3: a partial submission resets the count to zero ----------------

	// Deliberately partial: only one of three prompts answered.
	submission, err := Submit(ctx, pool, p, map[string]string{"next_session": "I want to meet the person Ra mentioned."})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if submission.Responses["next_session"] == "" {
		t.Fatal("a partial submission must keep the answers it was given")
	}
	if submission.PromptSetVersion != PromptSetVersion {
		t.Fatalf("submission must record the prompt set version, got %d", submission.PromptSetVersion)
	}

	count, _ = ConsecutiveSkips(ctx, pool, ownerUserID)
	if count != 0 {
		t.Fatalf("S8.3: submitting must reset the consecutive skip count, got %d", count)
	}

	// The skip rows themselves are NOT deleted -- the history stays, only
	// the "consecutive" window moves. That is what makes the count
	// impossible to decrement.
	var totalSkips int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM aftercare_skips WHERE user_id = $1`, ownerUserID).Scan(&totalSkips); err != nil {
		t.Fatalf("count total skips: %v", err)
	}
	if totalSkips != 2 {
		t.Fatalf("skip history must be preserved, got %d rows", totalSkips)
	}

	// Submitting clears the draft, so stale thinking cannot outlive it.
	state, err = Offer(ctx, pool, p)
	if err != nil {
		t.Fatalf("Offer after submit: %v", err)
	}
	if state.Draft != nil {
		t.Fatal("submitting must clear the draft")
	}
	if state.Submission == nil || !state.Resolved {
		t.Fatal("expected a resolved submission")
	}

	// A later skip counts again from the new baseline.
	if err := Skip(ctx, pool, p, true); err != nil {
		t.Fatalf("Skip after submit: %v", err)
	}
	count, _ = ConsecutiveSkips(ctx, pool, ownerUserID)
	if count != 1 {
		t.Fatalf("expected the count to restart at 1 after a submission, got %d", count)
	}
}

func TestValidateResponsesRejectsUnknownKeys(t *testing.T) {
	// An unknown key is a version skew or a probe. Both deserve a visible
	// failure rather than silently dropping a Player's words.
	if _, err := ValidateResponses(map[string]string{"favorite_colour": "blue"}); err == nil {
		t.Fatal("unknown prompt keys must be refused")
	}
}

func TestValidateResponsesEnforcesLength(t *testing.T) {
	long := strings.Repeat("x", 5000)
	if _, err := ValidateResponses(map[string]string{"favorite_moments": long}); err == nil {
		t.Fatal("over-long responses must be refused")
	}
}

func TestValidateResponsesAcceptsEmpty(t *testing.T) {
	// S8.1: every field is optional, so an entirely empty submission is
	// valid -- that is what "Save and Close may submit partial responses"
	// means at its limit.
	out, err := ValidateResponses(map[string]string{})
	if err != nil {
		t.Fatalf("an empty response set must be valid: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no responses, got %v", out)
	}

	out, err = ValidateResponses(map[string]string{"favorite_moments": "   "})
	if err != nil {
		t.Fatalf("whitespace-only must be valid: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("whitespace-only answers must not be stored as content, got %v", out)
	}
}

func TestPromptSetIsTheThreeLockedPrompts(t *testing.T) {
	// S1.12 locks these three. A fourth prompt, or a reworded one, is a
	// product decision that must also bump PromptSetVersion -- this test is
	// the reminder.
	if len(PromptSetV1) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(PromptSetV1))
	}
	wantKeys := []string{"favorite_moments", "who_surprised", "next_session"}
	for i, want := range wantKeys {
		if PromptSetV1[i].Key != want {
			t.Errorf("prompt %d = %q, want %q", i, PromptSetV1[i].Key, want)
		}
	}
	// S1.12: no numerical ratings. Nothing here may carry a scale.
	for _, p := range PromptSetV1 {
		if strings.Contains(strings.ToLower(p.Label), "rate ") ||
			strings.Contains(p.Label, "1-5") || strings.Contains(p.Label, "1-10") {
			t.Errorf("Aftercare is qualitative; prompt %q looks like a rating", p.Key)
		}
	}
}
