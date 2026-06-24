BEGIN;

ALTER TABLE warehouse_storage_settings
  ALTER COLUMN hard_limit_bytes SET DEFAULT 8589934592;

UPDATE warehouse_storage_settings
SET hard_limit_bytes = 8589934592
WHERE location_id = (
  SELECT id
  FROM locations
  WHERE slug = 'amurray-family'
  LIMIT 1
)
AND hard_limit_bytes = 16106127360;

COMMIT;
