package assets

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/db"
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

func TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails(t *testing.T) {
	pool := openWarehouseTestPool(t)
	stats, err := loadWarehouseStorageStats(context.Background(), pool, "/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("load warehouse storage stats: %v", err)
	}
	if stats.TotalStoredBytes <= 0 {
		t.Fatalf("expected database-backed stored bytes, got %d", stats.TotalStoredBytes)
	}
	if stats.ActiveAssets <= 0 {
		t.Fatalf("expected active assets to be counted, got %d", stats.ActiveAssets)
	}
}

func openWarehouseTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable")
	if err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	return pool
}
