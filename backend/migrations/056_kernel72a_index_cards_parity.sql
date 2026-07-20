-- Kernel 72A (second pass): index_cards_enabled parity for the stage venues.
-- Migration 010 only ever set this flag on the-cave. First Theater and
-- Catharsis ship the same stage toolset (including index cards), so they get
-- the same index-card policy. The placement resolver itself no longer reads
-- this flag (it checks stage_elements_enabled, the correct capability);
-- index_cards_enabled remains the index-card-specific authority policy.
-- Fresh installs get the flag from the Kernel 16 venue seed configs.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{index_cards_enabled}', 'true'::jsonb, TRUE)
WHERE slug IN ('first-theater', 'catharsis')
  AND NOT COALESCE((config ->> 'index_cards_enabled')::boolean, FALSE);
