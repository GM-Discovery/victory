package world

import (
	"context"
	"encoding/json"
	"testing"
)

// TestLoadVenueSnapshotFiltersRollDiceByAudience is the Kernel 86 §12/§15
// history-leak proof: a roll/dice Action stored with a restricted
// audienceMode must not appear in LoadVenueSnapshot's Actions for a viewer
// rollaudience.VisibleToViewer would reject, even though the pre-Kernel-86
// query returned every session/show-scoped action unfiltered.
func TestLoadVenueSnapshotFiltersRollDiceByAudience(t *testing.T) {
	pool := openWorldTestPool(t)
	director := insertWorldTestUser(t, pool, "wd_director")
	cohortMate := insertWorldTestUser(t, pool, "wd_mate")
	otherCohort := insertWorldTestUser(t, pool, "wd_other")
	audience := insertWorldTestUser(t, pool, "wd_audience")

	f := buildWorldFixture(t, pool, director)
	sessionID := insertSession(t, pool, f.venueID, f.showID, "live")

	ctx := context.Background()
	for userID, role := range map[string]string{
		director:    "director",
		cohortMate:  "cast",
		otherCohort: "cast",
		audience:    "audience",
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	var cohortID, otherCohortID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'wd-cohort-a-'||$2, 'Cohort A', $3::uuid)
		RETURNING id::text
	`, f.showID, testSuffix(t), director).Scan(&cohortID); err != nil {
		t.Fatalf("fixture cohort A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 2, 'wd-cohort-b-'||$2, 'Cohort B', $3::uuid)
		RETURNING id::text
	`, f.showID, testSuffix(t), director).Scan(&otherCohortID); err != nil {
		t.Fatalf("fixture cohort B: %v", err)
	}
	for _, assign := range []struct{ userID, cohortID string }{
		{cohortMate, cohortID},
		{otherCohort, otherCohortID},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_cohort_assignments (show_id, user_id, cohort_id, assigned_by_user_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
		`, f.showID, assign.userID, assign.cohortID, director); err != nil {
			t.Fatalf("fixture cohort assignment: %v", err)
		}
	}

	insertRoll := func(actorID, audienceMode, cohortID string) {
		target, _ := json.Marshal(map[string]any{"kind": "session", "id": sessionID})
		payload, _ := json.Marshal(map[string]any{"expression": "1d6", "total": 4, "visibility_mode": audienceMode})
		visibility, _ := json.Marshal(map[string]any{"audienceMode": audienceMode, "cohortId": cohortID, "toRoles": []string{}, "privateTo": []string{}})
		if _, err := pool.Exec(ctx, `
			INSERT INTO actions (session_id, show_id, moment_id, actor_id, type, target, payload, visibility)
			VALUES (
				$1, $2,
				COALESCE((SELECT MAX(moment_id) FROM actions WHERE session_id = $1), 0) + 1,
				$3, 'roll/dice', $4, $5, $6
			)
		`, sessionID, f.showID, actorID, target, payload, visibility); err != nil {
			t.Fatalf("insert roll/dice action (%s): %v", audienceMode, err)
		}
	}

	// cohortMate rolls a Cohort-restricted, a Director-only, and a Private
	// roll; director rolls one Show-wide roll for contrast.
	insertRoll(cohortMate, "cohort", cohortID)
	insertRoll(cohortMate, "director", "")
	insertRoll(cohortMate, "private", "")
	insertRoll(director, "show", "")

	countByMode := func(actions []Action, mode string) int {
		n := 0
		for _, a := range actions {
			if a.Type != "roll/dice" {
				continue
			}
			if m, _ := a.Visibility["audienceMode"].(string); m == mode {
				n++
			}
		}
		return n
	}

	// The other cohort's member: sees the Show roll only.
	snapOther, err := LoadVenueSnapshot(ctx, pool, "cast", otherCohort, f.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot(otherCohort): %v", err)
	}
	if got := countByMode(snapOther.Actions, "cohort"); got != 0 {
		t.Fatalf("other cohort member saw %d cohort-restricted rolls, want 0", got)
	}
	if got := countByMode(snapOther.Actions, "director"); got != 0 {
		t.Fatalf("other cohort member saw %d director-only rolls, want 0", got)
	}
	if got := countByMode(snapOther.Actions, "private"); got != 0 {
		t.Fatalf("other cohort member saw %d private rolls, want 0", got)
	}
	if got := countByMode(snapOther.Actions, "show"); got != 1 {
		t.Fatalf("other cohort member saw %d show rolls, want 1", got)
	}

	// The same cohort's member: sees the cohort roll (their own) but not
	// the Director-only or Private rolls.
	snapMateOwn, err := LoadVenueSnapshot(ctx, pool, "cast", cohortMate, f.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot(cohortMate): %v", err)
	}
	if got := countByMode(snapMateOwn.Actions, "cohort"); got != 1 {
		t.Fatalf("cohort mate (roller) saw %d cohort rolls, want 1", got)
	}
	if got := countByMode(snapMateOwn.Actions, "director"); got != 1 {
		t.Fatalf("roller should always see their own director-mode roll, got %d", got)
	}
	if got := countByMode(snapMateOwn.Actions, "private"); got != 1 {
		t.Fatalf("roller should always see their own private roll, got %d", got)
	}

	// The Director: sees the cohort roll (omniscient) and the director-mode
	// roll, but NOT the other user's Private roll -- private has no
	// exception, not even for Director+.
	snapDirector, err := LoadVenueSnapshot(ctx, pool, "director", director, f.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot(director): %v", err)
	}
	if got := countByMode(snapDirector.Actions, "cohort"); got != 1 {
		t.Fatalf("director saw %d cohort rolls, want 1 (director omniscience)", got)
	}
	if got := countByMode(snapDirector.Actions, "director"); got != 1 {
		t.Fatalf("director saw %d director-mode rolls, want 1", got)
	}
	if got := countByMode(snapDirector.Actions, "private"); got != 0 {
		t.Fatalf("director saw %d of another user's private rolls, want 0 (no exception for private)", got)
	}

	// Plain Audience viewer (no cohort, no backstage role): sees only the
	// Show roll, same as the other-cohort member.
	snapAudience, err := LoadVenueSnapshot(ctx, pool, "audience", audience, f.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot(audience): %v", err)
	}
	if got := countByMode(snapAudience.Actions, "show"); got != 1 {
		t.Fatalf("audience viewer saw %d show rolls, want 1", got)
	}
	if got := countByMode(snapAudience.Actions, "cohort") + countByMode(snapAudience.Actions, "director") + countByMode(snapAudience.Actions, "private"); got != 0 {
		t.Fatalf("audience viewer saw %d restricted rolls, want 0", got)
	}
}
