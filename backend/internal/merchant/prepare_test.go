package merchant

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// TestPrepareLockedCourtyardOpeningIsIdempotentAndReachable drives the
// Kernel 73A "Prepare Locked Courtyard Opening" action against the real
// migrated test database and the real seeded Courtyard Scene/Kessa
// merchant packet, proving: (1) a fresh Show becomes fully reachable in
// one call, (2) calling it again is a true no-op (nothing double-created),
// (3) a non-Director caller is refused.
func TestPrepareLockedCourtyardOpeningIsIdempotentAndReachable(t *testing.T) {
	// dbtest.OpenTestPool already registers t.Cleanup(pool.Close) -- an
	// extra `defer pool.Close()` here would run before any t.Cleanup
	// (plain defers in a test body always execute before the testing
	// framework's registered Cleanup funcs), closing the pool before this
	// test's own row-cleanup Cleanup below gets a chance to run its
	// deletes, silently orphaning rows across test runs.
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	suffix := time.Now().UTC().Format("150405.000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("lookup location: %v", err)
	}

	var directorUserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
	`, "k73a_prep_director_"+suffix).Scan(&directorUserID); err != nil {
		t.Fatalf("insert director user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role) VALUES ($1, $2, 'director')
	`, locationID, directorUserID); err != nil {
		t.Fatalf("grant director role: %v", err)
	}

	var outsiderUserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
	`, "k73a_prep_outsider_"+suffix).Scan(&outsiderUserID); err != nil {
		t.Fatalf("insert outsider user: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K73A Prep Production "+suffix, "k73a-prep-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "K73A Prep Run "+suffix, "k73a-prep-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "k73a-prep-show-"+suffix, "K73A Prep Show", directorUserID).Scan(&showID); err != nil {
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
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id IN ($1, $2)`, directorUserID, outsiderUserID)
	})

	if _, err := PrepareLockedCourtyardOpening(ctx, pool, outsiderUserID, showID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for a non-Director caller, got %v", err)
	}

	first, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("first PrepareLockedCourtyardOpening: %v", err)
	}
	if !first.SceneFound {
		t.Fatal("expected the seeded courtyard Scene to be found")
	}
	if !first.PlacementCreated {
		t.Fatal("expected a new placement to be created on first call")
	}
	if !first.InteractionCreated {
		t.Fatal("expected a new Kessa interaction to be created on first call")
	}
	if !first.StageElementCreated {
		t.Fatal("expected a new Kessa stage element to be created on first call")
	}
	var kessaContentURL, kessaThumbnailURL string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(data->>'asset_content_url', ''), COALESCE(data->>'asset_thumbnail_url', '')
		FROM scene_stage_elements WHERE id = $1
	`, first.StageElementID).Scan(&kessaContentURL, &kessaThumbnailURL); err != nil {
		t.Fatalf("load Kessa token artwork: %v", err)
	}
	if kessaContentURL != kessaTokenAssetURL || kessaThumbnailURL != kessaTokenAssetURL {
		t.Fatalf("Kessa token artwork = %q/%q, want %q", kessaContentURL, kessaThumbnailURL, kessaTokenAssetURL)
	}
	if !first.BindingCreated {
		t.Fatal("expected a new binding to be created on first call")
	}
	if !first.MerchantPacketFound || first.MerchantPacketStockCount == 0 {
		t.Fatalf("expected the seeded Kessa merchant packet with stock, got found=%v stock=%d", first.MerchantPacketFound, first.MerchantPacketStockCount)
	}
	if !first.Ready {
		t.Fatalf("expected Ready=true after first call, gaps=%v", first.Gaps)
	}

	second, err := PrepareLockedCourtyardOpening(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("second PrepareLockedCourtyardOpening: %v", err)
	}
	if second.PlacementCreated || second.InteractionCreated || second.StageElementCreated || second.BindingCreated {
		t.Fatalf("expected the second call to be a pure no-op, got %+v", second)
	}
	if second.PlacementID != first.PlacementID || second.InteractionID != first.InteractionID ||
		second.StageElementID != first.StageElementID || second.BindingID != first.BindingID {
		t.Fatal("expected the second call to resolve to the exact same ids as the first")
	}
	if !second.Ready {
		t.Fatalf("expected Ready=true after second call, gaps=%v", second.Gaps)
	}

	diag, err := KessaReachabilityDiagnostics(ctx, pool, directorUserID, showID)
	if err != nil {
		t.Fatalf("KessaReachabilityDiagnostics: %v", err)
	}
	if !diag.Ready {
		t.Fatalf("expected diagnostics Ready=true, gaps=%v", diag.Gaps)
	}
	if diag.PlacementID != first.PlacementID {
		t.Fatalf("diagnostics placement id mismatch: %q vs %q", diag.PlacementID, first.PlacementID)
	}
}
