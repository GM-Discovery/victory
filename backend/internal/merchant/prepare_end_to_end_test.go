package merchant

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showings"
	"victory/backend/internal/world"
)

// TestPrepareLockedCourtyardOpeningEndToEndKessaReachability is Kernel
// 73A's full closed-loop proof for item 4 -- unlike TestKessaGoldenPath
// (which hand-builds its placement/interaction rows directly via SQL) and
// unlike TestPrepareLockedCourtyardOpeningIsIdempotentAndReachable (which
// only checks the Ready/Gaps diagnostics shape), this test:
//  1. Calls the real PrepareLockedCourtyardOpening idempotent action.
//  2. Sets that placement current and links a real session to the Show.
//  3. Loads a REAL world.LoadVenueSnapshot and confirms Kessa's composition
//     token appears with the binding pointing at the prepared interaction.
//  4. Drives OpenEquipMode through that EXACT bound interaction id for an
//     eligible rostered Player with a selected, owned Character --
//     succeeds and returns the real Kessa packet + seeded stock.
//  5. Proves an Audience-role viewer and a non-rostered outsider are both
//     refused through that same interaction id.
//  6. Proves the Courtyard stays the Show's current Scene throughout (Kernel
//     73 spec S2.1: opening/attempting Equip Mode never mutates
//     shows.current_show_scene_placement_id).
func TestPrepareLockedCourtyardOpeningEndToEndKessaReachability(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	suffix := time.Now().UTC().Format("150405.000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("lookup location: %v", err)
	}

	mustUser := func(handle string) string {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text`, handle).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", handle, err)
		}
		return id
	}
	directorUserID := mustUser("k73a_e2e_director_" + suffix)
	playerUserID := mustUser("k73a_e2e_player_" + suffix)
	audienceUserID := mustUser("k73a_e2e_audience_" + suffix)
	outsiderUserID := mustUser("k73a_e2e_outsider_" + suffix)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role) VALUES ($1, $2, 'director')
	`, locationID, directorUserID); err != nil {
		t.Fatalf("grant director role: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K73A E2E Production "+suffix, "k73a-e2e-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K73A E2E Run "+suffix, "k73a-e2e-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k73a-e2e-show-"+suffix, "K73A E2E Show", directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id) VALUES ($1, $2, 'player', $3)
	`, showRunID, playerUserID, directorUserID); err != nil {
		t.Fatalf("insert player roster row: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id) VALUES ($1, $2, 'audience', $3)
	`, showRunID, audienceUserID, directorUserID); err != nil {
		t.Fatalf("insert audience roster row: %v", err)
	}

	var characterCardID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, playerUserID, locationID, "K73A E2E Hero").Scan(&characterCardID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $2 WHERE show_run_id = $1 AND user_id = $3
	`, showRunID, characterCardID, playerUserID); err != nil {
		t.Fatalf("select character for roster row: %v", err)
	}

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `DELETE FROM sessions WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM venues WHERE slug = $1`, "k73a-e2e-stage-"+suffix)
		_, _ = pool.Exec(cctx, `
			DELETE FROM stage_element_bindings WHERE scene_stage_element_id IN (
				SELECT id FROM scene_stage_elements WHERE scene_id IN (SELECT id FROM scenes WHERE location_id = $1 AND slug = 'courtyard')
				AND show_scene_placement_id IS NULL
			) AND participant_interaction_id IN (
				SELECT id FROM participant_interactions WHERE show_scene_placement_id IN (SELECT id FROM show_scene_placements WHERE show_id = $2)
			)
		`, locationID, showID)
		_, _ = pool.Exec(cctx, `
			DELETE FROM scene_stage_elements WHERE scene_id IN (SELECT id FROM scenes WHERE location_id = $1 AND slug = 'courtyard')
			  AND show_scene_placement_id IS NULL AND label = 'Kessa'
		`, locationID)
		_, _ = pool.Exec(cctx, `DELETE FROM participant_interactions WHERE show_scene_placement_id IN (SELECT id FROM show_scene_placements WHERE show_id = $1)`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_scene_placements WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM character_cards WHERE id = $1`, characterCardID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_run_roster_members WHERE show_run_id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM shows WHERE id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_runs WHERE id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(cctx, `DELETE FROM location_memberships WHERE user_id = $1`, directorUserID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2, $3, $4)`, directorUserID, playerUserID, audienceUserID, outsiderUserID)
	})

	// Step 1: the real idempotent Director action.
	prep, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("PrepareLockedCourtyardOpening: %v", err)
	}
	if !prep.Ready {
		t.Fatalf("expected Ready=true, gaps=%v", prep.Gaps)
	}

	// Step 2: make the Courtyard current and link a real live session.
	// The session (and the snapshot read below) deliberately use a
	// dedicated throwaway venue, NOT the real "catharsis" row -- mirroring
	// golden_path_dbtest_test.go's own established reason: `go test ./...`
	// runs different packages concurrently against the one shared
	// victory_test database, and a rehearsal-status session held open on
	// the real catharsis venue could race another package's own
	// most-recent-active-session lookup for that same venue slug. The
	// Kessa merchant packet, participant_interactions_enabled, and
	// scene_composer_enabled flags are all location-scoped (or read off
	// the Scene's own default_venue_id for the composer flag), not tied
	// to which venue actually hosts the session, so this substitution
	// doesn't skip anything this test is meant to prove.
	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = $2 WHERE id = $1`, showID, prep.PlacementID); err != nil {
		t.Fatalf("set current placement: %v", err)
	}
	var lotID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM lots WHERE location_id = $1 AND slug = 'main-lot'`, locationID).Scan(&lotID); err != nil {
		t.Fatalf("lookup lot: %v", err)
	}
	fixtureVenueSlug := "k73a-e2e-stage-" + suffix
	var fixtureVenueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public)
		VALUES ($1, 'K73A E2E Stage', $2, 'plaza', '{"participant_interactions_enabled": true}'::jsonb, FALSE)
		RETURNING id::text
	`, lotID, fixtureVenueSlug).Scan(&fixtureVenueID); err != nil {
		t.Fatalf("insert fixture venue: %v", err)
	}
	// Override the placement's own venue -- resolvePlacementVenueSlug
	// (backend/internal/merchant/interactions.go) prefers this over the
	// Scene's own default_venue_id (catharsis).
	if _, err := pool.Exec(ctx, `UPDATE show_scene_placements SET venue_id = $2 WHERE id = $1`, prep.PlacementID, fixtureVenueID); err != nil {
		t.Fatalf("override placement venue: %v", err)
	}
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id) VALUES ($1, 'rehearsal', $2) RETURNING id::text
	`, fixtureVenueID, showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := showings.EnsureForSession(ctx, pool, sessionID, directorUserID); err != nil {
		t.Fatalf("ensure showing: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role) VALUES ($1, $2, 'cast'::location_role)
	`, sessionID, playerUserID); err != nil {
		t.Fatalf("insert session participant: %v", err)
	}

	// Step 3: the real live snapshot must show Kessa's token, bound to the
	// exact interaction Prepare created.
	snap, err := world.LoadVenueSnapshot(ctx, pool, "director", directorUserID, fixtureVenueSlug)
	if err != nil {
		t.Fatalf("LoadVenueSnapshot: %v", err)
	}
	if snap.Session.ID != sessionID {
		t.Fatalf("snapshot resolved session = %q, want %q", snap.Session.ID, sessionID)
	}
	if snap.Session.CurrentShowScenePlacementID != prep.PlacementID {
		t.Fatalf("snapshot current placement = %q, want %q", snap.Session.CurrentShowScenePlacementID, prep.PlacementID)
	}
	var kessaBoundInteractionID string
	for _, el := range snap.Elements {
		if el.Name == "Kessa" && el.ContextClass == "scene_composition" {
			if binding, ok := el.Data["binding"].(map[string]any); ok {
				kessaBoundInteractionID, _ = binding["participant_interaction_id"].(string)
			}
		}
	}
	if kessaBoundInteractionID == "" {
		t.Fatal("expected Kessa's token with a resolved binding to appear in the live snapshot")
	}
	if kessaBoundInteractionID != prep.InteractionID {
		t.Fatalf("snapshot's bound interaction id = %q, want the exact id Prepare created: %q", kessaBoundInteractionID, prep.InteractionID)
	}

	// Step 4: the eligible Player opens Equip Mode through that EXACT id.
	equip, err := OpenEquipMode(ctx, pool, playerUserID, kessaBoundInteractionID)
	if err != nil {
		t.Fatalf("OpenEquipMode for the eligible player: %v", err)
	}
	if equip.Packet.Slug != "kessa" {
		t.Fatalf("packet slug = %q, want kessa", equip.Packet.Slug)
	}
	if len(equip.Packet.Stock) == 0 {
		t.Fatal("expected Kessa's seeded stock to be non-empty")
	}
	if equip.CharacterCardID != characterCardID {
		t.Fatalf("character card id = %q, want %q", equip.CharacterCardID, characterCardID)
	}

	// Step 5: Audience and a total outsider are both refused through the
	// same real interaction id.
	if _, err := OpenEquipMode(ctx, pool, audienceUserID, kessaBoundInteractionID); err == nil {
		t.Fatal("expected the Audience-role viewer to be refused")
	}
	if _, err := OpenEquipMode(ctx, pool, outsiderUserID, kessaBoundInteractionID); err == nil {
		t.Fatal("expected the non-rostered outsider to be refused")
	}

	// Step 6: the Courtyard is still current -- opening/attempting never
	// mutated the Show's stage pointer.
	var currentPlacementAfter *string
	if err := pool.QueryRow(ctx, `SELECT current_show_scene_placement_id::text FROM shows WHERE id = $1`, showID).Scan(&currentPlacementAfter); err != nil {
		t.Fatalf("reload show current placement: %v", err)
	}
	if currentPlacementAfter == nil || *currentPlacementAfter != prep.PlacementID {
		t.Fatalf("expected the Courtyard to remain the current Scene, got %v", currentPlacementAfter)
	}
}
