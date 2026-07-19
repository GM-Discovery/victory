BEGIN;

-- Kernel 61: Player Workbook root. One row per user account (existing or
-- future). Does not replace or reshape `users` -- ownership is by user_id
-- FK only, never a duplicated/derived UUID (Kernel 61 §6.1).
CREATE TABLE IF NOT EXISTS player_profile_workbooks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  catalogue_key TEXT NOT NULL DEFAULT 'player-profile',
  catalogue_version TEXT NOT NULL DEFAULT '1.0.0',
  projection_version TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Typed, append-only profile events (page commits + legacy imports). Deleting
-- an ordinary event is a real row delete (Kernel 61 §6.3) -- there is no
-- shadow/audit table for deleted ordinary history.
CREATE TABLE IF NOT EXISTS player_profile_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workbook_id UUID NOT NULL REFERENCES player_profile_workbooks(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  page_key TEXT NOT NULL DEFAULT '',
  catalogue_version TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  human_summary TEXT NOT NULL DEFAULT '',
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_player_profile_events_workbook
  ON player_profile_events(workbook_id, created_at);

-- Effective current facts, one row per field_key. Rebuilt deterministically
-- from remaining events after any event deletion (Kernel 61 §6.4).
CREATE TABLE IF NOT EXISTS player_profile_facts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workbook_id UUID NOT NULL REFERENCES player_profile_workbooks(id) ON DELETE CASCADE,
  field_key TEXT NOT NULL,
  value_json JSONB NOT NULL DEFAULT 'null'::jsonb,
  display_value TEXT NOT NULL DEFAULT '',
  source_event_id UUID NOT NULL REFERENCES player_profile_events(id) ON DELETE CASCADE,
  source_page_key TEXT NOT NULL DEFAULT '',
  catalogue_version TEXT NOT NULL DEFAULT '',
  effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workbook_id, field_key)
);

CREATE INDEX IF NOT EXISTS idx_player_profile_facts_workbook
  ON player_profile_facts(workbook_id);

-- Append-only stage-name ledger, tied to the account UUID directly (not the
-- workbook), since it must outlive any workbook/profile replacement
-- (Kernel 61 §6.5). The partial unique index enforces "exactly one open
-- interval per user" at the database level.
CREATE TABLE IF NOT EXISTS player_stage_name_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  stage_name TEXT NOT NULL,
  normalized_stage_name TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ,
  changed_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source TEXT NOT NULL DEFAULT 'owner_edit',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_player_stage_name_history_user
  ON player_stage_name_history(user_id, started_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_stage_name_history_current
  ON player_stage_name_history(user_id) WHERE ended_at IS NULL;

-- Trailer Face curation, keyed by field_key (Kernel 61 §6.6). Mirrors the
-- Kernel 59A character_face_overrides shape but deliberately has no lock
-- columns -- player profiles have no Director-equivalent authority.
CREATE TABLE IF NOT EXISTS player_profile_face_overrides (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workbook_id UUID NOT NULL REFERENCES player_profile_workbooks(id) ON DELETE CASCADE,
  field_key TEXT NOT NULL,
  visibility_mode TEXT NOT NULL DEFAULT 'inferred',
  priority_mode TEXT NOT NULL DEFAULT 'inferred',
  priority_score INT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workbook_id, field_key)
);

CREATE INDEX IF NOT EXISTS idx_player_profile_face_overrides_workbook
  ON player_profile_face_overrides(workbook_id);

COMMIT;
