BEGIN;

-- Kernel 71: after a valid Show ticket creates a Player roster row, the
-- Player selects any active Character they own to present as for that Show
-- Run, freely switchable, no Director approval required. This deliberately
-- stays independent of the site-wide active_user_characters and the
-- Session-scoped current_session_personas -- see world/snapshot.go's
-- resolveTheaterContext, the one place this column is read to answer
-- "has this Player chosen a Character."
ALTER TABLE show_run_roster_members
  ADD COLUMN IF NOT EXISTS character_card_id UUID REFERENCES character_cards(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_show_run_roster_members_character_card_id
  ON show_run_roster_members(character_card_id) WHERE character_card_id IS NOT NULL;

COMMIT;
