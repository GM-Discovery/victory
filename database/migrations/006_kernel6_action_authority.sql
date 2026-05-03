BEGIN;

-- Kernel 6 policy seam:
-- actors_can_reveal lives on the venue config for now so it is easy to toggle in dev.
-- The action authority layer reads this flag server-side before reveal/hide writes.
UPDATE venues
SET config = jsonb_set(
  COALESCE(config, '{}'::jsonb),
  '{actors_can_reveal}',
  'false'::jsonb,
  TRUE
)
WHERE slug = 'the-cave';

COMMIT;
