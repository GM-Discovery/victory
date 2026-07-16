BEGIN;

-- Kernel 71: a short, human-typeable code per Show, generated on creation
-- and Director/Producer/Operator-editable, used by /showtime and by
-- Audition Hall's "Join a Show" lookup. Deliberately a new column, not a
-- repurposing of the existing `slug` (which is only unique per Show Run and
-- serves page-routing, a different job). Uniqueness is enforced at the
-- application layer (backend/internal/shows/short_code.go) via a
-- transactional check scoped through show_runs.location_id, since a plain
-- column-level unique index can't reach across that join.
ALTER TABLE shows
  ADD COLUMN IF NOT EXISTS short_code TEXT;

CREATE INDEX IF NOT EXISTS idx_shows_short_code_lower ON shows(lower(short_code));

COMMIT;
