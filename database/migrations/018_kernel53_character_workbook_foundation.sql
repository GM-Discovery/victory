BEGIN;

ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS workbook_status TEXT NOT NULL DEFAULT 'draft';

ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS workbook_context JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS active_user_characters (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS character_workbook_modules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  ruleset_key TEXT NOT NULL,
  ruleset_version TEXT NOT NULL DEFAULT '',
  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,
  module_status TEXT NOT NULL DEFAULT 'draft',
  creation_flow_version TEXT NOT NULL DEFAULT 'v1',
  parentage_chart_version TEXT NOT NULL DEFAULT '',
  current_stage INT NOT NULL DEFAULT 1,
  current_event TEXT NOT NULL DEFAULT '',
  module_context JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_character_workbook_modules_unique
  ON character_workbook_modules(character_card_id, ruleset_key);

CREATE TABLE IF NOT EXISTS character_journals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  module_instance_id UUID REFERENCES character_workbook_modules(id) ON DELETE SET NULL,
  author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  visibility TEXT NOT NULL DEFAULT 'private',
  body TEXT NOT NULL DEFAULT '',
  venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,
  session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  showing_id UUID REFERENCES showings(id) ON DELETE SET NULL,
  archived_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_character_journals_character
  ON character_journals(character_card_id, created_at DESC);

COMMIT;
