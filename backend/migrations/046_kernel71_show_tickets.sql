BEGIN;

-- Kernel 71: the two-punch Show ticket. Either side (Player or Director)
-- may punch first; the second punch is the sole atomic transaction that
-- creates real Player participation (backend/internal/tickets.SecondPunch).
-- One compact record represents both request directions rather than
-- separate invitation/request tables (spec §4.1). requested_role is
-- CHECK-locked to 'player' -- Kernel 71 implements Player tickets only; a
-- future Audience-ticket kernel widens this CHECK rather than redesigning
-- the table.
CREATE TABLE IF NOT EXISTS show_run_tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_run_id UUID NOT NULL REFERENCES show_runs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  requested_role TEXT NOT NULL DEFAULT 'player',
  initiated_by_side TEXT NOT NULL,
  player_punched_at TIMESTAMPTZ,
  player_punched_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  director_punched_at TIMESTAMPTZ,
  director_punched_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'pending_player',
  message TEXT,
  roster_membership_id UUID REFERENCES show_run_roster_members(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at TIMESTAMPTZ,
  CONSTRAINT show_run_tickets_requested_role_check CHECK (requested_role = 'player'),
  CONSTRAINT show_run_tickets_initiated_by_side_check CHECK (initiated_by_side IN ('player', 'director')),
  CONSTRAINT show_run_tickets_status_check CHECK (status IN (
    'pending_player', 'pending_director', 'valid', 'declined', 'withdrawn'
  ))
);

-- Prevents multiple conflicting *active* tickets for the same user/show
-- run/role -- only one row may be unresolved (pending_player/
-- pending_director) at a time. A declined/withdrawn/valid row does not
-- block a fresh ticket later (spec §4.1, §4.4).
CREATE UNIQUE INDEX IF NOT EXISTS uq_show_run_tickets_active
  ON show_run_tickets(show_run_id, user_id, requested_role)
  WHERE status IN ('pending_player', 'pending_director');

CREATE INDEX IF NOT EXISTS idx_show_run_tickets_show_run_id ON show_run_tickets(show_run_id);
CREATE INDEX IF NOT EXISTS idx_show_run_tickets_user_id ON show_run_tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_show_run_tickets_status ON show_run_tickets(status);

COMMIT;
