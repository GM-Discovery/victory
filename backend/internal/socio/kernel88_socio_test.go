package socio

import (
	"context"
	"testing"

	"victory/backend/internal/venuecoordination"
)

// --- Fate --------------------------------------------------------------

func TestFateNormalCapIsSeven(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_fate_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_fate_alice")
	ctx := context.Background()

	state, err := AwardFate(ctx, pool, director, showID, cardID, 10, "director_award", "")
	if err != nil {
		t.Fatalf("award fate: %v", err)
	}
	if state.Balance != 7 {
		t.Fatalf("expected balance clamped to normal cap 7, got %d", state.Balance)
	}
}

func TestFateCreationModeAllowsTemporaryOverCapUpTo12(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_fate_creation_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_fate_creation_alice")
	ctx := context.Background()

	if _, err := SetCreationMode(ctx, pool, director, showID, cardID, true); err != nil {
		t.Fatalf("enable creation mode: %v", err)
	}
	state, err := AwardFate(ctx, pool, director, showID, cardID, 12, "creation_guidance", "")
	if err != nil {
		t.Fatalf("award fate during creation: %v", err)
	}
	if state.Balance != 12 {
		t.Fatalf("expected balance 12 during creation mode, got %d", state.Balance)
	}
	// Still clamps at the creation ceiling of 12, not unlimited.
	state, err = AwardFate(ctx, pool, director, showID, cardID, 5, "creation_guidance", "")
	if err != nil {
		t.Fatalf("award fate over creation cap: %v", err)
	}
	if state.Balance != 12 {
		t.Fatalf("expected balance clamped at creation cap 12, got %d", state.Balance)
	}
}

func TestFateCreationModeOffClampsExcessAndLogsLedger(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_fate_clamp_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_fate_clamp_alice")
	ctx := context.Background()

	if _, err := SetCreationMode(ctx, pool, director, showID, cardID, true); err != nil {
		t.Fatalf("enable creation mode: %v", err)
	}
	if _, err := AwardFate(ctx, pool, director, showID, cardID, 12, "creation_guidance", ""); err != nil {
		t.Fatalf("award fate: %v", err)
	}

	state, err := SetCreationMode(ctx, pool, director, showID, cardID, false)
	if err != nil {
		t.Fatalf("disable creation mode: %v", err)
	}
	if state.Balance != 7 {
		t.Fatalf("expected excess above 7 lost on creation-mode-off, got balance %d", state.Balance)
	}

	ledger, err := ListFateLedger(ctx, pool, cardID, 10)
	if err != nil {
		t.Fatalf("list fate ledger: %v", err)
	}
	found := false
	for _, e := range ledger {
		if e.Reason == "creation_clamp" && e.Delta == -5 && e.BalanceAfter == 7 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a creation_clamp ledger entry for -5, got %+v", ledger)
	}
}

func TestSpendFateOwnerCanSpendOwnButNeverIncreasesBalance(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_spend_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_spend_alice")
	ctx := context.Background()

	if _, err := AwardFate(ctx, pool, director, showID, cardID, 5, "director_award", ""); err != nil {
		t.Fatalf("award fate: %v", err)
	}
	state, err := SpendFate(ctx, pool, aliceID, showID, cardID, 2, "used a defined use")
	if err != nil {
		t.Fatalf("owner spend fate: %v", err)
	}
	if state.Balance != 3 {
		t.Fatalf("expected balance 3 after spending 2 of 5, got %d", state.Balance)
	}

	// Cannot spend more than the current balance.
	if _, err := SpendFate(ctx, pool, aliceID, showID, cardID, 999, ""); err == nil || err.Error() != "insufficient_fate" {
		t.Fatalf("expected insufficient_fate, got %v", err)
	}

	// A negative/zero amount is rejected outright -- SpendFate can never be
	// used to increase a balance.
	if _, err := SpendFate(ctx, pool, aliceID, showID, cardID, -5, ""); err == nil || err.Error() != "amount_must_be_positive" {
		t.Fatalf("expected amount_must_be_positive, got %v", err)
	}
}

