BEGIN;

ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS sheet_links JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMIT;
