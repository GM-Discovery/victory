WITH new_location AS (
  INSERT INTO locations (name, slug)
  VALUES ('amurray.family', 'amurray-family')
  ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
  RETURNING id
),
location_row AS (
  SELECT id FROM new_location
  UNION
  SELECT id FROM locations WHERE slug = 'amurray-family'
  LIMIT 1
),
new_lot AS (
  INSERT INTO lots (location_id, name, slug)
  SELECT id, 'main-lot', 'main-lot'
  FROM location_row
  ON CONFLICT (location_id, slug) DO UPDATE SET name = EXCLUDED.name
  RETURNING id, location_id
),
lot_row AS (
  SELECT id, location_id FROM new_lot
  UNION
  SELECT id, location_id FROM lots
  WHERE slug = 'main-lot'
    AND location_id = (SELECT id FROM location_row)
  LIMIT 1
),
new_venue AS (
  INSERT INTO venues (lot_id, name, slug, kind, config)
  SELECT id, 'the-cave', 'the-cave', 'presentation',
         '{
           "stage":{"anchor":"center-front"},
           "audience":{"anchor":"facing-stage"},
           "overlay":{"enabled":true},
           "backdrop":{"kind":"mountain-wall"},
           "terrain_editing":false,
           "vertical_complexity":false
         }'::jsonb
  FROM lot_row
  ON CONFLICT (lot_id, slug) DO UPDATE
    SET name = EXCLUDED.name,
        kind = EXCLUDED.kind,
        config = EXCLUDED.config
  RETURNING id
),
venue_row AS (
  SELECT id FROM new_venue
  UNION
  SELECT id FROM venues
  WHERE slug = 'the-cave'
    AND lot_id = (SELECT id FROM lot_row)
  LIMIT 1
),
new_library AS (
  INSERT INTO libraries (location_id, name)
  SELECT id, 'house-library'
  FROM location_row
  WHERE NOT EXISTS (
    SELECT 1 FROM libraries
    WHERE location_id = (SELECT id FROM location_row)
      AND name = 'house-library'
  )
  RETURNING id
),
library_row AS (
  SELECT id FROM new_library
  UNION
  SELECT id FROM libraries
  WHERE location_id = (SELECT id FROM location_row)
    AND name = 'house-library'
  LIMIT 1
),
new_element AS (
  INSERT INTO elements (library_id, name, slug, element_type, state, data)
  SELECT
    id,
    'first-fire',
    'first-fire',
    'image',
    'library',
    '{
      "src": "/assets/fire.png",
      "label": "First Fire",
      "description": "Persistent central fire element for the-cave",
      "visual": {
        "style": "static",
        "spatial_hint": true
      }
    }'::jsonb
  FROM library_row
  ON CONFLICT (library_id, slug) DO UPDATE
    SET name = EXCLUDED.name,
        element_type = EXCLUDED.element_type,
        state = EXCLUDED.state,
        data = EXCLUDED.data
  RETURNING id
),
element_row AS (
  SELECT id FROM new_element
  UNION
  SELECT id FROM elements
  WHERE library_id = (SELECT id FROM library_row)
    AND slug = 'first-fire'
  LIMIT 1
)
INSERT INTO venue_layout_elements (
  venue_id,
  element_id,
  surface,
  position,
  visibility,
  is_default
)
SELECT
  (SELECT id FROM venue_row),
  (SELECT id FROM element_row),
  'stage',
  '{
    "anchor":"center-front",
    "x":0,
    "y":0,
    "z":0
  }'::jsonb,
  '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}'::jsonb,
  TRUE
WHERE NOT EXISTS (
  SELECT 1
  FROM venue_layout_elements
  WHERE venue_id = (SELECT id FROM venue_row)
    AND element_id = (SELECT id FROM element_row)
    AND surface = 'stage'
);