func TestSpendFateRejectsNonOwnerNonDirector(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_spend_auth_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_spend_auth_alice")
	outsider := insertTestUser(t, pool, "k88_spend_auth_outsider")
	ctx := context.Background()

	if _, err := AwardFate(ctx, pool, director, showID, cardID, 5, "director_award", ""); err != nil {
		t.Fatalf("award fate: %v", err)
	}
	if _, err := SpendFate(ctx, pool, outsider, showID, cardID, 1, ""); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider spend, got %v", err)
	}
}

func TestAwardFateRequiresDirectorAuthority(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_award_auth_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_award_auth_alice")
	ctx := context.Background()

	if _, err := AwardFate(ctx, pool, aliceID, showID, cardID, 5, "director_award", ""); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for Player-attempted award, got %v", err)
	}
}

// --- Stance --------------------------------------------------------------

func TestStanceRegistryHasFiveCanonicalStances(t *testing.T) {
	pool := openTestPool(t)
	defs, err := ListStanceDefinitions(context.Background(), pool)
	if err != nil {
		t.Fatalf("list stance definitions: %v", err)
	}
	want := map[string]bool{"insight": true, "command": true, "convince": true, "sympathize": true, "follow": true}
	if len(defs) < 5 {
		t.Fatalf("expected at least 5 stances, got %d", len(defs))
	}
	for _, d := range defs {
		delete(want, d.Key)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected stance keys: %v", want)
	}
}

func TestSetStanceOwnerOrDirectorOnly(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_stance_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_stance_alice")
	outsider := insertTestUser(t, pool, "k88_stance_outsider")
	ctx := context.Background()

	stance, err := SetStance(ctx, pool, aliceID, showID, cardID, "command")
	if err != nil {
		t.Fatalf("owner sets own stance: %v", err)
	}
	if stance.Key != "command" {
		t.Fatalf("expected stance 'command', got %q", stance.Key)
	}

	if _, err := SetStance(ctx, pool, director, showID, cardID, "follow"); err != nil {
		t.Fatalf("director overrides stance: %v", err)
	}

	if _, err := SetStance(ctx, pool, outsider, showID, cardID, "insight"); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}

	if _, err := SetStance(ctx, pool, aliceID, showID, cardID, "not_a_real_stance"); err == nil || err.Error() != "invalid_stance_key" {
		t.Fatalf("expected invalid_stance_key, got %v", err)
	}
}

// --- Blank state flags -----------------------------------------------------

func TestAddFlagIsDistinctFromCanonicalStatuses(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_flag_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_flag_alice")
	ctx := context.Background()

	flag, err := AddFlag(ctx, pool, director, showID, cardID, "Waiting on Kessa")
	if err != nil {
		t.Fatalf("add flag: %v", err)
	}
	if !flag.IsBlank || flag.Label != "Waiting on Kessa" {
		t.Fatalf("expected blank flag with label, got %+v", flag)
	}

	if _, err := ApplyStatus(ctx, pool, director, showID, cardID, "winded", nil); err != nil {
		t.Fatalf("apply canonical status: %v", err)
	}

	active, err := ListActiveStatuses(ctx, pool, cardID)
	if err != nil {
		t.Fatalf("list active statuses: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active entries (1 blank + 1 canonical), got %+v", active)
	}
	blankCount, canonicalCount := 0, 0
	for _, a := range active {
		if a.IsBlank {
			blankCount++
		} else {
			canonicalCount++
		}
	}
	if blankCount != 1 || canonicalCount != 1 {
		t.Fatalf("expected 1 blank + 1 canonical, got blank=%d canonical=%d", blankCount, canonicalCount)
	}

	if err := ClearFlag(ctx, pool, director, showID, cardID, flag.ID); err != nil {
		t.Fatalf("clear flag: %v", err)
	}
	active, err = ListActiveStatuses(ctx, pool, cardID)
	if err != nil {
		t.Fatalf("list after clear: %v", err)
	}
	if len(active) != 1 || active[0].IsBlank {
		t.Fatalf("expected only the canonical status left, got %+v", active)
	}
}

