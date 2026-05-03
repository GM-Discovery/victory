BEGIN;

-- ---------------------------------------------------------------------------
-- Performer profiles: safe public-facing performer surface.
-- No birthdate, SSN, or credit-card-adjacent fields are stored here.
-- Age is represented only as a public performance range string.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS performer_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

  draft_stage_name TEXT NOT NULL DEFAULT '',
  draft_pronouns TEXT NOT NULL DEFAULT '',
  draft_headshot_url TEXT NOT NULL DEFAULT '',
  draft_performance_age_range TEXT NOT NULL DEFAULT '',
  draft_bio TEXT NOT NULL DEFAULT '',
  draft_credits TEXT NOT NULL DEFAULT '',
  draft_skills TEXT NOT NULL DEFAULT '',
  draft_availability TEXT NOT NULL DEFAULT '',
  draft_public_links TEXT NOT NULL DEFAULT '',

  published_stage_name TEXT NOT NULL DEFAULT '',
  published_pronouns TEXT NOT NULL DEFAULT '',
  published_headshot_url TEXT NOT NULL DEFAULT '',
  published_performance_age_range TEXT NOT NULL DEFAULT '',
  published_bio TEXT NOT NULL DEFAULT '',
  published_credits TEXT NOT NULL DEFAULT '',
  published_skills TEXT NOT NULL DEFAULT '',
  published_availability TEXT NOT NULL DEFAULT '',
  published_public_links TEXT NOT NULL DEFAULT '',

  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_performer_profiles_is_published
  ON performer_profiles(is_published);

-- ---------------------------------------------------------------------------
-- Venues: Greenroom + Trailers
-- These are logged-in-only venues surfaced by map visibility.
-- ---------------------------------------------------------------------------
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
  'The Greenroom',
  'greenroom',
  'profile',
  '{
    "surface":"identity",
    "public_profile":true,
    "edit_link":"/venues/trailers/"
  }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'greenroom'
);

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
  'Trailers',
  'trailers',
  'profile',
  '{
    "surface":"identity",
    "drafting":true,
    "public_profile":false
  }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'trailers'
);

COMMIT;
