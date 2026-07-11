BEGIN;

-- Kernel 65: Third Place venue and Headshots -- an opt-in social commons
-- built on top of the Trailer Face (Kernel 61A) and My People (Kernel 62)
-- primitives. A Headshot never stores its own copy of Face content; it is
-- always projected live from the owner's current Trailer Face at read time
-- (Kernel 65 §3.3, §5.2).
--
-- The venue seed below follows the exact same idempotent pattern used for
-- "trailers"/"greenroom" in 007_kernel9_profiles_greenroom_trailers.sql, so
-- this migration alone (no Go-side Ensure*Surface bootstrap) is enough to
-- make Third Place exist on any freshly migrated database -- see Kernel 64's
-- lesson that Go-only seeding makes a database "works live, fails fresh."
WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
),
lot_row AS (
  SELECT id FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'Third Place',
  'third-place',
  'commons',
  '{
    "surface": "social_commons",
    "public_profile": false
  }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'third-place'
);

-- One row per placement or removal event; the "active" row (removed_at IS
-- NULL) is what the Headshot Commons lists, everything else is history only
-- (Kernel 65 §5.2, §5.5). The partial unique index is the actual "one active
-- Headshot per account" enforcement -- LeaveHeadshot's INSERT ... ON
-- CONFLICT targets it directly, so the rule holds even under concurrent
-- requests, not just in application logic.
CREATE TABLE IF NOT EXISTS third_place_headshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'active',
  placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  removed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT third_place_headshots_status_check CHECK (status IN ('active', 'removed')),
  CONSTRAINT third_place_headshots_removed_at_matches_status CHECK (
    (status = 'active' AND removed_at IS NULL) OR
    (status = 'removed' AND removed_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_third_place_headshots_active_user
  ON third_place_headshots(user_id)
  WHERE removed_at IS NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_third_place_headshots_active_placed_at
  ON third_place_headshots(placed_at DESC)
  WHERE removed_at IS NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_third_place_headshots_user_placed_at
  ON third_place_headshots(user_id, placed_at DESC);

COMMIT;
