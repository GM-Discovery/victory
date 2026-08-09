package socio

import (
	"context"
	"testing"

	"victory/backend/internal/cohorts"
)

func TestAllEightPoolsReadWithZeroedDefaults(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_director")
	showRunID, _, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_alice")

	state, err := GetState(context.Background(), pool, characterCardID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(state.Pools) != 8 {
		t.Fatalf("expected 8 pools, got %d", len(state.Pools))
	}
	wantKeys := map[PoolKey]bool{
		PoolHealth: true, PoolPsyche: true, PoolMotion: true, PoolWill: true,
		PoolEssence: true, PoolFocus: true, PoolPerception: true, PoolHeart: true,
	}
	for _, p := range state.Pools {
		if !wantKeys[p.Key] {
			t.Fatalf("unexpected pool key %q", p.Key)
		}
		delete(wantKeys, p.Key)
		if p.Current != 0 || p.Max != 0 {
			t.Fatalf("expected zeroed default for %q, got current=%d max=%d", p.Key, p.Current, p.Max)
		}
	}
	if len(wantKeys) != 0 {
		t.Fatalf("missing pool keys: %v", wantKeys)
	}
}

func TestSetPoolClampsCurrentIntoMaxRange(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_clamp_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_clamp_alice")
	ctx := context.Background()

	state, err := SetPool(ctx, pool, director, showID, characterCardID, PoolHealth, 5, 10)
	if err != nil {
		t.Fatalf("set pool: %v", err)
	}
	health := poolByKey(t, state, PoolHealth)
	if health.Current != 5 || health.Max != 10 {
		t.Fatalf("expected current=5 max=10, got current=%d max=%d", health.Current, health.Max)
	}

	// Requesting current above max clamps down to max, never errors or
	// invents room (kernel-85 S2.2).
	state, err = SetPool(ctx, pool, director, showID, characterCardID, PoolHealth, 999, 10)
	if err != nil {
		t.Fatalf("set pool over max: %v", err)
	}
	health = poolByKey(t, state, PoolHealth)
	if health.Current != 10 {
		t.Fatalf("expected current clamped to max (10), got %d", health.Current)
	}

	// Negative current clamps to 0.
	state, err = SetPool(ctx, pool, director, showID, characterCardID, PoolHealth, -5, 10)
	if err != nil {
		t.Fatalf("set pool negative: %v", err)
	}
	health = poolByKey(t, state, PoolHealth)
	if health.Current != 0 {
		t.Fatalf("expected negative current clamped to 0, got %d", health.Current)
	}
}

func poolByKey(t *testing.T, state State, key PoolKey) Pool {
	t.Helper()
	for _, p := range state.Pools {
		if p.Key == key {
			return p
		}
	}
	t.Fatalf("pool %q not found in state", key)
	return Pool{}
}

func TestSetPoolRejectsInvalidPoolKey(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_invalid_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_invalid_alice")

	if _, err := SetPool(context.Background(), pool, director, showID, characterCardID, PoolKey("stamina"), 5, 10); err == nil || err.Error() != "invalid_pool_key" {
		t.Fatalf("expected invalid_pool_key, got %v", err)
	}
}

func TestSetPoolRequiresDirectorAuthorityAndRosterMembership(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_auth_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_auth_alice")
	outsider := insertTestUser(t, pool, "so_auth_outsider")
	ctx := context.Background()

	if _, err := SetPool(ctx, pool, outsider, showID, characterCardID, PoolHealth, 5, 10); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}

	// A Character that exists but is not on THIS show's roster must be
	// rejected even for the Director of this show.
	_, otherShowID, otherLocationID := showFixture(t, pool, insertTestUser(t, pool, "so_auth_other_director"))
	_ = otherLocationID
	if _, err := SetPool(ctx, pool, director, otherShowID, characterCardID, PoolHealth, 5, 10); err == nil || err.Error() != "character_not_on_show" {
		t.Fatalf("expected character_not_on_show, got %v", err)
	}
}

