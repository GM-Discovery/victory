package scenes

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSceneStageElementBaseAndShowLayers(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sce_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Composer Test Scene", Slug: "composer-test-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	// Base-layer element.
	baseEl, err := CreateSceneStageElement(context.Background(), pool, producer, scene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Base Token", Position: map[string]any{"x": 0.1, "y": 0.2},
	})
	if err != nil {
		t.Fatalf("create base stage element: %v", err)
	}
	if baseEl.Layer != "base" {
		t.Fatalf("expected base layer, got %q", baseEl.Layer)
	}

	// Show-layer element scoped to this placement only.
	showEl, err := CreatePlacementStageElement(context.Background(), pool, producer, placement.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Show Token", Position: map[string]any{"x": 0.5, "y": 0.5},
	})
	if err != nil {
		t.Fatalf("create show-layer stage element: %v", err)
	}
	if showEl.Layer != "show" {
		t.Fatalf("expected show layer, got %q", showEl.Layer)
	}

	// A second placement of the same Scene must NOT see the first
	// placement's Show-layer element -- only its own Show-layer additions
	// plus the shared Base layer.
	_, showID2 := showFixture(t, pool, producer, productionID)
	placement2, err := CreatePlacement(context.Background(), pool, producer, showID2, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create second placement: %v", err)
	}

	comp1, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition 1: %v", err)
	}
	if len(comp1.Elements) != 2 {
		t.Fatalf("expected 2 elements (1 base + 1 show) for placement 1, got %d", len(comp1.Elements))
	}

	comp2, err := LoadResolvedComposition(context.Background(), pool, placement2.ID)
	if err != nil {
		t.Fatalf("load resolved composition 2: %v", err)
	}
	if len(comp2.Elements) != 1 {
		t.Fatalf("expected only the base element (1) for placement 2 with no show-layer additions of its own, got %d", len(comp2.Elements))
	}
	for _, e := range comp2.Elements {
		if e.Label == "Show Token" {
			t.Fatal("placement 2 must never see placement 1's show-layer element")
		}
	}

	// Update, then delete.
	newLabel := "Base Token Renamed"
	updated, err := UpdateStageElement(context.Background(), pool, producer, baseEl.ID, UpdateStageElementPatch{Label: &newLabel})
	if err != nil {
		t.Fatalf("update stage element: %v", err)
	}
	if updated.Label != newLabel {
		t.Fatalf("label = %q, want %q", updated.Label, newLabel)
	}
	if err := DeleteStageElement(context.Background(), pool, producer, showEl.ID); err != nil {
		t.Fatalf("delete stage element: %v", err)
	}
	if _, err := LoadStageElementByID(context.Background(), pool, showEl.ID); err == nil {
		t.Fatal("expected deleted element to be gone")
	}

	outsider := insertScenesTestUser(t, pool, "sce_outsider")
	if _, err := CreateSceneStageElement(context.Background(), pool, outsider, scene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Should Fail",
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}
}

