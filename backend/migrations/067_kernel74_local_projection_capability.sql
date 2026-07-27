BEGIN;

-- Kernel 74: participant_local_projection_enabled venue capability flag,
-- following the Kernel 72A/73/73A pattern (055/058/064): a boolean in
-- venues.config, read fail-closed for unknown venues and missing flags,
-- seeded here for existing venue rows AND required in the Kernel 16 fresh-
-- install venue seed (access/kernel16_venue_bootstrap.go) -- miss either and
-- fresh installs silently lose the surface.
--
-- A separate flag from participant_interactions_enabled on purpose. That
-- flag gates opening a Program (a read, scoped to one Player's panel).
-- This one gates the genuinely new machinery: substituting which Scene a
-- single viewer's snapshot resolves. Different blast radius, so a Producer
-- can turn the projection off at a venue without also disabling Kessa.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{participant_local_projection_enabled}', 'true'::jsonb, TRUE)
WHERE slug = 'catharsis'
  AND NOT COALESCE((config ->> 'participant_local_projection_enabled')::boolean, FALSE);

COMMIT;
