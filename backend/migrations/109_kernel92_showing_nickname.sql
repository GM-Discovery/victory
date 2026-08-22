BEGIN;

-- Kernel 92: the Director-facing "Showing" scheduler reuses the existing
-- `shows` table (a Show Run can already own many Show rows -- Kernel 67 --
-- which is exactly the "many scheduled performances" shape Kernel 92
-- needs) rather than introducing a second table/code system (kernel doc
-- SS11). The one real gap was a human-friendly, non-unique label distinct
-- from Title -- this column closes it.
ALTER TABLE shows
  ADD COLUMN IF NOT EXISTS nickname TEXT;

-- Backfill pre-Kernel-92 rows from Title so the column is never blank for
-- existing data, even though "required" itself is enforced at the
-- application layer (shows.CreateShowing), matching how short_code
-- uniqueness is already app-layer-only rather than a DB constraint.
UPDATE shows
SET nickname = LEFT(COALESCE(NULLIF(title, ''), 'Showing'), 50)
WHERE nickname IS NULL;

-- ALTER TABLE ... ADD CONSTRAINT has no IF NOT EXISTS, so the idempotency
-- requirement (migrations run as one Exec, re-runnable on a database where
-- they already applied) forces the pg_constraint guard -- see migration
-- 084's ewrite_publications_current_revision_fk for the same pattern.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'shows_nickname_length_check'
  ) THEN
    ALTER TABLE shows
      ADD CONSTRAINT shows_nickname_length_check CHECK (char_length(nickname) <= 50);
  END IF;
END $$;

COMMENT ON COLUMN shows.nickname IS
  'Kernel 92 Showing nickname -- Director-facing label, <=50 chars, required at creation via shows.CreateShowing but not DB NOT NULL (backfilled from Title for pre-Kernel-92 rows). Not required to be globally unique -- the short_code/timestamp pair already provides uniqueness where needed.';

COMMIT;
