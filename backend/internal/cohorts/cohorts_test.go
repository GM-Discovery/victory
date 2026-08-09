package cohorts

import (
	"context"
	"sync"
	"testing"
)

func TestShowParticipantsBeginUngrouped(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_director")
	showRunID, showID := showFixture(t, pool, director)
	locationID := locationForShowRun(t, pool, showRunID)
	alice, _ := playerFixture(t, pool, director, showRunID, locationID, "co_alice")

	roster, err := ListRosterForShow(context.Background(), pool, director, showID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	if len(roster.Cohorts) != 0 {
		t.Fatalf("expected no cohorts yet, got %d", len(roster.Cohorts))
	}
	found := false
	for _, p := range roster.Ungrouped {
		if p.UserID == alice {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected alice in Ungrouped, roster=%+v", roster)
	}
}

func TestCohortSerialAllocationDeterministicAndCollisionSafe(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_serial_director")
	_, showID := showFixture(t, pool, director)
	ctx := context.Background()

	c1, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 1: %v", err)
	}
	if c1.SerialNumber != 1 || c1.Name != "Cohort 1" || c1.Slug != "cohort-1" {
		t.Fatalf("unexpected cohort 1: %+v", c1)
	}

	c2, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 2: %v", err)
	}
	if c2.SerialNumber != 2 {
		t.Fatalf("expected serial 2, got %d", c2.SerialNumber)
	}

	// Deleting (archiving) Cohort 2 must not free its serial for reuse.
	if err := ArchiveCohort(ctx, pool, director, showID, c2.ID); err != nil {
		t.Fatalf("archive cohort 2: %v", err)
	}
	c3, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 3: %v", err)
	}
	if c3.SerialNumber != 3 {
		t.Fatalf("expected the next cohort after archiving Cohort 2 to be serial 3 (no reuse), got %d", c3.SerialNumber)
	}
}

func TestConcurrentCohortCreationIsSerialSafe(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_concurrent_director")
	_, showID := showFixture(t, pool, director)
	ctx := context.Background()

	const n = 8
	var wg sync.WaitGroup
	serials := make([]int, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := CreateCohort(ctx, pool, director, showID)
			errs[i] = err
			if err == nil {
				serials[i] = c.SerialNumber
			}
		}(i)
	}
	wg.Wait()

	seen := map[int]bool{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent create %d failed: %v", i, err)
		}
		if seen[serials[i]] {
			t.Fatalf("duplicate serial number allocated concurrently: %d", serials[i])
		}
		seen[serials[i]] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct serials, got %d: %v", n, len(seen), serials)
	}
}

func TestParticipantAssignmentMovesAndReturnsToUngrouped(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_move_director")
	showRunID, showID := showFixture(t, pool, director)
	locationID := locationForShowRun(t, pool, showRunID)
	alice, _ := playerFixture(t, pool, director, showRunID, locationID, "co_move_alice")
	ctx := context.Background()

	cohortA, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort A: %v", err)
	}
	cohortB, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort B: %v", err)
	}

	if err := AssignParticipant(ctx, pool, director, showID, cohortA.ID, alice); err != nil {
		t.Fatalf("assign alice to cohort A: %v", err)
	}
	roster, err := ListRosterForShow(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	assertMember(t, roster, cohortA.ID, alice, true)
	assertMember(t, roster, cohortB.ID, alice, false)
	assertUngrouped(t, roster, alice, false)

	// Move A -> B: must not leave alice in both, or in Ungrouped.
	if err := AssignParticipant(ctx, pool, director, showID, cohortB.ID, alice); err != nil {
		t.Fatalf("move alice to cohort B: %v", err)
	}
	roster, err = ListRosterForShow(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	assertMember(t, roster, cohortA.ID, alice, false)
	assertMember(t, roster, cohortB.ID, alice, true)
	assertUngrouped(t, roster, alice, false)

	// Cohort -> Ungrouped.
	if err := UnassignParticipant(ctx, pool, director, showID, alice); err != nil {
		t.Fatalf("unassign alice: %v", err)
	}
	roster, err = ListRosterForShow(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("list roster: %v", err)
	}
	assertMember(t, roster, cohortA.ID, alice, false)
	assertMember(t, roster, cohortB.ID, alice, false)
	assertUngrouped(t, roster, alice, true)
}

func assertMember(t *testing.T, roster Roster, cohortID, userID string, want bool) {
	t.Helper()
	for _, c := range roster.Cohorts {
		if c.ID != cohortID {
			continue
		}
		got := false
		for _, m := range c.Members {
			if m.UserID == userID {
				got = true
			}
		}
		if got != want {
			t.Fatalf("cohort %s membership for %s: got %v, want %v", cohortID, userID, got, want)
		}
		return
	}
	t.Fatalf("cohort %s not found in roster", cohortID)
}

func assertUngrouped(t *testing.T, roster Roster, userID string, want bool) {
	t.Helper()
	got := false
	for _, p := range roster.Ungrouped {
		if p.UserID == userID {
			got = true
		}
	}
	if got != want {
		t.Fatalf("ungrouped membership for %s: got %v, want %v", userID, got, want)
	}
}

func TestNonDirectorCannotManageCohorts(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_auth_director")
	showRunID, showID := showFixture(t, pool, director)
	locationID := locationForShowRun(t, pool, showRunID)
	outsider := insertTestUser(t, pool, "co_auth_outsider")
	alice, _ := playerFixture(t, pool, director, showRunID, locationID, "co_auth_alice")
	ctx := context.Background()

	if _, err := CreateCohort(ctx, pool, outsider, showID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider create, got %v", err)
	}

	cohortA, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort: %v", err)
	}
	if err := AssignParticipant(ctx, pool, outsider, showID, cohortA.ID, alice); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider assign, got %v", err)
	}
}

func TestAssignParticipantRejectsNonRosterUser(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_nonroster_director")
	_, showID := showFixture(t, pool, director)
	strangerNotOnRoster := insertTestUser(t, pool, "co_nonroster_stranger")
	ctx := context.Background()

	cohortA, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort: %v", err)
	}
	if err := AssignParticipant(ctx, pool, director, showID, cohortA.ID, strangerNotOnRoster); err == nil || err.Error() != "user_not_a_show_participant" {
		t.Fatalf("expected user_not_a_show_participant, got %v", err)
	}
}
