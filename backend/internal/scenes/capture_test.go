package scenes

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// plantLiveCatharsisToken inserts a real token directly onto Catharsis's
// live stage (elements + venue_layout_elements), the actual source
// CaptureLiveVenueComposition reads from since Kernel 93 -- UpdateCurrentScene
// and SaveArrangementAsNewScene no longer read scene_stage_elements as their
// input at all (see live_bridge.go's own header comment). liveBridgeVenueSlug
// is hardcoded in production code (capture.go), so there is no way to point
// Capture at a disposable per-test venue the way most other fixtures in this
// package do -- this plants onto the one real, shared Catharsis venue
// instead, and cleans up after itself so it never lingers for another test
// or a real dev/prod use of this database.
func plantLiveCatharsisToken(t *testing.T, pool *pgxpool.Pool, label string) {
	t.Helper()
	ctx := context.Background()

	var venueID, libraryID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("lookup catharsis venue: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM libraries WHERE location_id = (SELECT l.location_id FROM venues v JOIN lots l ON l.id = v.lot_id WHERE v.id = $1) LIMIT 1
	`, venueID).Scan(&libraryID); err != nil {
		t.Fatalf("lookup catharsis library: %v", err)
	}

	var elementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO elements (library_id, name, slug, element_type, data)
		VALUES ($1, $2, $3, 'token', '{}'::jsonb)
		RETURNING id::text
	`, libraryID, label, "capture-test-live-"+testSuffix(t)).Scan(&elementID); err != nil {
		t.Fatalf("insert live element: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM elements WHERE id = $1`, elementID) })

	if _, err := pool.Exec(ctx, `
		INSERT INTO venue_layout_elements (venue_id, element_id, surface, position, visibility, is_default)
		VALUES ($1, $2, 'stage', '{"anchor":"stage","x":10,"y":10}'::jsonb,
		        '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[],"visible":true}'::jsonb, FALSE)
	`, venueID, elementID); err != nil {
		t.Fatalf("place live element on stage: %v", err)
	}
	// venue_layout_elements rows cascade-delete with their element (migrations/
	// 001_init.sql), so no separate cleanup is needed for that row.
}

// findStageElementByLabel scans a captured scene's own stage elements for
// one with the given label -- the assertion shape this whole file now
// needs, since Capture reads from the shared live Catharsis venue rather
// than an isolated per-test fixture, and another real element could
// legitimately already be on that stage from something else.
func findStageElementByLabel(elements []StageElement, label string) (StageElement, bool) {
	for _, el := range elements {
		if el.Label == label {
			return el, true
		}
	}
	return StageElement{}, false
}

// TestUpdateCurrentSceneCapturesLiveVenueIntoBase is the Kernel 101 (101-11)
// rewrite of a test that exercised a model Kernel 93 already replaced two
// weeks earlier than this rewrite: UpdateCurrentScene used to promote
// whatever scene_stage_elements a placement already held into the Scene's
// Base layer. It no longer does -- it captures whatever is actually live on
// Catharsis's real stage right now (CaptureLiveVenueComposition), replacing
// the Scene's entire Base layer with that snapshot. This proves the real,
// current contract instead of the retired one.
func TestUpdateCurrentSceneCapturesLiveVenueIntoBase(t *testing.T) {
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

	liveLabel := "Live Stage Token " + suffix
	plantLiveCatharsisToken(t, pool, liveLabel)

	if _, err := UpdateCurrentScene(context.Background(), pool, producer, placement.ID); err != nil {
		t.Fatalf("update current scene: %v", err)
	}

	// The Scene's Base layer must now contain what was actually live on
	// stage, captured as a Base-layer row (no placement).
	baseElements, err := ListSceneStageElements(context.Background(), pool, scene.ID)
	if err != nil {
		t.Fatalf("list scene base elements: %v", err)
	}
	captured, found := findStageElementByLabel(baseElements, liveLabel)
	if !found {
		t.Fatalf("expected the live stage token %q to be captured into the Scene's Base layer, got %+v", liveLabel, baseElements)
	}
	if captured.Layer != "base" {
		t.Fatalf("expected captured element to be base layer, got %q", captured.Layer)
	}
	if captured.Kind != StageElementKindToken {
		t.Fatalf("expected captured element kind %q, got %q", StageElementKindToken, captured.Kind)
	}

	// Resolving the placement's composition must surface the same captured
	// content -- nothing was invented or lost between Base layer and the
	// resolved view a Show-layer placement with no overrides falls back to.
	resolved, err := LoadResolvedComposition(context.Background(), pool, placement.ID)
	if err != nil {
		t.Fatalf("load resolved composition: %v", err)
	}
	if _, found := findStageElementByLabel(resolved.Elements, liveLabel); !found {
		t.Fatalf("expected the captured live token to appear in the resolved composition, got %+v", resolved.Elements)
	}
}

// TestSaveArrangementAsNewSceneLeavesOriginalUnchanged is the Kernel 101
// (101-11) rewrite of the same stale-model test as above, applied to
// SaveArrangementAsNewScene: it captures the live Catharsis stage into a
// brand new Scene, and must never touch the original Scene's own Base
// layer while doing it.
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
	origLabel := "Original Base Token " + suffix
	if _, err := CreateSceneStageElement(context.Background(), pool, producer, origScene.ID, CreateStageElementInput{
		Kind: StageElementKindToken, Label: origLabel,
	}); err != nil {
		t.Fatalf("create original base element: %v", err)
	}

	liveLabel := "Live Stage Token " + suffix
	plantLiveCatharsisToken(t, pool, liveLabel)

	newScene, err := SaveArrangementAsNewScene(context.Background(), pool, producer, placement.ID, "New Snapshot Scene", "new-snapshot-"+suffix)
	if err != nil {
		t.Fatalf("save arrangement as new scene: %v", err)
	}
	if newScene.ID == origScene.ID {
		t.Fatal("expected a distinct new Scene id")
	}

	// The new Scene's Base layer must hold what was actually live on stage
	// when it was captured.
	newBaseElements, err := ListSceneStageElements(context.Background(), pool, newScene.ID)
	if err != nil {
		t.Fatalf("list new scene base elements: %v", err)
	}
	if _, found := findStageElementByLabel(newBaseElements, liveLabel); !found {
		t.Fatalf("expected the live stage token %q to be captured into the new Scene, got %+v", liveLabel, newBaseElements)
	}

	// The original Scene's own Base layer must be completely untouched --
	// still just the one element it started with, and it must never have
	// picked up the live token that only the NEW scene captured.
	origBaseElements, err := ListSceneStageElements(context.Background(), pool, origScene.ID)
	if err != nil {
		t.Fatalf("list original scene base elements: %v", err)
	}
	if len(origBaseElements) != 1 {
		t.Fatalf("expected original scene's base layer unchanged at 1 element, got %d: %+v", len(origBaseElements), origBaseElements)
	}
	if origBaseElements[0].Label != origLabel {
		t.Fatalf("expected original scene's own element %q untouched, got %q", origLabel, origBaseElements[0].Label)
	}
	if _, found := findStageElementByLabel(origBaseElements, liveLabel); found {
		t.Fatal("the original Scene must never pick up the live token that only SaveArrangementAsNewScene's target (the new Scene) captured")
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
