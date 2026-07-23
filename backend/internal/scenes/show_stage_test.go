package scenes

import (
	"context"
	"testing"

	"victory/backend/internal/shows"
)

// TestSetCurrentScenePlacement and friends live in the scenes package (not
// shows) because they exercise shows.SetCurrentScenePlacement against a
// real scenes.CreatePlacement/ArchivePlacement fixture -- scenes already
// imports shows in production code, so this direction has no cycle,
// whereas a shows-package test file importing scenes would (scenes
// imports shows).

func TestSetCurrentScenePlacementRequiresPlacementBelongToShow(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_stage_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	_, showID := showFixture(t, pool, producer, productionID)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Stage Test Scene", Slug: "stage-test-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	updated, err := shows.SetCurrentScenePlacement(context.Background(), pool, producer, showID, placement.ID)
	if err != nil {
		t.Fatalf("set current scene placement: %v", err)
	}
	if updated.CurrentShowScenePlacementID == nil || *updated.CurrentShowScenePlacementID != placement.ID {
		t.Fatalf("expected current_show_scene_placement_id = %q, got %v", placement.ID, updated.CurrentShowScenePlacementID)
	}

	// A placement belonging to a different Show is rejected.
	_, otherShowID := showFixture(t, pool, producer, productionID)
	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, producer, otherShowID, placement.ID); err == nil || err.Error() != "placement_show_mismatch" {
		t.Fatalf("expected placement_show_mismatch, got %v", err)
	}
}

func TestSetCurrentScenePlacementRejectsArchivedPlacement(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_stage_archived_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	_, showID := showFixture(t, pool, producer, productionID)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Archivable Stage Scene", Slug: "archivable-stage-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	if _, err := ArchivePlacement(context.Background(), pool, producer, placement.ID); err != nil {
		t.Fatalf("archive placement: %v", err)
	}
	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, producer, showID, placement.ID); err == nil || err.Error() != "placement_not_eligible" {
		t.Fatalf("expected placement_not_eligible for an archived placement, got %v", err)
	}
}

func TestClearCurrentScenePlacement(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_stage_clear_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	_, showID := showFixture(t, pool, producer, productionID)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Clearable Stage Scene", Slug: "clearable-stage-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, producer, showID, placement.ID); err != nil {
		t.Fatalf("set current scene placement: %v", err)
	}
	cleared, err := shows.ClearCurrentScenePlacement(context.Background(), pool, producer, showID)
	if err != nil {
		t.Fatalf("clear current scene placement: %v", err)
	}
	if cleared.CurrentShowScenePlacementID != nil {
		t.Fatalf("expected current_show_scene_placement_id cleared, got %v", cleared.CurrentShowScenePlacementID)
	}
}

// TestArchivingCurrentPlacementClearsShowPointer proves
// scenes.ArchivePlacement's side effect (Kernel 70 SS4.1's "archiving the
// current placement... must resolve the pointer safely and explicitly")
// actually clears a Show's pointer when it targets that exact placement.
func TestArchivingCurrentPlacementClearsShowPointer(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_stage_archive_current_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	_, showID := showFixture(t, pool, producer, productionID)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Current Then Archived Scene", Slug: "current-then-archived-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, producer, showID, placement.ID); err != nil {
		t.Fatalf("set current scene placement: %v", err)
	}

	if _, err := ArchivePlacement(context.Background(), pool, producer, placement.ID); err != nil {
		t.Fatalf("archive current placement: %v", err)
	}

	reloaded, err := shows.LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("reload show: %v", err)
	}
	if reloaded.CurrentShowScenePlacementID != nil {
		t.Fatalf("expected archiving the current placement to clear the Show's pointer, got %v", reloaded.CurrentShowScenePlacementID)
	}
}

func TestSetCurrentScenePlacementRequiresManageAuthority(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_stage_auth_producer")
	locationID, productionID := productionFixture(t, pool, producer)
	_, showID := showFixture(t, pool, producer, productionID)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, locationID, productionID, CreateSceneInput{
		Title: "Auth Stage Scene", Slug: "auth-stage-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	outsider := insertScenesTestUser(t, pool, "sc_stage_auth_outsider")
	if _, err := shows.SetCurrentScenePlacement(context.Background(), pool, outsider, showID, placement.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}
}

// The former TestProjectCompositionIntoSessionRequiresActiveSession and
// TestProjectCompositionIntoSessionAppendsActionForLinkedSession tests
// lived here, covering the now-removed ProjectCompositionIntoSession
// stopgap. The real integration is covered instead by
// backend/internal/world's TestLoadVenueSnapshotIncludesCurrentSceneComposition
// (and its Kessa-binding sibling), which drive the actual
// world.LoadVenueSnapshot seam that replaced it.
