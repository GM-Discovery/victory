WITH new_location AS (
  INSERT INTO locations (name, slug)
  VALUES ('Victory Theater', 'victory-theater')
  ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
  RETURNING id
),
location_row AS (
  SELECT id FROM new_location
  UNION
  SELECT id FROM locations WHERE slug = 'victory-theater'
  LIMIT 1
),
new_lot AS (
  INSERT INTO lots (location_id, name, slug)
  SELECT id, 'main-lot', 'main-lot'
  FROM location_row
  ON CONFLICT (location_id, slug) DO UPDATE SET name = EXCLUDED.name
  RETURNING id
)
SELECT 1;
