package cues

import (
	"context"
	"testing"
)

func TestCreateCueRequiresCrewOrManageAuthority(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_producer")
	f := buildCueFixture(t, pool, producer)

	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Director Cue",
	}); err != nil {
		t.Fatalf("expected producer to create cue: %v", err)
	}

	crew := insertCuesTestUser(t, pool, "cue_crew")
	addRoster(t, pool, producer, f.showRunID, crew, "crew")
	if _, err := CreateCue(context.Background(), pool, crew, f.placementID, CreateCueInput{
		InternalName: "Crew Cue",
	}); err != nil {
		t.Fatalf("expected crew to create a non-destructive cue: %v", err)
	}

	outsider := insertCuesTestUser(t, pool, "cue_outsider")
	if _, err := CreateCue(context.Background(), pool, outsider, f.placementID, CreateCueInput{
		InternalName: "Outsider Cue",
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}
}

func TestCreateCueValidatesActionShape(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_validate_producer")
	f := buildCueFixture(t, pool, producer)

	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Bad Cue", Actions: []CueAction{{Type: "go_to_scene"}},
	}); err == nil || err.Error() != "go_to_scene_target_required" {
		t.Fatalf("expected go_to_scene_target_required, got %v", err)
	}

	// Kernel 90 made reveal_object real, so it is no longer an unknown type.
	// What replaced that assertion is stricter, not weaker: the action is
	// accepted as a TYPE but refused without canonical identity, which is
	// what makes it impossible to author a Cue targeting a label or a
	// coordinate (§22/§54) -- there is no field for one and the id is
	// mandatory.
	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Untargeted Reveal Cue", Actions: []CueAction{{Type: ActionTypeRevealObject}},
	}); err == nil || err.Error() != "stage_object_target_required" {
		t.Fatalf("expected stage_object_target_required, got %v", err)
	}

	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Half Targeted Cue",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: "scene_stage_element",
		}}},
	}); err == nil || err.Error() != "stage_object_target_required" {
		t.Fatalf("expected stage_object_target_required for a kind with no id, got %v", err)
	}

	// An ephemeral effect can never be authored as a Cue target, and it is
	// refused at authoring time rather than only at fire time (§39).
	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Ephemeral Target Cue",
		Actions: []CueAction{{Type: ActionTypeHideObject, StageObject: &StageObjectAction{
			ObjectKind: "stage_effect", ObjectID: "00000000-0000-0000-0000-000000000001",
		}}},
	}); err == nil || err.Error() != "unsupported_object_kind" {
		t.Fatalf("expected unsupported_object_kind for an ephemeral effect target, got %v", err)
	}

	// A genuinely unknown action type is still refused.
	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Unknown Type Cue", Actions: []CueAction{{Type: "summon_dragon"}},
	}); err == nil || err.Error() != "unknown_cue_action_type" {
		t.Fatalf("expected unknown_cue_action_type, got %v", err)
	}
}

func TestCanTriggerCueDirectorCrewOnlyDefault(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_trigger_dco_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Director Crew Only Cue", TriggerScope: TriggerScopeDirectorCrewOnly,
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	crew := insertCuesTestUser(t, pool, "cue_trigger_dco_crew")
	addRoster(t, pool, producer, f.showRunID, crew, "crew")
	player := insertCuesTestUser(t, pool, "cue_trigger_dco_player")
	addRoster(t, pool, producer, f.showRunID, player, "player")
	audience := insertCuesTestUser(t, pool, "cue_trigger_dco_audience")
	addRoster(t, pool, producer, f.showRunID, audience, "audience")

	if ok, err := CanTriggerCue(context.Background(), pool, producer, c.ID); err != nil || !ok {
		t.Fatalf("expected producer to trigger, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, crew, c.ID); err != nil || !ok {
		t.Fatalf("expected crew to trigger, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, player, c.ID); err != nil || ok {
		t.Fatalf("expected player denied under director_crew_only, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, audience, c.ID); err != nil || ok {
		t.Fatalf("expected audience denied, ok=%v err=%v", ok, err)
	}
}

func TestCanTriggerCuePlayersMayTrigger(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_trigger_pmt_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Players May Trigger Cue", TriggerScope: TriggerScopePlayersMayTrigger,
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	player := insertCuesTestUser(t, pool, "cue_trigger_pmt_player")
	addRoster(t, pool, producer, f.showRunID, player, "player")
	audience := insertCuesTestUser(t, pool, "cue_trigger_pmt_audience")
	addRoster(t, pool, producer, f.showRunID, audience, "audience")
	guest := insertCuesTestUser(t, pool, "cue_trigger_pmt_guest")
	addRoster(t, pool, producer, f.showRunID, guest, "guest")

	if ok, err := CanTriggerCue(context.Background(), pool, player, c.ID); err != nil || !ok {
		t.Fatalf("expected player to trigger, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, audience, c.ID); err != nil || ok {
		t.Fatalf("expected audience denied, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, guest, c.ID); err != nil || ok {
		t.Fatalf("expected a non-player, non-crew, non-audience role (guest) denied under players_may_trigger, ok=%v err=%v", ok, err)
	}
}

// TestCanTriggerCueAnyRosterMemberExcludesAudience is the explicit
// negative-case proof (Kernel 70 §1.7, §6.3): even under the broadest
// trigger_scope, an Audience roster row must never be permitted to
// trigger a Cue.
func TestCanTriggerCueAnyRosterMemberExcludesAudience(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_trigger_arm_producer")
	f := buildCueFixture(t, pool, producer)

	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Any Roster Member Cue", TriggerScope: TriggerScopeAnyRosterMember,
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}

	guest := insertCuesTestUser(t, pool, "cue_trigger_arm_guest")
	addRoster(t, pool, producer, f.showRunID, guest, "guest")
	observer := insertCuesTestUser(t, pool, "cue_trigger_arm_observer")
	addRoster(t, pool, producer, f.showRunID, observer, "observer")
	audience := insertCuesTestUser(t, pool, "cue_trigger_arm_audience")
	addRoster(t, pool, producer, f.showRunID, audience, "audience")

	if ok, err := CanTriggerCue(context.Background(), pool, guest, c.ID); err != nil || !ok {
		t.Fatalf("expected guest (non-audience roster role) to trigger under any_roster_member, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, observer, c.ID); err != nil || !ok {
		t.Fatalf("expected observer (non-audience roster role) to trigger under any_roster_member, ok=%v err=%v", ok, err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, audience, c.ID); err != nil || ok {
		t.Fatalf("expected audience denied even under any_roster_member, ok=%v err=%v", ok, err)
	}
}

func TestCanTriggerCueDeniedForDisabledCue(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_disabled_producer")
	f := buildCueFixture(t, pool, producer)

	disabled := false
	c, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Disabled Cue", Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	if ok, err := CanTriggerCue(context.Background(), pool, producer, c.ID); err != nil || ok {
		t.Fatalf("expected a disabled cue to never be triggerable, ok=%v err=%v", ok, err)
	}
}
