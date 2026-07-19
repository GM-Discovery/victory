BEGIN;

-- Kernel 70: Go Cue foundation. A Cue is a stable, Show-Scene-Placement-
-- scoped record with an internal/backstage name, an optional player-facing
-- stage-button label, a trigger-permission scope, and an ordered JSON
-- action list. The visible stage-button label is never the Cue's identity
-- -- the button always triggers the underlying Cue by id.
--
-- Deliberately one Cue table with a validated ordered-action JSON array
-- (not a table per action type, not a script node graph) and one
-- execution-log table for atomicity/idempotency/audit, per the kernel's
-- explicit data-restraint requirement (SS10).

CREATE TABLE IF NOT EXISTS cues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_scene_placement_id UUID NOT NULL REFERENCES show_scene_placements(id) ON DELETE CASCADE,
  internal_name TEXT NOT NULL,
  stage_button_label TEXT,
  trigger_scope TEXT NOT NULL DEFAULT 'director_crew_only',
  sort_order INTEGER NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  actions JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT cues_trigger_scope_check CHECK (trigger_scope IN (
    'director_crew_only', 'players_may_trigger', 'any_roster_member'
  ))
);

CREATE INDEX IF NOT EXISTS idx_cues_show_scene_placement_id ON cues(show_scene_placement_id);

-- Cue execution log -- one row per GO press, recording the outcome of
-- every individual action inside cues.actions in order, so a partial
-- failure is visible (status = 'partial_failure' with per-action detail in
-- action_results) rather than silently swallowed. idempotency_key is
-- client-supplied per GO-button-press; the UNIQUE index below is the real
-- double-press/race guard, not just an application-side check.
CREATE TABLE IF NOT EXISTS cue_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cue_id UUID NOT NULL REFERENCES cues(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  triggered_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'in_progress',
  action_results JSONB NOT NULL DEFAULT '[]'::jsonb,
  error_message TEXT,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  CONSTRAINT cue_executions_status_check CHECK (status IN (
    'in_progress', 'succeeded', 'partial_failure', 'failed'
  ))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cue_executions_idempotency
  ON cue_executions(cue_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_cue_executions_cue_id ON cue_executions(cue_id);
CREATE INDEX IF NOT EXISTS idx_cue_executions_show_id ON cue_executions(show_id);

COMMIT;
