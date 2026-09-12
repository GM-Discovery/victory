BEGIN;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS shape TEXT NOT NULL DEFAULT 'circle';

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS default_grid_width INTEGER NOT NULL DEFAULT 1;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS default_grid_height INTEGER NOT NULL DEFAULT 1;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS retain_original BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS crop_x DOUBLE PRECISION NOT NULL DEFAULT 0.5;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS crop_y DOUBLE PRECISION NOT NULL DEFAULT 0.5;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS zoom DOUBLE PRECISION NOT NULL DEFAULT 1;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS stored_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_assets_location_asset_type_status_created_at
  ON assets(location_id, asset_type, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_assets_deleted_at
  ON assets(deleted_at);

CREATE TABLE IF NOT EXISTS warehouse_storage_settings (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  hard_limit_bytes BIGINT NOT NULL DEFAULT 16106127360,
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

INSERT INTO warehouse_storage_settings (
  location_id,
  hard_limit_bytes,
  warning_threshold_percent,
  critical_threshold_percent,
  max_upload_bytes,
  retain_originals_default,
  token_master_max_dimension,
  token_stage_max_dimension,
  token_thumbnail_max_dimension,
  image_quality
)
SELECT
  l.id,
  16106127360,
  80,
  90,
  26214400,
  FALSE,
  1024,
  512,
  128,
  85
FROM locations l
WHERE l.is_default
ON CONFLICT (location_id) DO NOTHING;

UPDATE assets
SET stored_bytes = COALESCE(stored_bytes, 0),
    name = COALESCE(NULLIF(name, ''), COALESCE(NULLIF(original_filename, ''), id::text)),
    shape = COALESCE(NULLIF(shape, ''), 'circle');

COMMIT;
