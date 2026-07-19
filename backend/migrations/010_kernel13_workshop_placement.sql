BEGIN;

-- Enable index card placement for the workshop-enabled venue surface.
UPDATE venues
SET config = jsonb_set(
  COALESCE(config, '{}'::jsonb),
  '{index_cards_enabled}',
  'true'::jsonb,
  TRUE
)
WHERE slug IN ('the-cave');

COMMIT;
