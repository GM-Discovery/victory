BEGIN;

-- Kernel 66: Show Run, Audience Program, and Roster MVP -- a bounded
-- container (Production -> Show Format -> Show Run -> Roster/Audience
-- Program) for a cohort/campaign/table series of a Production. This is the
-- code implementation of the dictionary's pre-existing "Production Run"
-- concept ("Show Run" is its product-facing name -- see Construction/
-- Dictionary.txt).
--
-- Show Run is deliberately NOT "Showing" (Kernel 22, backend/internal/
-- showings): Showing is the live, 1:1 runtime wrapper around an already-
-- active session. Show Run is the scheduling/rostering container a Showing
-- may eventually happen inside of.
--
-- Roster rows never carry their own copy of Trailer Face content -- roster
-- cards are always projected live from the member's current Trailer Face at
-- read time (mirrors Kernel 65's third_place_headshots discipline exactly).
--
-- Seeded with plain idempotent SQL, not a Go-side Ensure*Surface bootstrap,
-- following the exact precedent set by 038_kernel65_third_place.sql -- this
-- venue has no dependency on already-live session/showing state, so a
-- migration-only seed is sufficient for both live and fresh-install
-- databases (Kernel 64's "raw SQL alone can under-seed" lesson applies to
-- surfaces derived from *runtime* state, not static venue tiles like this
-- one).
WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
),
lot_row AS (
  SELECT id FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'Show Runs',
  'show-runs',
  'commons',
  '{
    "surface": "show_run_roster"
  }'::jsonb,
  FALSE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row)
    AND slug = 'show-runs'
);

CREATE TABLE IF NOT EXISTS show_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  production_id UUID NOT NULL REFERENCES productions(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT,
  show_format TEXT NOT NULL DEFAULT 'one-shot',
  custom_show_format TEXT,
  cohort_name TEXT,
  status TEXT NOT NULL DEFAULT 'planning',
  audience_self_join_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  CONSTRAINT show_runs_show_format_check CHECK (show_format IN (
    'one-shot', 'campaign', 'mini-series', 'seasonal-release', 'random-drop',
    'workshop', 'character-making', 'playtest', 'custom'
  )),
  CONSTRAINT show_runs_custom_format_requires_label CHECK (
    show_format <> 'custom' OR custom_show_format IS NOT NULL
  ),
  CONSTRAINT show_runs_status_check CHECK (status IN (
    'planning', 'active', 'paused', 'completed', 'archived'
  )),
  CONSTRAINT show_runs_archived_matches_status CHECK (
    (status = 'archived' AND archived_at IS NOT NULL) OR
    (status <> 'archived' AND archived_at IS NULL)
  ),
  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_show_runs_location_id ON show_runs(location_id);
CREATE INDEX IF NOT EXISTS idx_show_runs_production_id ON show_runs(production_id);
CREATE INDEX IF NOT EXISTS idx_show_runs_status ON show_runs(status);

-- One *active* roster row per (show_run, user) -- role changes update this
-- row via UpdateRosterMemberRole, they do not create a second row. The
-- partial unique index is the real enforcement (mirrors Kernel 65's
-- uq_third_place_headshots_active_user), not just an application check.
CREATE TABLE IF NOT EXISTS show_run_roster_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_run_id UUID NOT NULL REFERENCES show_runs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  custom_role_label TEXT,
  program_visible BOOLEAN NOT NULL DEFAULT TRUE,
  added_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  removed_at TIMESTAMPTZ,
  CONSTRAINT show_run_roster_members_role_check CHECK (role IN (
    'producer', 'director', 'player', 'crew', 'audience', 'guest', 'observer', 'custom'
  )),
  CONSTRAINT show_run_roster_members_custom_requires_label CHECK (
    role <> 'custom' OR custom_role_label IS NOT NULL
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_show_run_roster_members_active_user
  ON show_run_roster_members(show_run_id, user_id)
  WHERE removed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_show_run_roster_members_show_run_id
  ON show_run_roster_members(show_run_id) WHERE removed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_show_run_roster_members_user_id
  ON show_run_roster_members(user_id) WHERE removed_at IS NULL;

-- Run-scoped only, not a site-wide moderation system (Kernel 66 §"There is
-- always some form of ban/block" -- deliberately minimal, no reason
-- visibility to anyone but the Producer/Director/Operator who set it).
CREATE TABLE IF NOT EXISTS show_run_audience_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_run_id UUID NOT NULL REFERENCES show_runs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lifted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_show_run_audience_blocks_active_user
  ON show_run_audience_blocks(show_run_id, user_id)
  WHERE lifted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_show_run_audience_blocks_show_run_id
  ON show_run_audience_blocks(show_run_id);

COMMIT;
