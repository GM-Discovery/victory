BEGIN;

-- Kernel 88: a small nested pending-action stack for Interrupt/Help
-- resolution (spec §13). Deliberately not a general workflow engine --
-- parent_id gives stack nesting, status is the only state machine, and
-- LIFO ordering is enforced by the Go layer's validation order (a parent
-- must still be open before a new interrupt attaches to it; resolving a
-- node requires its own children to already be resolved/cancelled), not by
-- a queue data structure.
CREATE TABLE IF NOT EXISTS socio_pending_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  parent_id UUID REFERENCES socio_pending_actions(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  actor_character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  skill_id TEXT,
  target_complexity INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  roll_action_id UUID,
  roll_total INTEGER,
  overage_bonus INTEGER NOT NULL DEFAULT 0,
  overage_consumed_at TIMESTAMPTZ,
  opened_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at TIMESTAMPTZ,

  CONSTRAINT socio_pending_actions_kind_check CHECK (kind IN ('primary', 'interrupt')),
  CONSTRAINT socio_pending_actions_status_check CHECK (status IN ('open', 'resolved', 'cancelled')),
  CONSTRAINT socio_pending_actions_interrupt_needs_parent CHECK (kind = 'primary' OR parent_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_socio_pending_actions_open
  ON socio_pending_actions(show_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_socio_pending_actions_parent
  ON socio_pending_actions(parent_id);

COMMIT;
