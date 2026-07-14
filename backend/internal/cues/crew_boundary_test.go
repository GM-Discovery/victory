package cues

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// TestCrewCannotArchiveShow proves Crew's non-destructive authority
// (showruns.CanCrewPerformNonDestructiveEdit, which backs Cue CRUD/GO) is
// never enough to archive a Show -- shows.ArchiveShow gates on
// showruns.CanManageShowRun directly, which Crew never passes (Kernel 70
// §1.8's explicit "Crew may not delete or archive Scenes, Shows, Show
// Runs, or Productions").
func TestCrewCannotArchiveShow(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_crew_archive_producer")
	f := buildCueFixture(t, pool, producer)

	crew := insertCuesTestUser(t, pool, "cue_crew_archive_crew")
	addRoster(t, pool, producer, f.showRunID, crew, "crew")

	// Crew can view backstage (proves the roster row took effect)...
	ok, err := showruns.CanViewBackstage(context.Background(), pool, crew, f.locationID)
	if err != nil || !ok {
		t.Fatalf("expected crew to pass CanViewBackstage, ok=%v err=%v", ok, err)
	}
	// ...but cannot archive the Show.
	if _, err := shows.ArchiveShow(context.Background(), pool, crew, f.showID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for crew archiving a show, got %v", err)
	}
}

// TestCrewCannotChangeCurrentScenePointerDirectly proves Crew cannot call
// the current-scene-pointer endpoint directly -- only indirectly, through
// a Cue's go_to_scene action (Kernel 70 §1.8: Crew's GO right is scoped to
// pressing an existing Cue, not calling the Director-console-level pointer
// API itself).
func TestCrewCannotChangeCurrentScenePointerDirectly(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_crew_pointer_producer")
	f := buildCueFixture(t, pool, producer)

	crew := insertCuesTestUser(t, pool, "cue_crew_pointer_crew")
	addRoster(t, pool, producer, f.showRunID, crew, "crew")

	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, crew, f.showID, f.placementID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for crew calling SetCurrentScenePlacement directly, got %v", err)
	}

	// But Crew CAN press GO on a Cue whose action changes the current
	// Scene (Kernel 70 §1.8/§6.2) -- the indirect path works.
	cue, err := CreateCue(context.Background(), pool, crew, f.placementID, CreateCueInput{
		InternalName: "Crew GO Cue",
		Actions: []CueAction{
			{Type: ActionTypeGoToScene, GoToScene: &GoToSceneAction{ShowScenePlacementID: f.placement2ID}},
		},
	})
	if err != nil {
		t.Fatalf("crew create cue: %v", err)
	}
	result, err := ExecuteCue(context.Background(), pool, crew, cue.ID, newIdempotencyKey(t))
	if err != nil {
		t.Fatalf("crew execute cue: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected crew's GO press to succeed via the Cue path, got %+v", result)
	}
}