func TestApplyAndClearStatus(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_status_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_status_alice")
	ctx := context.Background()

	intensity := 2
	applied, err := ApplyStatus(ctx, pool, director, showID, characterCardID, "winded", &intensity)
	if err != nil {
		t.Fatalf("apply status: %v", err)
	}
	if applied.StatusKey != "winded" || applied.Intensity == nil || *applied.Intensity != 2 {
		t.Fatalf("unexpected applied status: %+v", applied)
	}

	active, err := ListActiveStatuses(ctx, pool, characterCardID)
	if err != nil {
		t.Fatalf("list active statuses: %v", err)
	}
	if len(active) != 1 || active[0].StatusKey != "winded" {
		t.Fatalf("expected 1 active status 'winded', got %+v", active)
	}

	// Re-applying updates intensity in place rather than duplicating.
	intensity2 := 3
	if _, err := ApplyStatus(ctx, pool, director, showID, characterCardID, "winded", &intensity2); err != nil {
		t.Fatalf("re-apply status: %v", err)
	}
	active, err = ListActiveStatuses(ctx, pool, characterCardID)
	if err != nil {
		t.Fatalf("list active statuses after re-apply: %v", err)
	}
	if len(active) != 1 || active[0].Intensity == nil || *active[0].Intensity != 3 {
		t.Fatalf("expected single status updated to intensity 3, got %+v", active)
	}

	if err := ClearStatus(ctx, pool, director, showID, characterCardID, "winded"); err != nil {
		t.Fatalf("clear status: %v", err)
	}
	active, err = ListActiveStatuses(ctx, pool, characterCardID)
	if err != nil {
		t.Fatalf("list active statuses after clear: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected no active statuses after clear, got %+v", active)
	}

	// Clearing again is an idempotent no-op, not an error.
	if err := ClearStatus(ctx, pool, director, showID, characterCardID, "winded"); err != nil {
		t.Fatalf("expected idempotent clear, got %v", err)
	}
}

func TestApplyStatusRejectsUnknownStatusKey(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_unknown_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, characterCardID := playerFixture(t, pool, director, showRunID, locationID, "so_unknown_alice")

	if _, err := ApplyStatus(context.Background(), pool, director, showID, characterCardID, "not_a_real_status", nil); err == nil || err.Error() != "invalid_status_key" {
		t.Fatalf("expected invalid_status_key, got %v", err)
	}
}

func TestGameStatusFiltersToSelectedCohort(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "so_gs_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, aliceCard := playerFixture(t, pool, director, showRunID, locationID, "so_gs_alice")
	_, bobCard := playerFixture(t, pool, director, showRunID, locationID, "so_gs_bob")
	ctx := context.Background()

	cohort1, err := cohorts.CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 1: %v", err)
	}
	if err := cohorts.AssignParticipant(ctx, pool, director, showID, cohort1.ID, aliceID); err != nil {
		t.Fatalf("assign alice: %v", err)
	}
	// bob stays Ungrouped.

	if _, err := SetPool(ctx, pool, director, showID, aliceCard, PoolHealth, 7, 10); err != nil {
		t.Fatalf("set alice pool: %v", err)
	}
	if _, err := SetPool(ctx, pool, director, showID, bobCard, PoolHealth, 3, 10); err != nil {
		t.Fatalf("set bob pool: %v", err)
	}

	blocks, err := BuildGameStatusForCohort(ctx, pool, director, showID, cohort1.ID)
	if err != nil {
		t.Fatalf("build game status for cohort 1: %v", err)
	}
	if len(blocks) != 1 || blocks[0].CharacterCardID != aliceCard {
		t.Fatalf("expected only alice's block in cohort 1, got %+v", blocks)
	}

	ungroupedBlocks, err := BuildGameStatusForCohort(ctx, pool, director, showID, "ungrouped")
	if err != nil {
		t.Fatalf("build game status for ungrouped: %v", err)
	}
	found := false
	for _, b := range ungroupedBlocks {
		if b.CharacterCardID == bobCard {
			found = true
		}
		if b.CharacterCardID == aliceCard {
			t.Fatal("alice must not appear in Ungrouped game status once assigned to a cohort")
		}
	}
	if !found {
		t.Fatalf("expected bob in Ungrouped game status, got %+v", ungroupedBlocks)
	}
}
