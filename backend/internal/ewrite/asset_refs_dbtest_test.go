package ewrite

// Kernel 79: proves reconcilePublicationAssetRefs stays in sync with a
// publication's actual embedded images across saves, and that
// BackfillPublicationAssetRefs catches publications that predate the
// ewrite_publication_assets table (the live Core Rulebook's situation).
// Requires TEST_DATABASE_URL (internal/dbtest safety gate).

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func insertAssetRefTestAsset(t *testing.T, pool *pgxpool.Pool, locationID, userID string) string {
	t.Helper()
	var assetID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO assets (
			producer_user_id, location_id, uploader_user_id, owner_user_id, owner_state,
			width, height, byte_size, checksum_sha256, storage_root, original_path
		)
		VALUES ($1::uuid, $2::uuid, $1::uuid, $1::uuid, 'personal', 64, 64, 100, decode('ab', 'hex'), '/tmp', '/tmp/asset-ref-test-asset')
		RETURNING id::text
	`, userID, locationID).Scan(&assetID); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ewrite_publication_assets WHERE asset_id = $1`, assetID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM assets WHERE id = $1`, assetID)
	})
	return assetID
}

func boundAssetIDs(t *testing.T, pool *pgxpool.Pool, publicationID string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT asset_id::text FROM ewrite_publication_assets WHERE publication_id = $1
	`, publicationID)
	if err != nil {
		t.Fatalf("query bound assets: %v", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestReconcilePublicationAssetRefsTracksSaves(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "asset_ref_author")
	grantRole(t, pool, loc, crew, "producer")

	coll := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Asset Ref Ruleset")
	pub := mustCreatePublication(t, pool, crew, coll.ID, "Asset Ref Pub")

	assetA := insertAssetRefTestAsset(t, pool, loc, crew)
	assetB := insertAssetRefTestAsset(t, pool, loc, crew)

	source1 := "# Test\n\n![a](/api/assets/" + assetA + "/content)\n\n![b](/api/assets/" + assetB + "/content?variant=thumbnail)\n"
	saved1 := mustSave(t, pool, crew, pub.ID, source1, "")
	got := boundAssetIDs(t, pool, pub.ID)
	if len(got) != 2 {
		t.Fatalf("expected 2 bound assets after first save, got %d (%v)", len(got), got)
	}

	// Removing one image from the source must drop its row on the next save.
	source2 := "# Test\n\n![a](/api/assets/" + assetA + "/content)\n"
	mustSave(t, pool, crew, pub.ID, source2, saved1.RevisionID)
	got = boundAssetIDs(t, pool, pub.ID)
	if len(got) != 1 || got[0] != assetA {
		t.Fatalf("expected only asset A bound after second save, got %v", got)
	}
}

func TestBackfillPublicationAssetRefsCoversPreExistingPublications(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "asset_ref_backfill_author")
	grantRole(t, pool, loc, crew, "producer")

	coll := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Asset Ref Backfill Ruleset")
	pub := mustCreatePublication(t, pool, crew, coll.ID, "Asset Ref Backfill Pub")
	asset := insertAssetRefTestAsset(t, pool, loc, crew)

	// Simulate a publication that predates ewrite_publication_assets: write
	// source_markdown directly (as the old Kernel 78 seed path did), never
	// going through SavePublicationSource, so no reconcile has ever run.
	source := "# Predates The Table\n\n![img](/api/assets/" + asset + "/content)\n"
	if _, err := pool.Exec(ctx, `UPDATE ewrite_publications SET source_markdown = $2 WHERE id = $1`, pub.ID, source); err != nil {
		t.Fatalf("simulate pre-existing source: %v", err)
	}
	if got := boundAssetIDs(t, pool, pub.ID); len(got) != 0 {
		t.Fatalf("expected zero bound assets before backfill, got %v", got)
	}

	if err := BackfillPublicationAssetRefs(ctx, pool); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	got := boundAssetIDs(t, pool, pub.ID)
	if len(got) != 1 || got[0] != asset {
		t.Fatalf("expected the backfill to bind the referenced asset, got %v", got)
	}

	// Idempotent: a publication that already has rows is left alone on a
	// second call (no error, no duplicate work).
	if err := BackfillPublicationAssetRefs(ctx, pool); err != nil {
		t.Fatalf("second backfill call: %v", err)
	}
	if got := boundAssetIDs(t, pool, pub.ID); len(got) != 1 {
		t.Fatalf("expected exactly 1 bound asset after second backfill, got %d", len(got))
	}
}
