BEGIN;

-- Kernel 62: private, directional player-to-player relationship records.
-- observer_user_id owns the row absolutely; subject_user_id never gains any
-- read or write path to it (Kernel 62 §4.1, §4.2). The record references the
-- subject's stable account UUID so it survives stage-name changes
-- (Kernel 62 §5.1).
CREATE TABLE IF NOT EXISTS player_relationships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  observer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  private_nickname TEXT NOT NULL DEFAULT '',
  relationship_state TEXT NOT NULL DEFAULT 'active',
  trust_level TEXT NOT NULL DEFAULT 'unknown',
  closeness_level TEXT NOT NULL DEFAULT 'unknown',
  reliability_level TEXT NOT NULL DEFAULT 'unknown',
  communication_ease_level TEXT NOT NULL DEFAULT 'unknown',
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (observer_user_id, subject_user_id),
  CONSTRAINT player_relationships_not_self CHECK (observer_user_id <> subject_user_id)
);

CREATE INDEX IF NOT EXISTS idx_player_relationships_observer
  ON player_relationships(observer_user_id, updated_at);

-- Private checkbox categories; multiple allowed, custom requires a label
-- (Kernel 62 §5.2). Categories are labels only -- they never grant authority.
CREATE TABLE IF NOT EXISTS player_relationship_categories (
  relationship_id UUID NOT NULL REFERENCES player_relationships(id) ON DELETE CASCADE,
  category_key TEXT NOT NULL,
  custom_label TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (relationship_id, category_key, custom_label)
);

-- Effective current relationship facts, one row per field_key, rebuilt
-- deterministically from remaining events (Kernel 62 §5.3) -- same
-- full-replace recompute model as player_profile_facts.
CREATE TABLE IF NOT EXISTS player_relationship_facts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  relationship_id UUID NOT NULL REFERENCES player_relationships(id) ON DELETE CASCADE,
  field_key TEXT NOT NULL,
  value_json JSONB NOT NULL DEFAULT 'null'::jsonb,
  display_value TEXT NOT NULL DEFAULT '',
  source_event_id UUID,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (relationship_id, field_key)
);

CREATE INDEX IF NOT EXISTS idx_player_relationship_facts_relationship
  ON player_relationship_facts(relationship_id);

-- Typed, observer-private relationship events (page commits). Deleting an
-- ordinary event is a real row delete followed by fact recompute
-- (Kernel 62 §5.4).
CREATE TABLE IF NOT EXISTS player_relationship_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  relationship_id UUID NOT NULL REFERENCES player_relationships(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  page_key TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  human_summary TEXT NOT NULL DEFAULT '',
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_player_relationship_events_relationship
  ON player_relationship_events(relationship_id, created_at);

-- Private dated journal entries about one person (Kernel 62 §5.5). Soft
-- delete via deleted_at; the body never reaches any surface the subject can
-- read.
CREATE TABLE IF NOT EXISTS player_relationship_journal_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  relationship_id UUID NOT NULL REFERENCES player_relationships(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  entry_date DATE,
  note_category TEXT NOT NULL DEFAULT 'general',
  tags TEXT[] NOT NULL DEFAULT '{}',
  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  session_id UUID,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_player_relationship_journal_relationship
  ON player_relationship_journal_entries(relationship_id, created_at);

-- Stored private follow-up intentions. Deliberately no scheduler, reminder,
-- notification, or calendar surface exists for these rows (Kernel 62 §4.5).
CREATE TABLE IF NOT EXISTS player_relationship_followups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  relationship_id UUID NOT NULL REFERENCES player_relationships(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  target_date DATE,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_player_relationship_followups_relationship
  ON player_relationship_followups(relationship_id, status);

COMMIT;