func TestStageElementBindingLifecycle(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "seb_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}
	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Binding Test Scene", Slug: "binding-test-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}
	el, err := CreateSceneStageElement(context.Background(), pool, producer, scene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Bindable Token",
	})
	if err != nil {
		t.Fatalf("create stage element: %v", err)
	}

	var interactionID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO participant_interactions (show_scene_placement_id, internal_name, stage_button_label, interaction_type)
		VALUES ($1, 'test_interaction', 'Test', 'open_equip_mode')
		RETURNING id::text
	`, placement.ID).Scan(&interactionID); err != nil {
		t.Fatalf("insert participant_interaction fixture: %v", err)
	}

	if _, err := CreateStageElementBinding(context.Background(), pool, producer, el.ID, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Fatal("expected a nonexistent participant_interaction_id to be rejected")
	}

	binding, err := CreateStageElementBinding(context.Background(), pool, producer, el.ID, interactionID)
	if err != nil {
		t.Fatalf("create binding: %v", err)
	}
	if binding.ParticipantInteractionID != interactionID {
		t.Fatalf("binding interaction id = %q, want %q", binding.ParticipantInteractionID, interactionID)
	}

	comp, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition: %v", err)
	}
	found := false
	for _, e := range comp.Elements {
		if e.ID == el.ID {
			found = true
			if e.Binding == nil || e.Binding.ParticipantInteractionID != interactionID {
				t.Fatalf("expected resolved composition to include the binding on element %s", el.ID)
			}
		}
	}
	if !found {
		t.Fatal("expected bound element in resolved composition")
	}

	if err := DeleteStageElementBinding(context.Background(), pool, producer, el.ID); err != nil {
		t.Fatalf("delete binding: %v", err)
	}
	comp2, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition after unbind: %v", err)
	}
	for _, e := range comp2.Elements {
		if e.ID == el.ID && e.Binding != nil {
			t.Fatal("expected binding to be gone after DeleteStageElementBinding")
		}
	}
}

func TestSceneComposerEnabledFailsClosedForUnknownVenue(t *testing.T) {
	pool := openScenesTestPool(t)
	enabled, err := SceneComposerEnabled(context.Background(), pool, "no-such-venue-xyz")
	if err != nil {
		t.Fatalf("SceneComposerEnabled: %v", err)
	}
	if enabled {
		t.Fatal("expected fail-closed FALSE for an unknown venue slug")
	}
	enabled, err = SceneComposerEnabled(context.Background(), pool, "catharsis")
	if err != nil {
		t.Fatalf("SceneComposerEnabled: %v", err)
	}
	if !enabled {
		t.Fatal("expected catharsis to have scene_composer_enabled TRUE (migration 064)")
	}
}

// TestUpdateStageElementPositionRoundTripsExactly is the backend-level
// proof standing in for Kernel 73A item 5's "prove exact save/reload
// restoration" -- no browser automation tooling exists in this
// environment (Construction/OperatorLogs precedent), so this drives the
// exact same PATCH /api/stage-elements/{id} write path
// frontend/venues/show-runs/show.html's drag-surface and precision X/Y
// inputs both call, then reloads via a fresh LoadSceneStageElements read
// (not a cached client value) and asserts the float survives untouched.
func TestUpdateStageElementPositionRoundTripsExactly(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sce_roundtrip_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}
	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Roundtrip Scene", Slug: "roundtrip-scene-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	el, err := CreateSceneStageElement(context.Background(), pool, producer, scene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Precision Token", Position: map[string]any{"x": 0.5, "y": 0.5},
	})
	if err != nil {
		t.Fatalf("create stage element: %v", err)
	}

	wantPosition := map[string]any{"x": 0.337, "y": 0.881}
	updated, err := UpdateStageElement(context.Background(), pool, producer, el.ID, UpdateStageElementPatch{Position: &wantPosition})
	if err != nil {
		t.Fatalf("update stage element position: %v", err)
	}
	var gotFromUpdate map[string]any
	if err := json.Unmarshal(updated.Position, &gotFromUpdate); err != nil {
		t.Fatalf("unmarshal updated position: %v", err)
	}
	if gotFromUpdate["x"] != 0.337 || gotFromUpdate["y"] != 0.881 {
		t.Fatalf("position from the update response = %+v, want x=0.337 y=0.881", gotFromUpdate)
	}

	// Reload independently (a fresh Load, not the value UpdateStageElement
	// happened to return) -- this is the "reload" half of save/reload.
	reloaded, err := LoadStageElementByID(context.Background(), pool, el.ID)
	if err != nil {
		t.Fatalf("reload stage element: %v", err)
	}
	var gotFromReload map[string]any
	if err := json.Unmarshal(reloaded.Position, &gotFromReload); err != nil {
		t.Fatalf("unmarshal reloaded position: %v", err)
	}
	if gotFromReload["x"] != 0.337 || gotFromReload["y"] != 0.881 {
		t.Fatalf("position after independent reload = %+v, want x=0.337 y=0.881", gotFromReload)
	}
}
