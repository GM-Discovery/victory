-- Kernel 72A: venue-config capability flags replace hardcoded venue-slug
-- allowlists (actions.isSingleVenueLegacySlug and session_control.go's
-- normalizeSessionControlVenueSlug). The hardcoded lists caused the
-- operator-reported bug where catharsis shipped the full stage toolset but
-- every element action there was denied "unknown_target" — a bug class every
-- future game venue would have repeated. Follows migration 044's
-- scene_rehearsal_enabled jsonb_set precedent.
--
-- Fresh-install note: first-theater/catharsis/middle-school-stage venue rows
-- are Go-seeded (access.EnsureKernel16VenueSurface) AFTER migrations run, so
-- the seed configs also carry these flags; this migration covers databases
-- whose venue rows already exist.

-- stage_elements_enabled: the element-action surface (tokens, index cards,
-- reveal, remove, duplicate, lock, nameplate) is available on this venue's
-- stage.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{stage_elements_enabled}', 'true'::jsonb, TRUE)
WHERE slug IN ('the-cave', 'first-theater', 'catharsis')
  AND NOT COALESCE((config ->> 'stage_elements_enabled')::boolean, FALSE);

-- session_control_enabled: /session status|start|end commands may target
-- this venue (the former session_control.go allowlist, unchanged membership).
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{session_control_enabled}', 'true'::jsonb, TRUE)
WHERE slug IN ('the-cave', 'first-theater', 'catharsis', 'middle-school-stage')
  AND NOT COALESCE((config ->> 'session_control_enabled')::boolean, FALSE);
