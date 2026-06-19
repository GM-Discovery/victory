BEGIN;

ALTER TABLE venue_active_maps
  ADD COLUMN IF NOT EXISTS display_mode TEXT NOT NULL DEFAULT 'theater';

COMMIT;
