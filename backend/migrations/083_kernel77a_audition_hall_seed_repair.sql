-- Kernel 77A: restore the Audition Hall as canonical migration/bootstrap
-- truth. It was previously added by hand from the terminal rather than
-- through a migration, so Kernel 76's clean database rebuild removed it --
-- confirmed by its absence from the live `venues` table and by
-- `internal/access/visibility.go`'s ResolveVisibleVenues, which already
-- hardcodes `v.slug IN ('audition-hall', 'trailers')` as the
-- "authenticated_surface" visibility class. That code has been silently
-- returning one fewer venue than it was written to return ever since.
--
-- Lot: 'amurray-family' / 'main-lot', the same lot every other canonical
-- venue already lives in (confirmed empirically: all 16 existing venues sit
-- under this one lot; the separate 'victory-theater' neutral-install
-- location, seeded by migration 025, deliberately holds no venues and is
-- left alone here).
--
-- kind='commons': not behaviorally load-bearing (grepped -- venues.kind is
-- never branched on in Go), chosen for consistency with other
-- broadly-visible, non-workshop, non-profile venues (third-place, show-runs).
--
-- No map/grid/layout row is needed: the map pin position and icon for
-- audition-hall are already hardcoded in frontend/app.js (confirmed present
-- and unconditional), and venue_grid_configs is empty for every venue, not
-- just this one -- map placement is a frontend-static concern here, not a
-- database one.
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
  'Audition Hall',
  'audition-hall',
  'commons',
  '{
    "surface": "admission_commons",
    "join_show_by_code": true
  }'::jsonb,
  FALSE,
  FALSE
FROM lot_row
ON CONFLICT (lot_id, slug) DO UPDATE
  SET name = EXCLUDED.name,
      kind = EXCLUDED.kind,
      is_public = EXCLUDED.is_public,
      is_workshop = EXCLUDED.is_workshop;
