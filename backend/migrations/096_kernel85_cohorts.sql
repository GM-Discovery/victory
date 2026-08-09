BEGIN;

-- Kernel 85: Show Cohorts -- Director-run sustained-play grouping.
--
-- A cohort is Show-scoped (not a global/account-level grouping -- see My
-- People, Third Place, and the location_memberships/show_run_roster_members
-- family, all deliberately untouched here). show_cohort_serial_counters
-- holds one monotonic per-Show counter, row-locked (SELECT ... FOR UPDATE)
-- inside the same transaction that inserts the new show_cohorts row, so
-- concurrent Director requests can never race to the same serial number and
-- a deleted Cohort 2 can never be re-issued to a later cohort (kernel-85
-- S1.2, S3.2, S3.3).
--
-- show_cohort_assignments is deliberately "at most one row per (show_id,
-- user_id)" via its primary key rather than a membership-history table --
-- moving a participant between cohorts is an UPDATE, returning them to
-- Ungrouped is a DELETE, and Ungrouped itself is never a stored row (kernel
-- 85 S3.1: "Ungrouped should preferably be computed"). This is what makes
-- "at most one active cohort per participant" true by construction rather
-- than by an application-level check.
--
-- current_show_scene_placement_id mirrors shows.current_show_scene_
-- placement_id's existing single-pointer shape (Kernel 70 SS4.1), just
-- living on the cohort instead of the Show -- the same show_scene_placements
-- row a cohort points at is already validated (belongs to this Show, not
-- archived/retired) by backend/internal/cohorts using the same pattern
-- backend/internal/shows/stage.go already established.

CREATE TABLE IF NOT EXISTS show_cohort_serial_counters (
  show_id UUID PRIMARY KEY REFERENCES shows(id) ON DELETE CASCADE,
  next_serial INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS show_cohorts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  serial_number INTEGER NOT NULL,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  current_show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  UNIQUE (show_id, serial_number),
  UNIQUE (show_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_show_cohorts_show_id ON show_cohorts(show_id) WHERE archived_at IS NULL;

CREATE TABLE IF NOT EXISTS show_cohort_assignments (
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  cohort_id UUID NOT NULL REFERENCES show_cohorts(id) ON DELETE CASCADE,
  assigned_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (show_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_show_cohort_assignments_cohort_id ON show_cohort_assignments(cohort_id);

COMMIT;