func TestAddFlagRejectsTooLongLabel(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_flag_long_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_flag_long_alice")

	long := ""
	for i := 0; i < 61; i++ {
		long += "x"
	}
	if _, err := AddFlag(context.Background(), pool, director, showID, cardID, long); err == nil || err.Error() != "label_too_long" {
		t.Fatalf("expected label_too_long, got %v", err)
	}
}

// --- Projection (Owner/Director tiered read) --------------------------------

func TestProjectSocioStateOwnerGetsQualitativePoolsButExactFate(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_proj_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_proj_alice")
	ctx := context.Background()

	if _, err := SetPool(ctx, pool, director, showID, cardID, PoolHealth, 3, 10); err != nil {
		t.Fatalf("set pool: %v", err)
	}
	if _, err := AwardFate(ctx, pool, director, showID, cardID, 4, "director_award", ""); err != nil {
		t.Fatalf("award fate: %v", err)
	}

	proj, err := ProjectSocioState(ctx, pool, aliceID, showID, cardID)
	if err != nil {
		t.Fatalf("project as owner: %v", err)
	}
	if proj.Tier != TierOwner {
		t.Fatalf("expected owner tier, got %q", proj.Tier)
	}
	if proj.ExactPools != nil {
		t.Fatalf("owner tier must not receive exact pools, got %+v", proj.ExactPools)
	}
	if proj.FateBalance != 4 {
		t.Fatalf("owner tier should see own exact fate balance, expected 4 got %d", proj.FateBalance)
	}
	var healthCondition string
	for _, p := range proj.QualitativePools {
		if p.Key == PoolHealth {
			healthCondition = p.Condition
		}
	}
	if healthCondition != "critical" {
		t.Fatalf("expected qualitative condition 'critical' for 3/10 health, got %q", healthCondition)
	}
}

func TestProjectSocioStateDirectorGetsExactPools(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_proj_dir_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_proj_dir_alice")
	ctx := context.Background()

	if _, err := SetPool(ctx, pool, director, showID, cardID, PoolHealth, 3, 10); err != nil {
		t.Fatalf("set pool: %v", err)
	}

	proj, err := ProjectSocioState(ctx, pool, director, showID, cardID)
	if err != nil {
		t.Fatalf("project as director: %v", err)
	}
	if proj.Tier != TierDirector {
		t.Fatalf("expected director tier, got %q", proj.Tier)
	}
	if proj.ExactPools == nil {
		t.Fatal("director tier must receive exact pools")
	}
}

func TestProjectSocioStateRejectsNonOwnerNonDirector(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_proj_reject_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88_proj_reject_alice")
	_, otherCardID := playerFixture(t, pool, director, showRunID, locationID, "k88_proj_reject_bob")
	ctx := context.Background()

	// Bob's own Player account, viewing Alice's Character, must be
	// rejected -- there is no "view a teammate's mechanical state" surface.
	bobID := ""
	if err := pool.QueryRow(ctx, `SELECT owner_user_id::text FROM character_cards WHERE id = $1`, otherCardID).Scan(&bobID); err != nil {
		t.Fatalf("resolve bob's user id: %v", err)
	}
	if _, err := ProjectSocioState(ctx, pool, bobID, showID, cardID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized, got %v", err)
	}
}

// --- Current Turn coordination ---------------------------------------------

func TestAssignCurrentTurnRequiresCohortMembership(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_turn_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, _ := playerFixture(t, pool, director, showRunID, locationID, "k88_turn_alice")
	outsider := insertTestUser(t, pool, "k88_turn_outsider")
	reg := venuecoordination.NewRegistry()
	ctx := context.Background()

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", aliceID); err != nil {
		t.Fatalf("assign current turn to ungrouped member: %v", err)
	}
	view, err := BuildCoordinationView(ctx, pool, reg, showID, "ungrouped")
	if err != nil {
		t.Fatalf("build coordination view: %v", err)
	}
	if view.CurrentTurnUserID != aliceID {
		t.Fatalf("expected current turn user %s, got %s", aliceID, view.CurrentTurnUserID)
	}

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", outsider); err == nil {
		t.Fatal("expected error assigning current turn to a non-roster outsider")
	}
}

