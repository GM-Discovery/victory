BEGIN;

-- Kernel 75 S8: the Aftercare capability flag, following the pattern
-- established by migrations 055, 058, 064 and 067.
--
-- Fail-closed: a venue without this flag offers no Aftercare at all, rather
-- than offering a form whose submissions go nowhere.
--
-- PAIRED CHANGE, and the single most-repeated trap in this repo's notes:
-- this UPDATE only fixes EXISTING installations. Fresh installs get their
-- venue rows from access.EnsureVenueSurface, so "aftercare_enabled": true
-- must ALSO be present in the catharsis config string in
-- backend/internal/access/kernel16_venue_bootstrap.go. Changing one without
-- the other silently loses the capability on new installs, where it is
-- hardest to notice.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{aftercare_enabled}', 'true'::jsonb, TRUE)
WHERE slug = 'catharsis'
  AND NOT COALESCE((config ->> 'aftercare_enabled')::boolean, FALSE);

COMMIT;
