BEGIN;

-- Kernel 75 S1.9 / S6.4: Director-authored Character moments.
--
-- S1.9 is explicit that this must REUSE the existing /journal system rather
-- than becoming a second, unrelated Director-note store. character_journals
-- already carries the right shape -- a Character, an author, a body, a
-- visibility, and Session context -- and needs only the smallest compatible
-- extension:
--
--   source     tells a reader whether a row is the Player's own reflection
--              or a Director's note about their Character. Without it the
--              two would be indistinguishable in the same table, which is
--              exactly the conflation S1.8 forbids for history.
--
--   show_id /  the Show and Scene context S6.4 requires a Director-authored
--   placement  moment to preserve. The table had venue_id and session_id but
--              never learned about Shows.
--
-- Note what is NOT added: nothing that lets a Director edit a Player's own
-- journal entry. The author-scoped WHERE clauses in UpdateCharacterJournal
-- and ArchiveCharacterJournal are unchanged, so a Director may add their own
-- note and may not touch anyone else's words (S6.4).
ALTER TABLE character_journals
  ADD COLUMN IF NOT EXISTS show_id UUID REFERENCES shows(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'player';

ALTER TABLE character_journals
  DROP CONSTRAINT IF EXISTS character_journals_source_check;
ALTER TABLE character_journals
  ADD CONSTRAINT character_journals_source_check
  CHECK (source IN ('player', 'director_journal'));

CREATE INDEX IF NOT EXISTS idx_character_journals_show
  ON character_journals(show_id, created_at DESC)
  WHERE show_id IS NOT NULL;

COMMIT;
