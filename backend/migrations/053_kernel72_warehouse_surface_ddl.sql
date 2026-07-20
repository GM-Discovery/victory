-- Kernel 72: DDL extracted verbatim from assets.EnsureKernel49WarehouseStorageSurface.
-- The Go bootstrap now carries only settings/backfill seeds.
ALTER TABLE assets ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';

ALTER TABLE assets ADD COLUMN IF NOT EXISTS shape TEXT NOT NULL DEFAULT 'circle';

ALTER TABLE assets ADD COLUMN IF NOT EXISTS default_grid_width INTEGER NOT NULL DEFAULT 1;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS default_grid_height INTEGER NOT NULL DEFAULT 1;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS retain_original BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

ALTER TABLE assets ADD COLUMN IF NOT EXISTS crop_x DOUBLE PRECISION NOT NULL DEFAULT 0.5;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS crop_y DOUBLE PRECISION NOT NULL DEFAULT 0.5;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS zoom DOUBLE PRECISION NOT NULL DEFAULT 1;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS stored_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS warehouse_storage_settings (
	location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
	hard_limit_bytes BIGINT NOT NULL DEFAULT 8589934592,
	warning_threshold_percent INTEGER NOT NULL DEFAULT 80,
	critical_threshold_percent INTEGER NOT NULL DEFAULT 90,
	max_upload_bytes BIGINT NOT NULL DEFAULT 26214400,
	retain_originals_default BOOLEAN NOT NULL DEFAULT FALSE,
	token_master_max_dimension INTEGER NOT NULL DEFAULT 1024,
	token_stage_max_dimension INTEGER NOT NULL DEFAULT 512,
	token_thumbnail_max_dimension INTEGER NOT NULL DEFAULT 128,
	image_quality INTEGER NOT NULL DEFAULT 85,
	updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

CREATE INDEX IF NOT EXISTS idx_assets_location_asset_type_status_created_at
	ON assets(location_id, asset_type, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_assets_deleted_at
	ON assets(deleted_at);
