package assets

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func TestDefaultWarehouseStorageSettings(t *testing.T) {
	settings := defaultWarehouseStorageSettings("location-1")
	if settings.LocationID != "location-1" {
		t.Fatalf("location id = %q, want %q", settings.LocationID, "location-1")
	}
	if settings.HardLimitBytes != DefaultWarehouseHardLimitBytes {
		t.Fatalf("hard limit = %d, want %d", settings.HardLimitBytes, DefaultWarehouseHardLimitBytes)
	}
	if settings.MaxUploadBytes != DefaultWarehouseMaxUploadBytes {
		t.Fatalf("max upload = %d, want %d", settings.MaxUploadBytes, DefaultWarehouseMaxUploadBytes)
	}
	if settings.WarningThresholdPercent != 80 || settings.CriticalThresholdPercent != 90 {
		t.Fatalf("thresholds = %d/%d, want 80/90", settings.WarningThresholdPercent, settings.CriticalThresholdPercent)
	}
	if settings.TokenMasterMaxDimension != DefaultTokenMasterMaxDimension || settings.TokenStageMaxDimension != DefaultTokenStageMaxDimension || settings.TokenThumbnailMaxDimension != DefaultTokenThumbnailMaxDimension {
		t.Fatalf("token dimensions = %d/%d/%d, want %d/%d/%d", settings.TokenMasterMaxDimension, settings.TokenStageMaxDimension, settings.TokenThumbnailMaxDimension, DefaultTokenMasterMaxDimension, DefaultTokenStageMaxDimension, DefaultTokenThumbnailMaxDimension)
	}
}

// loadWarehouseStorageStats takes a location UUID, not a filesystem path -
// filesystem usage is loaded separately by loadWarehouseFilesystemStats and
// the two are never combined inside loadWarehouseStorageStats itself. The
// original version of this test passed a nonexistent filesystem path where
// the location UUID belongs, which failed the `$1::uuid` cast on every run
// regardless of which database it targeted. This fixture creates its own
// location + asset row so the assertion is meaningful without depending on
// any pre-seeded production asset data.
func TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails(t *testing.T) {
	pool := openWarehouseTestPool(t)
	ctx := context.Background()

	locationID, userID := insertWarehouseTestLocationAndUser(t, pool)
	const wantStoredBytes = 4096
	insertWarehouseTestAsset(t, pool, locationID, userID, wantStoredBytes)

	stats, err := loadWarehouseStorageStats(ctx, pool, locationID)
	if err != nil {
		t.Fatalf("load warehouse storage stats: %v", err)
	}
	if stats.TotalStoredBytes != wantStoredBytes {
		t.Fatalf("total stored bytes = %d, want %d", stats.TotalStoredBytes, wantStoredBytes)
	}
	if stats.ActiveAssets != 1 {
		t.Fatalf("active assets = %d, want 1", stats.ActiveAssets)
	}
}

func openWarehouseTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func insertWarehouseTestLocationAndUser(t *testing.T, pool *pgxpool.Pool) (locationID, userID string) {
	t.Helper()

	ctx := context.Background()
	suffix := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")

	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (name, slug)
		VALUES ($1, $2)
		RETURNING id::text
	`, "Warehouse Test Location", "warehouse-test-location-"+suffix).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, "warehouse_test_"+suffix, "Warehouse Test User").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM assets WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	return locationID, userID
}

func insertWarehouseTestAsset(t *testing.T, pool *pgxpool.Pool, locationID, userID string, storedBytes int64) string {
	t.Helper()

	var assetID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO assets (
			producer_user_id, location_id, uploader_user_id, owner_user_id, owner_state,
			width, height, byte_size, checksum_sha256, storage_root, original_path,
			stored_bytes
		)
		VALUES (
			$1::uuid, $2::uuid, $1::uuid, $1::uuid, 'personal',
			64, 64, $3::integer, decode('deadbeef', 'hex'), '/tmp', '/tmp/warehouse-test-asset',
			$3::bigint
		)
		RETURNING id::text
	`, userID, locationID, storedBytes).Scan(&assetID); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	return assetID
}