func TestAssignCurrentTurnAllowsCurrentHolderToPassItOn(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_turn_pass_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, _ := playerFixture(t, pool, director, showRunID, locationID, "k88_turn_pass_alice")
	bobID, _ := playerFixture(t, pool, director, showRunID, locationID, "k88_turn_pass_bob")
	reg := venuecoordination.NewRegistry()
	ctx := context.Background()

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", aliceID); err != nil {
		t.Fatalf("director assigns turn to alice: %v", err)
	}
	// Alice (the current holder), not a Director, passes the turn to Bob.
	if _, err := AssignCurrentTurn(ctx, pool, reg, aliceID, showID, "ungrouped", bobID); err != nil {
		t.Fatalf("alice passes turn to bob: %v", err)
	}
	view, err := BuildCoordinationView(ctx, pool, reg, showID, "ungrouped")
	if err != nil {
		t.Fatalf("build coordination view: %v", err)
	}
	if view.CurrentTurnUserID != bobID {
		t.Fatalf("expected current turn user %s, got %s", bobID, view.CurrentTurnUserID)
	}
}

// --- Interrupts / Helping ----------------------------------------------------

func TestOpenPrimaryActionRequiresOwnCurrentTurn(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_pa_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, aliceCard := playerFixture(t, pool, director, showRunID, locationID, "k88_pa_alice")
	bobID, _ := playerFixture(t, pool, director, showRunID, locationID, "k88_pa_bob")
	reg := venuecoordination.NewRegistry()
	ctx := context.Background()

	// No one has Current Turn yet -- Alice cannot open a primary action for
	// herself.
	if _, err := OpenPrimaryAction(ctx, pool, reg, aliceID, showID, "", "ungrouped", aliceCard, "Bake Pie"); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized before current turn is set, got %v", err)
	}

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", aliceID); err != nil {
		t.Fatalf("assign current turn to alice: %v", err)
	}

	action, err := OpenPrimaryAction(ctx, pool, reg, aliceID, showID, "", "ungrouped", aliceCard, "Bake Pie")
	if err != nil {
		t.Fatalf("alice opens her own primary action on her turn: %v", err)
	}
	if action.Kind != "primary" || action.Status != "open" {
		t.Fatalf("unexpected primary action: %+v", action)
	}

	// Bob does not hold Current Turn -- he cannot open a primary action for
	// Alice's Character (or even attempt one using her card ID).
	if _, err := OpenPrimaryAction(ctx, pool, reg, bobID, showID, "", "ungrouped", aliceCard, "Peel Apples"); err == nil {
		t.Fatal("expected error for non-current-turn-holder opening a primary action")
	}

	// Director may open on a Player's behalf regardless of Current Turn.
	if _, err := OpenPrimaryAction(ctx, pool, reg, director, showID, "", "ungrouped", aliceCard, "Director-declared action"); err != nil {
		t.Fatalf("director opens primary action on player's behalf: %v", err)
	}
}

