-- Kernel 78: seed the Writer's Room venue (eWrite authoring home) and make
-- the Library's seed migration-backed.
--
-- Writer's Room is new: map-visible only to Crew+ (producer/director/crew,
-- plus the Operator short-circuit) via a dedicated UNION arm in
-- internal/access/visibility.go with visible_because='ewrite_author_surface'.
--
-- The Library row has been seeded every boot by EnsureKernel16VenueSurface
-- (internal/access/kernel16_venue_bootstrap.go) since Kernel 16, but Kernel
-- 77A established that migrations, not Go bootstrap, are canonical venue
-- truth ("works-live, fails-fresh" is exactly the trap 77A repaired for the
-- Audition Hall). The ON CONFLICT DO NOTHING form leaves any existing row --
-- whichever mechanism created it -- untouched.
--
-- Same lot rationale as migration 083: every canonical venue lives under
-- 'amurray-family'/'main-lot'. kind is not behaviorally load-bearing
-- (venues.kind is never branched on in Go); 'commons' matches other
-- broadly-scoped non-workshop venues. No map/grid row: map pin position and
-- icon are frontend-static in frontend/app.js, per 083's recorded decision.
WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family'
),
lot_row AS (
  SELECT id FROM lots
  WHERE slug = 'main-lot'
    AND location_id = (SELECT id FROM location_row)
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  id,
  'Writer''s Room',
  'writers-room',
  'commons',
  '{
    "surface": "ewrite_authoring"
  }'::jsonb,
  FALSE,
  FALSE
FROM lot_row
ON CONFLICT (lot_id, slug) DO UPDATE
  SET name = EXCLUDED.name,
      kind = EXCLUDED.kind,
      is_public = EXCLUDED.is_public,
      is_workshop = EXCLUDED.is_workshop;

WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family'
),
lot_row AS (
  SELECT id FROM lots
  WHERE slug = 'main-lot'
    AND location_id = (SELECT id FROM location_row)
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  id,
  'Library',
  'library',
  'library',
  '{
    "surface": "public_library",
    "reference_only": true
  }'::jsonb,
  TRUE,
  FALSE
FROM lot_row
ON CONFLICT (lot_id, slug) DO NOTHING;
