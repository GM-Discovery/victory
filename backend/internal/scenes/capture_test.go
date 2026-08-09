package scenes

import (
	"context"
	"testing"
)

func TestUpdateCurrentScenePromotesShowLayerIntoBase(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "cap_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Capture Test Scene", Slug: "capture-test-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	baseEl, err := CreateSceneStageElement(context.Background(), pool, producer, scene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Backdrop Token", Position: map[string]any{"x": 0.1, "y": 0.1},
	})
	if err != nil {
		t.Fatalf("create base element: %v", err)
	}
	showEl, err := CreatePlacementStageElement(context.Background(), pool, producer, placement.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Placed Character Token", Position: map[string]any{"x": 0.5, "y": 0.5},
	})
	if err != nil {
		t.Fatalf("create show-layer element: %v", err)
	}

	before, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition before: %v", err)
	}
	if len(before.Elements) != 2 {
		t.Fatalf("expected 2 elements before update, got %d", len(before.Elements))
	}

	if _, err := UpdateCurrentScene(context.Background(), pool, producer, placement.ID); err != nil {
		t.Fatalf("update current scene: %v", err)
	}

	// The show-layer element's id and content must survive the promotion
	// (a binding on it would too, if there were one) -- it becomes a
	// Base-layer row in place, not a delete+recreate.
	promoted, err := LoadStageElementByID(context.Background(), pool, showEl.ID)
	if err != nil {
		t.Fatalf("load promoted element: %v", err)
	}
	if promoted.Layer != "base" {
		t.Fatalf("expected promoted element to now be base layer, got %q", promoted.Layer)
	}
	if promoted.Label != "Placed Character Token" {
		t.Fatalf("expected promoted element content unchanged, got label %q", promoted.Label)
	}

	// The Scene's Base layer (no placement) must now contain both elements.
	baseElements, err := ListSceneStageElements(context.Background(), pool, scene.ID)
	if err != nil {
		t.Fatalf("list scene base elements: %v", err)
	}
	if len(baseElements) != 2 {
		t.Fatalf("expected 2 base-layer elements after update, got %d", len(baseElements))
	}

	// Resolving the placement's composition again must return the same 2
	// elements -- nothing was invented or duplicated by the fold.
	after, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition after: %v", err)
	}
	if len(after.Elements) != 2 {
		t.Fatalf("expected 2 elements after update (unchanged resolved view), got %d", len(after.Elements))
	}
	_ = baseEl
}

func TestSaveArrangementAsNewSceneLeavesOriginalUnchanged(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "cap_save_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}

	origScene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Original Scene", Slug: "original-scene-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create original scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: origScene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}
	if _, err := CreateSceneStageElement(context.Background(), pool, producer, origScene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Original Base Token",
	}); err != nil {
		t.Fatalf("create base element: %v", err)
	}
	if _, err := CreatePlacementStageElement(context.Background(), pool, producer, placement.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: "Arranged Token",
	}); err != nil {
		t.Fatalf("create show-layer element: %v", err)
	}

	newScene, err := SaveArrangementAsNewScene(context.Background(), pool, producer, placement.ID, "New Snapshot Scene", "new-snapshot-"+suffix)
	if err != nil {
		t.Fatalf("save arrangement as new scene: %v", err)
	}
	if newScene.ID == origScene.ID {
		t.Fatal("expected a distinct new Scene id")
	}

	newBaseElements, err := ListSceneStageElements(context.Background(), pool, newScene.ID)
	if err != nil {
		t.Fatalf("list new scene base elements: %v", err)
	}
	if len(newBaseElements) != 2 {
		t.Fatalf("expected 2 copied base elements on the new scene, got %d", len(newBaseElements))
	}

	// The original Scene's own Base layer must be untouched (still just the
	// 1 element it started with), and the original placement's Show-layer
	// element must still be there, still Show-layer.
	origBaseElements, err := ListSceneStageElements(context.Background(), pool, origScene.ID)
	if err != nil {
		t.Fatalf("list original scene base elements: %v", err)
	}
	if len(origBaseElements) != 1 {
		t.Fatalf("expected original scene's base layer unchanged at 1 element, got %d", len(origBaseElements))
	}
	origResolved, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load original resolved composition: %v", err)
	}
	if len(origResolved.Elements) != 2 {
		t.Fatalf("expected original placement's resolved composition still 2 elements, got %d", len(origResolved.Elements))
	}
}

func TestCaptureActionsRejectNonDirectorAuthority(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "cap_auth_producer")
	outsider := insertScenesTestUser(t, pool, "cap_auth_outsider")
	locationID, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}
	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Auth Scene", Slug: "auth-scene-" + suffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	if _, err := UpdateCurrentScene(context.Background(), pool, outsider, placement.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider update, got %v", err)
	}
	if _, err := SaveArrangementAsNewScene(context.Background(), pool, outsider, placement.ID, "Should Fail", "should-fail-"+suffix); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider save, got %v", err)
	}
}
