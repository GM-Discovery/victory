package assets

// Kernel 81: proves the fix for a real bug found via Playwright testing --
// a Storyboard grant holder with no location_memberships relationship to
// the board owner's fallback storage location was 403'd loading a card
// image they were fully authorized to see on the board itself.
// storyboardCardImageViewerAllowed (read.go) closes this without assets
// importing the storyboards package (would close a direct two-package
// cycle, since storyboards already imports assets for the upload helper).
//
// Requires TEST_DATABASE_URL (internal/dbtest safety gate). Reuses the
// location/user/asset helpers from ewrite_visibility_dbtest_test.go.

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestStoryboardCardImageViewerAllowed_GrantHolderWithNoLocationMembershipCanRead(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	locationID := insertEwriteVizLocation(t, pool, "K81 Image Viz Location")
	owner := insertEwriteVizUser(t, pool, "k81_owner")
	assetID := insertEwriteVizAsset(t, pool, locationID, owner)

	var boardID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboards (owner_user_id, title) VALUES ($1, 'K81 Viz Board') RETURNING id::text
	`, owner).Scan(&boardID); err != nil {
		t.Fatalf("insert storyboard: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM storyboards WHERE id = $1`, boardID) })

	var columnID, bandID, rowID string
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_columns (storyboard_id, title, sort_order) VALUES ($1, 'Col', 0) RETURNING id::text`, boardID).Scan(&columnID); err != nil {
		t.Fatalf("insert column: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_bands (storyboard_id, label, sort_order) VALUES ($1, 'Band', 0) RETURNING id::text`, boardID).Scan(&bandID); err != nil {
		t.Fatalf("insert band: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_rows (storyboard_id, band_id, label, sort_order_in_band) VALUES ($1, $2, 'Row', 0) RETURNING id::text`, boardID, bandID).Scan(&rowID); err != nil {
		t.Fatalf("insert row: %v", err)
	}
	var cardID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_cards (storyboard_id, row_id, column_id, sort_order_in_cell, title, image_asset_id)
		VALUES ($1, $2, $3, 0, 'Card', $4) RETURNING id::text
	`, boardID, rowID, columnID, assetID).Scan(&cardID); err != nil {
		t.Fatalf("insert card: %v", err)
	}

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	// A grant holder with zero location memberships anywhere -- exactly the
	// scenario Playwright caught (a throwaway crew test account).
	crew := insertEwriteVizUser(t, pool, "k81_crew")
	if _, err := pool.Exec(ctx, `
		INSERT INTO storyboard_grants (storyboard_id, user_id, granted_role, granted_by)
		VALUES ($1, $2, 'crew', $3)
	`, boardID, crew, owner); err != nil {
		t.Fatalf("insert grant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM storyboard_grants WHERE storyboard_id = $1`, boardID) })

	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, crew, rec); err != nil || !allowed {
		t.Fatalf("crew grant holder with no location membership should read the card image: allowed=%v err=%v", allowed, err)
	}

	// The board owner always reads it too.
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, owner, rec); err != nil || !allowed {
		t.Fatalf("board owner should read the card image: allowed=%v err=%v", allowed, err)
	}

	// A user with neither a grant nor location membership is still denied --
	// the fix must not over-grant access to every storyboard-card image.
	stranger := insertEwriteVizUser(t, pool, "k81_stranger")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, stranger, rec); err != nil || allowed {
		t.Fatalf("a user with no grant and no membership must be denied: allowed=%v err=%v", allowed, err)
	}
}
