package assets

// Kernel 79: proves the image-visibility fix. Before this kernel,
// userCanReadAsset gated purely on location-membership/ownership -- an
// eWrite-embedded image never consulted the referencing publication's own
// visibility at all. These tests prove userCanReadAssetConsideringEwrite
// closes both directions of that gap: a public/published eWriting's image
// is readable by any authenticated non-member, while a draft or
// production-scoped eWriting's image is denied to a user who merely holds
// unrelated membership at the asset's own (different) location.
//
// Each test authors its publication through the real ewrite package API
// (CreateCollection/CreatePublication/SavePublicationSource/
// PublishPublication) with an embedded image URL, so
// ewrite_publication_assets is populated by the actual reconcile wiring in
// revisions.go -- not a hand-inserted row -- proving the whole path, not
// just the read-side check in isolation. Requires TEST_DATABASE_URL
// (internal/dbtest safety gate).

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/ewrite"
)

func ewriteVizSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertEwriteVizLocation(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + ewriteVizSuffix(t)
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (name, slug) VALUES ($1, $2) RETURNING id::text
	`, name, slug).Scan(&id); err != nil {
		t.Fatalf("insert location %q: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, id)
	})
	return id
}

func insertEwriteVizUser(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	var id string
	handle := prefix + "_" + ewriteVizSuffix(t)
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, prefix).Scan(&id); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func grantEwriteVizRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant role %q: %v", role, err)
	}
}

func insertEwriteVizAsset(t *testing.T, pool *pgxpool.Pool, locationID, ownerUserID string) string {
	t.Helper()
	var assetID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO assets (
			producer_user_id, location_id, uploader_user_id, owner_user_id, owner_state,
			width, height, byte_size, checksum_sha256, storage_root, original_path
		)
		VALUES ($1::uuid, $2::uuid, $1::uuid, $1::uuid, 'personal', 64, 64, 100, decode('ab', 'hex'), '/tmp', '/tmp/ewrite-viz-test-asset')
		RETURNING id::text
	`, ownerUserID, locationID).Scan(&assetID); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ewrite_publication_assets WHERE asset_id = $1`, assetID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM assets WHERE id = $1`, assetID)
	})
	return assetID
}

// createEwriteVizPublication authors a publication embedding assetID as an
// image through the real save path, so ewrite_publication_assets is
// populated by the actual reconcile wiring.
func createEwriteVizPublication(t *testing.T, pool *pgxpool.Pool, authorUserID, locationID, assetID, visibility string, publish bool) *ewrite.Publication {
	t.Helper()
	ctx := context.Background()
	coll, err := ewrite.CreateCollection(ctx, pool, authorUserID, locationID, "", "ruleset", "Viz Test Ruleset "+ewriteVizSuffix(t), "", visibility)
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	pub, err := ewrite.CreatePublication(ctx, pool, authorUserID, coll.ID, "Viz Test Pub "+ewriteVizSuffix(t), "", visibility)
	if err != nil {
		t.Fatalf("create publication: %v", err)
	}
	source := "# Test\n\n![img](/api/assets/" + assetID + "/content)\n"
	if _, conflict, err := ewrite.SavePublicationSource(ctx, pool, authorUserID, pub.ID, source, pub.CurrentRevisionID); err != nil {
		t.Fatalf("save: %v (conflict %+v)", err, conflict)
	}
	if publish {
		if _, err := ewrite.PublishPublication(ctx, pool, authorUserID, pub.ID); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
	loaded, err := ewrite.LoadPublication(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ewrite_publications WHERE id = $1`, pub.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM ewrite_collections WHERE id = $1`, coll.ID)
	})
	return loaded
}

func TestUserCanReadAssetConsideringEwrite_OwnerAlwaysAllowed(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	locationID := insertEwriteVizLocation(t, pool, "Owner Test Location")
	owner := insertEwriteVizUser(t, pool, "owner")
	assetID := insertEwriteVizAsset(t, pool, locationID, owner)

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, owner, rec)
	if err != nil || !allowed {
		t.Fatalf("owner should always be allowed: allowed=%v err=%v", allowed, err)
	}
}

func TestUserCanReadAssetConsideringEwrite_NonEwriteBoundAssetUnchanged(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	locationID := insertEwriteVizLocation(t, pool, "Legacy Test Location")
	owner := insertEwriteVizUser(t, pool, "owner")
	assetID := insertEwriteVizAsset(t, pool, locationID, owner)

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	outsider := insertEwriteVizUser(t, pool, "outsider")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, outsider, rec); err != nil || allowed {
		t.Fatalf("a user with no membership/ownership should be denied a non-eWrite-bound asset: allowed=%v err=%v", allowed, err)
	}

	grantEwriteVizRole(t, pool, locationID, outsider, "crew")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, outsider, rec); err != nil || !allowed {
		t.Fatalf("crew at the asset's own location should read a non-eWrite-bound asset (legacy behavior): allowed=%v err=%v", allowed, err)
	}
}

