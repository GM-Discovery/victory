-- Kernel 73: participant_interactions_enabled venue capability flag, exactly
-- mirroring Kernel 72A's pattern (055_kernel72a_stage_capability_flags.sql):
-- a boolean in venues.config, read fail-closed for unknown venues/missing
-- flags, seeded here for existing venue rows AND in the Kernel 16 fresh-
-- install venue seed (access/kernel16_venue_bootstrap.go). This deliberately
-- is NOT folded into stage_elements_enabled -- private Program interactions
-- have different authority (participant-scoped, not role-scoped) and
-- projection (targeted-push, never broadcast) rules than the general
-- element-action surface that flag gates.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{participant_interactions_enabled}', 'true'::jsonb, TRUE)
WHERE slug = 'catharsis'
  AND NOT COALESCE((config ->> 'participant_interactions_enabled')::boolean, FALSE);