func TestInterruptResolutionOverageAndFailure(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_interrupt_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, aliceCard := playerFixture(t, pool, director, showRunID, locationID, "k88_interrupt_alice")
	_, bobCard := playerFixture(t, pool, director, showRunID, locationID, "k88_interrupt_bob")
	reg := venuecoordination.NewRegistry()
	ctx := context.Background()

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", aliceID); err != nil {
		t.Fatalf("assign current turn: %v", err)
	}
	primary, err := OpenPrimaryAction(ctx, pool, reg, aliceID, showID, "", "ungrouped", aliceCard, "Bake Pie")
	if err != nil {
		t.Fatalf("open primary action: %v", err)
	}

	// A successful Help: target complexity 8, roll total 11 -> overage 3.
	interrupt, err := OpenInterrupt(ctx, pool, director, showID, "", primary.ID, bobCard, "Peel Apples", 8)
	if err != nil {
		t.Fatalf("open interrupt: %v", err)
	}
	resolved, err := ResolveInterrupt(ctx, pool, director, showID, interrupt.ID, "", 11)
	if err != nil {
		t.Fatalf("resolve interrupt: %v", err)
	}
	if resolved.Status != "resolved" || resolved.OverageBonus != 3 {
		t.Fatalf("expected resolved with overage 3, got %+v", resolved)
	}

	overage, err := ConsumeOverageFor(ctx, pool, primary.ID)
	if err != nil {
		t.Fatalf("consume overage: %v", err)
	}
	if overage != 3 {
		t.Fatalf("expected consumed overage 3, got %d", overage)
	}
	// Consuming again yields nothing -- a bonus is never applied twice.
	overage, err = ConsumeOverageFor(ctx, pool, primary.ID)
	if err != nil {
		t.Fatalf("consume overage again: %v", err)
	}
	if overage != 0 {
		t.Fatalf("expected 0 on second consume, got %d", overage)
	}

	// A failed Help (roll below target complexity) adds no overage.
	interrupt2, err := OpenInterrupt(ctx, pool, director, showID, "", primary.ID, bobCard, "Peel Apples Again", 8)
	if err != nil {
		t.Fatalf("open second interrupt: %v", err)
	}
	resolved2, err := ResolveInterrupt(ctx, pool, director, showID, interrupt2.ID, "", 5)
	if err != nil {
		t.Fatalf("resolve failed interrupt: %v", err)
	}
	if resolved2.OverageBonus != 0 {
		t.Fatalf("expected 0 overage on a failed help, got %d", resolved2.OverageBonus)
	}
}

func TestInterruptOfInterruptResolvesInnermostFirst(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88_nested_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, aliceCard := playerFixture(t, pool, director, showRunID, locationID, "k88_nested_alice")
	_, bobCard := playerFixture(t, pool, director, showRunID, locationID, "k88_nested_bob")
	_, carolCard := playerFixture(t, pool, director, showRunID, locationID, "k88_nested_carol")
	reg := venuecoordination.NewRegistry()
	ctx := context.Background()

	if _, err := AssignCurrentTurn(ctx, pool, reg, director, showID, "ungrouped", aliceID); err != nil {
		t.Fatalf("assign current turn: %v", err)
	}
	primary, err := OpenPrimaryAction(ctx, pool, reg, aliceID, showID, "", "ungrouped", aliceCard, "Bake Pie")
	if err != nil {
		t.Fatalf("open primary action: %v", err)
	}
	interrupt1, err := OpenInterrupt(ctx, pool, director, showID, "", primary.ID, bobCard, "Peel Apples", 8)
	if err != nil {
		t.Fatalf("open first interrupt: %v", err)
	}

	// Cannot resolve the outer primary action's roll implicitly, and cannot
	// cancel the primary while interrupt1 is still open.
	if err := CancelPendingAction(ctx, pool, director, showID, primary.ID); err != nil {
		t.Fatalf("cancel should cascade, not error: %v", err)
	}
	// Cancelling the primary cascades to its open child.
	stack, err := ListOpenStack(ctx, pool, showID)
	if err != nil {
		t.Fatalf("list open stack: %v", err)
	}
	for _, a := range stack {
		if a.ID == primary.ID || a.ID == interrupt1.ID {
			t.Fatalf("expected primary and its interrupt both closed after cascade cancel, found open: %+v", a)
		}
	}

	// Re-run the nested scenario without cancelling, proving
	// interrupt-of-interrupt attaches and resolves innermost-first.
	primary2, err := OpenPrimaryAction(ctx, pool, reg, aliceID, showID, "", "ungrouped", aliceCard, "Bake Pie Again")
	if err != nil {
		t.Fatalf("open second primary action: %v", err)
	}
	outer, err := OpenInterrupt(ctx, pool, director, showID, "", primary2.ID, bobCard, "Peel Apples", 8)
	if err != nil {
		t.Fatalf("open outer interrupt: %v", err)
	}
	inner, err := OpenInterrupt(ctx, pool, director, showID, "", outer.ID, carolCard, "Hold the Bowl Steady", 5)
	if err != nil {
		t.Fatalf("open inner interrupt (interrupt of an interrupt): %v", err)
	}

	// The outer interrupt cannot resolve while its own child is still open.
	if _, err := ResolveInterrupt(ctx, pool, director, showID, outer.ID, "", 12); err == nil || err.Error() != "interrupt_has_open_children" {
		t.Fatalf("expected interrupt_has_open_children, got %v", err)
	}

	if _, err := ResolveInterrupt(ctx, pool, director, showID, inner.ID, "", 9); err != nil {
		t.Fatalf("resolve inner interrupt: %v", err)
	}
	if _, err := ResolveInterrupt(ctx, pool, director, showID, outer.ID, "", 12); err != nil {
		t.Fatalf("resolve outer interrupt after inner resolved: %v", err)
	}
}

