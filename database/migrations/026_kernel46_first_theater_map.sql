BEGIN;

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS asset_type TEXT NOT NULL DEFAULT 'generic';

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}'::text[];

CREATE INDEX IF NOT EXISTS idx_assets_location_asset_type_created_at
  ON assets(location_id, asset_type, created_at DESC);

CREATE TABLE IF NOT EXISTS venue_active_maps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
  asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
  fit TEXT NOT NULL DEFAULT 'cover',
  crop_x DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  crop_y DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  scale DOUBLE PRECISION NOT NULL DEFAULT 1,
  safe_margin INTEGER NOT NULL DEFAULT 24,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (venue_id)
);

CREATE INDEX IF NOT EXISTS idx_venue_active_maps_venue_id
  ON venue_active_maps(venue_id);

CREATE INDEX IF NOT EXISTS idx_venue_active_maps_asset_id
  ON venue_active_maps(asset_id);

COMMIT;
