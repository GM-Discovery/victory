package world

import (
	"context"
	"testing"
)

// TestLoadVenueSnapshotIncludesCurrentSceneComposition is the Kernel 73A
// integration proof for the seam that replaced pass one's manual "Load
// Composition Into Live Session" button and its 'scene_composition_loaded'
// stopgap action: once a Show has a current Scene Placement, that
// placement's resolved (Base + Show layer) scene_stage_elements
// composition must appear in the SAME Elements array
// world.LoadVenueSnapshot already returns for warehouse-asset tokens and
// index cards -- no separate endpoint, no client-side action-type
// interpretation, just part of what a snapshot *is*.
func TestLoadVenueSnapshotIncludesCurrentSceneComposition(t *testing.T) {
	pool := openWorldTestPool(t)
	actor := insertWorldTestUser(t, pool, "wc_scene_actor")
	fx := buildWorldFixture(t, pool, actor)
	sessionID := insertSession(t, pool, fx.venueID, fx.showID, "rehearsal")

	ctx := context.Background()

	var sceneID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO scenes (location_id, slug, title, status, default_venue_id)
		VALUES ($1, 'wc-composition-scene', 'WC Composition Scene', 'ready', $2)
		RETURNING id::text
	`, fx.locationID, fx.venueID).Scan(&sceneID); err != nil {
		t.Fatalf("insert scene: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE id = $1`, sceneID) })

	var placementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, status)
		VALUES ($1, $2, 'ready')
		RETURNING id::text
	`, fx.showID, sceneID).Scan(&placementID); err != nil {
		t.Fatalf("insert placement: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE shows SET current_show_scene_placement_id = $2 WHERE id = $1
	`, fx.showID, placementID); err != nil {
		t.Fatalf("set current placement: %v", err)
	}

	// A Base-layer token, an unbound Base-layer index card, and a
	// participant_interactions row + binding proving the reachability
	// chain the real Kessa proof needs.
	var tokenID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO scene_stage_elements (scene_id, kind, label, position)
		VALUES ($1, 'token', 'Kessa', '{"x":0.4,"y":0.6}'::jsonb)
		RETURNING id::text
	`, sceneID).Scan(&tokenID); err != nil {
		t.Fatalf("insert token: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO scene_stage_elements (scene_id, kind, label, position)
		VALUES ($1, 'index_card', 'A Note', '{"x":0.2,"y":0.2}'::jsonb)
	`, sceneID); err != nil {
		t.Fatalf("insert index card: %v", err)
	}

	var interactionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO participant_interactions (show_scene_placement_id, internal_name, stage_button_label, interaction_type)
		VALUES ($1, 'talk_to_kessa', 'Talk to Kessa', 'open_equip_mode')
		RETURNING id::text
	`, placementID).Scan(&interactionID); err != nil {
		t.Fatalf("insert participant_interaction: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO stage_element_bindings (scene_stage_element_id, binding_type, participant_interaction_id)
		VALUES ($1, 'participant_interaction', $2)
	`, tokenID, interactionID); err != nil {
		t.Fatalf("insert binding: %v", err)
	}

	snap, err := LoadVenueSnapshot(ctx, pool, "director", actor, fx.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot: %v", err)
	}
	if snap.Session.ID != sessionID {
		t.Fatalf("expected the freshly created session to be the resolved active session, got %q want %q", snap.Session.ID, sessionID)
	}
	if snap.Session.CurrentShowScenePlacementID != placementID {
		t.Fatalf("snapshot's current_show_scene_placement_id = %q, want %q", snap.Session.CurrentShowScenePlacementID, placementID)
	}

	var kessaEl *PlacedElement
	var noteEl *PlacedElement
	for i := range snap.Elements {
		el := &snap.Elements[i]
		if el.Name == "Kessa" {
			kessaEl = el
		}
		if el.Name == "A Note" {
			noteEl = el
		}
	}
	if kessaEl == nil {
		t.Fatal("expected the Kessa scene composition token to appear in the live snapshot's Elements")
	}
	if kessaEl.ContextClass != "scene_composition" {
		t.Fatalf("Kessa element context_class = %q, want scene_composition", kessaEl.ContextClass)
	}
	if kessaEl.ElementType != "scene_token" {
		t.Fatalf("Kessa element_type = %q, want scene_token", kessaEl.ElementType)
	}
	if kessaEl.ElementID != "scene:"+tokenID {
		t.Fatalf("Kessa element id = %q, want scene:%s", kessaEl.ElementID, tokenID)
	}
	binding, ok := kessaEl.Data["binding"].(map[string]any)
	if !ok {
		t.Fatalf("expected Kessa's element Data to include a binding map, got %#v", kessaEl.Data)
	}
	if binding["participant_interaction_id"] != interactionID {
		t.Fatalf("bound interaction id = %v, want %q", binding["participant_interaction_id"], interactionID)
	}
	if binding["stage_button_label"] != "Talk to Kessa" {
		t.Fatalf("bound interaction stage_button_label = %v, want %q", binding["stage_button_label"], "Talk to Kessa")
	}
	if binding["enabled"] != true {
		t.Fatalf("bound interaction enabled = %v, want true", binding["enabled"])
	}
	if x, _ := kessaEl.Position["x"].(float64); x != 0.4 {
		t.Fatalf("Kessa position.x = %v, want 0.4", kessaEl.Position["x"])
	}

	if noteEl == nil {
		t.Fatal("expected the unbound index card to appear in the snapshot too")
	}
	if noteEl.Data["binding"] != nil {
		t.Fatal("expected the unbound index card to have no binding")
	}

	// Clearing the current-Scene pointer must remove the composition layer
	// again -- it is not sticky/cached once the Show moves off this Scene.
	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = NULL WHERE id = $1`, fx.showID); err != nil {
		t.Fatalf("clear current placement: %v", err)
	}
	snap2, err := LoadVenueSnapshot(ctx, pool, "director", actor, fx.venueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot after clearing current scene: %v", err)
	}
	for _, el := range snap2.Elements {
		if el.Name == "Kessa" {
			t.Fatal("expected Kessa's composition token to disappear once the Show's current Scene is cleared")
		}
	}
}
