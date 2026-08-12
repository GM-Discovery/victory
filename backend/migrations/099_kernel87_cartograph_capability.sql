-- Kernel 87: cartograph_enabled and ic_chat_enabled venue capability flags,
-- mirroring Kernel 72A/73's pattern (055/058) exactly -- booleans in
-- venues.config, read fail-closed for unknown venues/missing flags, seeded
-- here for existing venue rows AND in the Kernel 16 fresh-install venue
-- seed (access/kernel16_venue_bootstrap.go). Catharsis-only for the same
-- reason Kernel 73's Equip Mode is Catharsis-only (current-state.md):
-- First Theater integration is explicitly deferred, not a scope gap.
UPDATE venues
SET config = jsonb_set(jsonb_set(COALESCE(config, '{}'::jsonb),
  '{cartograph_enabled}', 'true'::jsonb, TRUE),
  '{ic_chat_enabled}', 'true'::jsonb, TRUE)
WHERE slug = 'catharsis'
  AND (NOT COALESCE((config ->> 'cartograph_enabled')::boolean, FALSE)
    OR NOT COALESCE((config ->> 'ic_chat_enabled')::boolean, FALSE));
