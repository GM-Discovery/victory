package merchant

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// TestPreviewInteractionIsReadOnlyAndBackstageGated proves Kernel 73A item
// 7: PreviewInteraction (a) is backstage-authority gated -- an Audience
// viewer and a non-rostered outsider are both refused, a Director is
// allowed -- the OPPOSITE gate from OpenEquipMode's participant
// eligibility, so a Preview caller is never treated as an eligible Player;
// and (b) never creates or changes any row anywhere -- no character
// inventory, no roster row, no session/showing row -- proving "Preview as
// Player" is structurally incapable of purchasing/rolling/participating,
// not merely "the test client doesn't happen to call those endpoints."
func TestPreviewInteractionIsReadOnlyAndBackstageGated(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	suffix := time.Now().UTC().Format("150405.000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("lookup location: %v", err)
	}

	var directorUserID, audienceUserID, outsiderUserID string
	for _, pair := range []struct {
		handle string
		dest   *string
	}{
		{"k73a_preview_director_" + suffix, &directorUserID},
		{"k73a_preview_audience_" + suffix, &audienceUserID},
		{"k73a_preview_outsider_" + suffix, &outsiderUserID},
	} {
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
		`, pair.handle).Scan(pair.dest); err != nil {
			t.Fatalf("insert user %s: %v", pair.handle, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role) VALUES ($1, $2, 'director')
	`, locationID, directorUserID); err != nil {
		t.Fatalf("grant director role: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K73A Preview Production "+suffix, "k73a-preview-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K73A Preview Run "+suffix, "k73a-preview-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k73a-preview-show-"+suffix, "K73A Preview Show", directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `
			DELETE FROM stage_element_bindings WHERE scene_stage_element_id IN (
				SELECT id FROM scene_stage_elements WHERE scene_id IN (SELECT id FROM scenes WHERE location_id = $1 AND slug = 'courtyard')
				AND show_scene_placement_id IS NULL
			) AND participant_interaction_id IN (
				SELECT id FROM participant_interactions WHERE show_scene_placement_id IN (
					SELECT id FROM show_scene_placements WHERE show_id = $2
				)
			)
		`, locationID, showID)
		_, _ = pool.Exec(cctx, `
			DELETE FROM scene_stage_elements WHERE scene_id IN (SELECT id FROM scenes WHERE location_id = $1 AND slug = 'courtyard')
			  AND show_scene_placement_id IS NULL AND label = 'Kessa'
		`, locationID)
		_, _ = pool.Exec(cctx, `
			DELETE FROM participant_interactions WHERE show_scene_placement_id IN (
				SELECT id FROM show_scene_placements WHERE show_id = $1
			)
		`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_scene_placements WHERE show_id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM shows WHERE id = $1`, showID)
		_, _ = pool.Exec(cctx, `DELETE FROM show_runs WHERE id = $1`, showRunID)
		_, _ = pool.Exec(cctx, `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(cctx, `DELETE FROM location_memberships WHERE user_id = $1`, directorUserID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2, $3)`, directorUserID, audienceUserID, outsiderUserID)
	})

	prep, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("PrepareLockedCourtyardOpening: %v", err)
	}
	if !prep.Ready {
		t.Fatalf("expected the Courtyard/Kessa chain to be ready before testing preview, gaps=%v", prep.Gaps)
	}

	// Baseline row counts, taken AFTER setup (which itself only inserts
	// placement/interaction/stage-element/binding rows, none of which are
	// inventory/roster/session rows) -- proves preview adds nothing further.
	countRows := func(table, where string, args ...any) int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	inventoryBefore := countRows("character_inventory_items", "acquired_by_user_id = ANY($1)", []string{directorUserID, audienceUserID, outsiderUserID})
	rosterBefore := countRows("show_run_roster_members", "show_run_id = $1", showRunID)
	sessionsBefore := countRows("sessions", "show_id = $1", showID)

	// Director (backstage) is allowed.
	preview, err := PreviewInteraction(ctx, pool, directorUserID, prep.InteractionID)
	if err != nil {
		t.Fatalf("PreviewInteraction as director: %v", err)
	}
	if preview.PacketDisplay != "Kessa" {
		t.Fatalf("preview packet display = %q, want Kessa", preview.PacketDisplay)
	}
	if len(preview.StockNames) == 0 {
		t.Fatal("expected preview to include Kessa's seeded stock names")
	}
	if !preview.Enabled {
		t.Fatal("expected the prepared interaction to be enabled")
	}

	// Audience (not backstage) is refused.
	if _, err := PreviewInteraction(ctx, pool, audienceUserID, prep.InteractionID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for an audience-role viewer, got %v", err)
	}
	// A total outsider with no role anywhere at this location is refused.
	if _, err := PreviewInteraction(ctx, pool, outsiderUserID, prep.InteractionID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for an outsider, got %v", err)
	}
	// Anonymous is refused.
	if _, err := PreviewInteraction(ctx, pool, "", prep.InteractionID); err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("expected not_authenticated for an empty actor, got %v", err)
	}

	inventoryAfter := countRows("character_inventory_items", "acquired_by_user_id = ANY($1)", []string{directorUserID, audienceUserID, outsiderUserID})
	rosterAfter := countRows("show_run_roster_members", "show_run_id = $1", showRunID)
	sessionsAfter := countRows("sessions", "show_id = $1", showID)
	if inventoryAfter != inventoryBefore {
		t.Fatalf("expected zero inventory rows created by preview, before=%d after=%d", inventoryBefore, inventoryAfter)
	}
	if rosterAfter != rosterBefore {
		t.Fatalf("expected zero roster rows created by preview, before=%d after=%d", rosterBefore, rosterAfter)
	}
	if sessionsAfter != sessionsBefore {
		t.Fatalf("expected zero session rows created by preview, before=%d after=%d", sessionsBefore, sessionsAfter)
	}
}