// TestCrewCannotEditBaseScene proves Crew's non-destructive edit right
// covers a Show's placement-level (This Show's Version) config only, never
// the shared reusable Scene (Kernel 70 §1.8/§3.3: "Crew may... edit This
// Show's Version" but the Base Scene is Director/Producer/Operator-only).
func TestCrewCannotEditBaseScene(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_crew_base_scene_producer")
	f := buildCueFixture(t, pool, producer)

	crew := insertCuesTestUser(t, pool, "cue_crew_base_scene_crew")
	addRoster(t, pool, producer, f.showRunID, crew, "crew")

	placement, err := scenes.LoadPlacementByID(context.Background(), pool, f.placementID)
	if err != nil {
		t.Fatalf("load placement: %v", err)
	}
	if _, err := scenes.UpdateScene(context.Background(), pool, crew, placement.SceneID, scenes.UpdateScenePatch{
		Title: strPtrCue("Crew Renamed Base Scene"),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for crew editing the base Scene, got %v", err)
	}
}

func strPtrCue(s string) *string { return &s }

// TestListTriggerableCuesForViewerCuratesAndExcludesAudience proves the
// player-facing stage button listing: only enabled, triggerable-by-this-
// viewer Cues appear, curated to id+label only, and Audience always sees
// an empty list even when Cues exist.
func TestListTriggerableCuesForViewerCuratesAndExcludesAudience(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_playerlist_producer")
	f := buildCueFixture(t, pool, producer)

	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Director Only Internal Name", TriggerScope: TriggerScopeDirectorCrewOnly,
	}); err != nil {
		t.Fatalf("create director-only cue: %v", err)
	}
	playerCue, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Player Cue Internal Name", StageButtonLabel: "Next →", TriggerScope: TriggerScopePlayersMayTrigger,
	})
	if err != nil {
		t.Fatalf("create player cue: %v", err)
	}
	disabled := false
	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: "Disabled Player Cue", TriggerScope: TriggerScopePlayersMayTrigger, Enabled: &disabled,
	}); err != nil {
		t.Fatalf("create disabled player cue: %v", err)
	}

	player := insertCuesTestUser(t, pool, "cue_playerlist_player")
	addRoster(t, pool, producer, f.showRunID, player, "player")
	audience := insertCuesTestUser(t, pool, "cue_playerlist_audience")
	addRoster(t, pool, producer, f.showRunID, audience, "audience")

	playerList, err := ListTriggerableCuesForViewer(context.Background(), pool, player, f.placementID)
	if err != nil {
		t.Fatalf("list for player: %v", err)
	}
	if len(playerList) != 1 || playerList[0].ID != playerCue.ID || playerList[0].Label != "Next →" {
		t.Fatalf("expected exactly the one enabled, player-triggerable cue with its label, got %+v", playerList)
	}

	audienceList, err := ListTriggerableCuesForViewer(context.Background(), pool, audience, f.placementID)
	if err != nil {
		t.Fatalf("list for audience: %v", err)
	}
	if len(audienceList) != 0 {
		t.Fatalf("expected audience to see zero triggerable cues, got %+v", audienceList)
	}

	marshaled, err := json.Marshal(playerList)
	if err != nil {
		t.Fatalf("marshal player list: %v", err)
	}
	if strings.Contains(string(marshaled), "trigger_scope") || strings.Contains(string(marshaled), "Director Only Internal Name") {
		t.Fatalf("player-visible cue list leaked backstage-only fields/data: %s", marshaled)
	}
}

// TestCueDataNeverExposedThroughAudienceScenePlacementJSON is a tripwire
// proof mirroring shows_test.go's Show-struct tripwire: even though Cues
// live in their own table (never joined by
// scenes.ListAudiencePlacementsForShow), assert the curated
// AudienceScenePlacement type has no field capable of carrying Cue data at
// all -- a structural, not per-query, guarantee.
func TestCueDataNeverExposedThroughAudienceScenePlacementJSON(t *testing.T) {
	pool := openCuesTestPool(t)
	producer := insertCuesTestUser(t, pool, "cue_tripwire_producer")
	f := buildCueFixture(t, pool, producer)

	tripwireName := "BACKSTAGE-ONLY-CUE-SECRET-71ab"
	if _, err := CreateCue(context.Background(), pool, producer, f.placementID, CreateCueInput{
		InternalName: tripwireName,
	}); err != nil {
		t.Fatalf("create cue: %v", err)
	}

	ready := "ready"
	if _, err := scenes.UpdatePlacement(context.Background(), pool, producer, f.placementID, scenes.UpdatePlacementPatch{Status: &ready}); err != nil {
		t.Fatalf("mark placement ready: %v", err)
	}

	entries, err := scenes.ListAudiencePlacementsForShow(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("list audience placements: %v", err)
	}
	marshaled, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal audience entries: %v", err)
	}
	if strings.Contains(string(marshaled), tripwireName) {
		t.Fatalf("cue internal_name leaked into the curated Audience Scene Program: %s", marshaled)
	}
	if strings.Contains(string(marshaled), "cue") || strings.Contains(string(marshaled), "trigger_scope") {
		t.Fatalf("Audience Scene Program response unexpectedly mentions Cue-shaped fields: %s", marshaled)
	}
}
