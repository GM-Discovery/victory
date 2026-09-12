BEGIN;

WITH location_row AS (
  SELECT id
  FROM locations
  WHERE is_default
  LIMIT 1
),
lot_row AS (
  SELECT id
  FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'Producer''s Office',
  'producers-office',
  'office',
  '{ "surface": "permissions", "office": "producer", "requests": true, "permissions": true }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1
  FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'producers-office'
);

WITH location_row AS (
  SELECT id
  FROM locations
  WHERE is_default
  LIMIT 1
),
lot_row AS (
  SELECT id
  FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'The Director''s Chair',
  'directors-chair',
  'plaza',
  '{ "surface": "permissions", "office": "director", "requests": true, "permissions": true }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1
  FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'directors-chair'
);

COMMIT;