// --- Kernel 88A: rollable mechanics ------------------------------------

// The Player HUD used to source its roll buttons from the equipped-persona
// venue-sheet, a different identity mechanism than the roster selection the
// roll path authorizes against. A Player with a valid roster Character but no
// equipped persona got no buttons and no explanation. These tests pin the
// replacement to the roster Character and to ProjectSocioState's authority.
func TestListCharacterMechanicsResolvesRosterCharacterWithoutEquippedPersona(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88a_mech_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	aliceID, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88a_mech_alice")
	ctx := context.Background()

	// Deliberately no persona/equip: current_session_personas stays empty,
	// which is exactly the state that produced the original bug.
	var personaCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM current_session_personas WHERE character_card_id = $1
	`, cardID).Scan(&personaCount); err != nil {
		t.Fatalf("count personas: %v", err)
	}
	if personaCount != 0 {
		t.Fatalf("fixture unexpectedly equipped a persona (%d rows)", personaCount)
	}

	mechanics, tier, err := ListCharacterMechanics(ctx, pool, aliceID, showID, cardID)
	if err != nil {
		t.Fatalf("owner listing own mechanics must succeed with no equipped persona: %v", err)
	}
	if tier != TierOwner {
		t.Fatalf("tier = %q, want %q", tier, TierOwner)
	}
	// A bare fixture Character has no compiler skills yet; the contract under
	// test is that the call resolves and is authorized, not that it is
	// non-empty -- the HUD now renders an explanation for the empty case
	// rather than silently omitting the section.
	if mechanics == nil {
		t.Fatal("mechanics must be a list, never nil, so the HUD can distinguish empty from failed")
	}
}

func TestListCharacterMechanicsRejectsUnrelatedViewer(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88a_mech_auth_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88a_mech_auth_alice")
	outsider := insertTestUser(t, pool, "k88a_mech_auth_outsider")
	ctx := context.Background()

	if _, _, err := ListCharacterMechanics(ctx, pool, outsider, showID, cardID); err == nil {
		t.Fatal("an unrelated user must not be able to enumerate another Character's mechanics")
	}
}

func TestListCharacterMechanicsAllowsDirectorButMarksTier(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "k88a_mech_tier_director")
	showRunID, showID, locationID := showFixture(t, pool, director)
	_, cardID := playerFixture(t, pool, director, showRunID, locationID, "k88a_mech_tier_alice")
	ctx := context.Background()

	_, tier, err := ListCharacterMechanics(ctx, pool, director, showID, cardID)
	if err != nil {
		t.Fatalf("director listing a Character's mechanics: %v", err)
	}
	if tier != TierDirector {
		t.Fatalf("tier = %q, want %q -- rolling your own mechanic is an Owner-tier act, so the caller must be able to tell these apart", tier, TierDirector)
	}
}