func TestUserCanReadAssetConsideringEwrite_PublicPublishedAllowsAnyAuthenticatedReader(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	assetLocationID := insertEwriteVizLocation(t, pool, "Asset Home Location")
	pubLocationID := insertEwriteVizLocation(t, pool, "Publication Home Location")

	uploader := insertEwriteVizUser(t, pool, "uploader")
	pubAuthor := insertEwriteVizUser(t, pool, "pub_author")
	grantEwriteVizRole(t, pool, pubLocationID, pubAuthor, "producer")

	assetID := insertEwriteVizAsset(t, pool, assetLocationID, uploader)
	createEwriteVizPublication(t, pool, pubAuthor, pubLocationID, assetID, "public", true)

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	reader := insertEwriteVizUser(t, pool, "public_reader")
	// The reader holds no membership anywhere -- this is exactly the
	// Kernel 78 false-negative bug: a legitimate Library reader denied
	// because they aren't a member of the asset's own location.
	allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, reader, rec)
	if err != nil || !allowed {
		t.Fatalf("authenticated non-member should read a public published eWriting's image: allowed=%v err=%v", allowed, err)
	}
}

func TestUserCanReadAssetConsideringEwrite_DraftDeniesNonEditorAllowsEditor(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	assetLocationID := insertEwriteVizLocation(t, pool, "Asset Home Location")
	pubLocationID := insertEwriteVizLocation(t, pool, "Publication Home Location")

	uploader := insertEwriteVizUser(t, pool, "uploader")
	pubAuthor := insertEwriteVizUser(t, pool, "pub_author")
	grantEwriteVizRole(t, pool, pubLocationID, pubAuthor, "producer")

	assetID := insertEwriteVizAsset(t, pool, assetLocationID, uploader)
	// Not published: stays 'draft'.
	createEwriteVizPublication(t, pool, pubAuthor, pubLocationID, assetID, "production", false)

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	// Unrelated membership at the asset's own location, but no authoring
	// role or grant at the publication's location -- must be denied.
	outsider := insertEwriteVizUser(t, pool, "outsider")
	grantEwriteVizRole(t, pool, assetLocationID, outsider, "crew")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, outsider, rec); err != nil || allowed {
		t.Fatalf("a non-editor must be denied a draft eWriting's image: allowed=%v err=%v", allowed, err)
	}

	// The publication's own author has edit authority -> allowed.
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, pubAuthor, rec); err != nil || !allowed {
		t.Fatalf("the publication's editor should read its draft image: allowed=%v err=%v", allowed, err)
	}
}

func TestUserCanReadAssetConsideringEwrite_ProductionVisibilityScopesToPublicationLocation(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	assetLocationID := insertEwriteVizLocation(t, pool, "Asset Home Location")
	pubLocationID := insertEwriteVizLocation(t, pool, "Publication Home Location")

	uploader := insertEwriteVizUser(t, pool, "uploader")
	pubAuthor := insertEwriteVizUser(t, pool, "pub_author")
	grantEwriteVizRole(t, pool, pubLocationID, pubAuthor, "producer")

	assetID := insertEwriteVizAsset(t, pool, assetLocationID, uploader)
	createEwriteVizPublication(t, pool, pubAuthor, pubLocationID, assetID, "production", true)

	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		t.Fatalf("load warehouse record: %v", err)
	}

	// This is the core "unrelated asset access" bypass Kernel 79 closes:
	// membership at the asset's own location must NOT grant access to a
	// production-scoped publication the user has no relationship to.
	unrelated := insertEwriteVizUser(t, pool, "unrelated_member")
	grantEwriteVizRole(t, pool, assetLocationID, unrelated, "crew")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, unrelated, rec); err != nil || allowed {
		t.Fatalf("membership at the asset's own (unrelated) location must not grant access: allowed=%v err=%v", allowed, err)
	}

	// A member -- any role, including audience -- at the publication's own
	// location is allowed (CanReadPublication's production-visibility rule).
	member := insertEwriteVizUser(t, pool, "pub_location_member")
	grantEwriteVizRole(t, pool, pubLocationID, member, "audience")
	if allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, member, rec); err != nil || !allowed {
		t.Fatalf("a member of the publication's own location should read its production-visibility image: allowed=%v err=%v", allowed, err)
	}
}
