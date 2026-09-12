BEGIN;

-- Kernel 93 follow-up (Grant, 2026-08-21): the frontend (frontend/app.js)
-- has always carried icon/position/routing logic for a "victory-theater"
-- venue tile -- a special ~180%-scale "hub" pin (see the old
-- Construction/Kernels/reportbackGUImap exploration notes) -- but no
-- `venues` row with that slug was ever committed via a migration. The only
-- thing actually named "victory-theater" in the schema is the unrelated,
-- deliberately venue-less neutral-install *Location* seeded by migration
-- 025 (see 083's comment) -- a different table, a different concept, left
-- alone here. This creates the venue itself: a "coming soon" advertisement
-- page (frontend/venues/victory-theater/), reachable once unlocked (see
-- access.ResolveVisibleVenues's new UNION branch, added alongside this
-- migration).
--
-- Same lot every other canonical venue lives in (main-lot / the install's default location,
-- per migration 083's own confirmation). kind='commons', matching
-- audition-hall/third-place/show-runs -- venues.kind is not branched on in
-- Go.
WITH location_row AS (
  SELECT id FROM locations WHERE is_default
),
lot_row AS (
  SELECT id FROM lots
  WHERE slug = 'main-lot'
    AND location_id = (SELECT id FROM location_row)
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  id,
  'Victory Theater',
  'victory-theater',
  'commons',
  '{}'::jsonb,
  FALSE,
  FALSE
FROM lot_row
ON CONFLICT (lot_id, slug) DO UPDATE
  SET name = EXCLUDED.name,
      kind = EXCLUDED.kind;

COMMIT;
