-- Kernel 96: Grant's Cabin keeps its name (Grant wants it to), but the
-- contact info baked into its config shouldn't be his personal address.
-- EnsureKernel16VenueSurface's own INSERT is "WHERE NOT EXISTS", so
-- updating the Go source only affects installs that haven't seeded this
-- venue yet -- this one-time UPDATE fixes already-existing rows (the
-- original amurray-family install, and any Windows installs already set up).
UPDATE venues
SET config = jsonb_set(config, '{contact}', '"GM-Discovery on GitHub, gm_discovery on Discord"')
WHERE slug = 'grants-cabin'
  AND config->>'contact' = 'grant@amurray.family';
