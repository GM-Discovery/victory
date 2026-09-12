-- Kernel 73: seed the reusable "Courtyard" Scene that Catharsis's Equip Mode
-- golden path assumes exists (kernel-73 spec S0/S13.3). Scenes are
-- location-scoped and reusable across any Show (Kernel 70's
-- 043_kernel70_scene_location_scoping.sql moved them off production_id onto
-- location_id + a nullable source_production_id for provenance) -- this
-- mirrors the one existing seeded Scene, "character-making-opening"
-- (042_kernel69_scenes.sql), which is itself only Go/manually created, not
-- migration-seeded; this is the first migration-seeded Scene.
--
-- This migration deliberately does NOT create a show_scene_placements row.
-- Live production has no non-test Show currently linked to the Catharsis
-- venue's session (session.show_id is NULL) -- Shows are created through
-- Stage Management, and placing a reusable Scene into one is the existing
-- Kernel 69 "Add Scene to Show" flow. Kessa's participant_interactions
-- attachment (061) is likewise done through the real authoring UI (Kernel 73
-- Phase E), never hardcoded against a specific show_id here.
INSERT INTO scenes (
  location_id,
  source_production_id,
  slug,
  title,
  short_title,
  default_venue_id,
  audience_title,
  audience_summary,
  player_brief,
  status,
  created_by_user_id
)
SELECT
  l.id,
  NULL,
  'courtyard',
  'The Courtyard',
  'Courtyard',
  v.id,
  'The Courtyard',
  'A shared gathering space at the heart of the story.',
  'The Courtyard is where the company gathers before a scene begins.',
  'ready',
  NULL
FROM locations l
JOIN venues v ON v.slug = 'catharsis' AND v.lot_id IN (
  SELECT id FROM lots WHERE location_id = l.id
)
WHERE l.is_default
  AND NOT EXISTS (
    SELECT 1 FROM scenes s WHERE s.location_id = l.id AND s.slug = 'courtyard'
  );
