BEGIN;

-- Kernel 67: Show Instance Model and Show Run Bridge -- fills the missing
-- container between a Show Run (Kernel 66) and the live Session/Showing
-- system (Kernel 22): Production -> Show Run -> Show -> Session(s).
--
-- A Show is NOT a Session (technical live runtime window, `sessions` table)
-- and NOT a Showing (Kernel 22's live 1:1 audience-visibility wrapper,
-- backend/internal/showings) -- both stay completely untouched by this
-- migration. A live-code audit found showings.session_id is NOT NULL
-- UNIQUE and read/written from 20+ call sites (internal/actions/*.go,
-- network/session_control.go, identity/discord_mic.go, identity/join.go);
-- loosening it to support a pre-session, multi-session Show was rejected as
-- too risky. See Construction/Dictionary.txt's corrected "Showing vs Show"
-- note.
--
-- A Show inherits its parent Show Run's roster/authority wholesale -- it
-- has no roster table of its own this kernel.

CREATE TABLE IF NOT EXISTS shows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_run_id UUID NOT NULL REFERENCES show_runs(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  audience_title TEXT,
  audience_program_blurb TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  scheduled_start_at TIMESTAMPTZ,
  scheduled_end_at TIMESTAMPTZ,
  actual_start_at TIMESTAMPTZ,
  actual_end_at TIMESTAMPTZ,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  CONSTRAINT shows_status_check CHECK (status IN (
    'draft', 'scheduled', 'live', 'paused', 'completed', 'cancelled', 'archived'
  )),
  CONSTRAINT shows_archived_matches_status CHECK (
    (status = 'archived' AND archived_at IS NOT NULL) OR
    (status <> 'archived' AND archived_at IS NULL)
  ),
  CONSTRAINT shows_scheduled_end_after_start CHECK (
    scheduled_start_at IS NULL OR scheduled_end_at IS NULL OR
    scheduled_end_at >= scheduled_start_at
  ),
  UNIQUE (show_run_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_shows_show_run_id ON shows(show_run_id);
CREATE INDEX IF NOT EXISTS idx_shows_status ON shows(status);

-- Minimal, separate manual link so a Session can point back at its Show
-- without rewiring network/session_control.go's venue-triggered start flow
-- (Kernel 67 explicitly does not replace command-based session start).
-- Nullable, one-directional, no join table: a session belongs to zero or
-- one Show.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS show_id UUID REFERENCES shows(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_show_id ON sessions(show_id) WHERE show_id IS NOT NULL;

COMMIT;
