-- Kernel 72: DDL + token_aura backfill extracted verbatim from characters.EnsureKernel23CharacterSurface (function removed).
CREATE TABLE IF NOT EXISTS permission_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  grantee_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  capability TEXT NOT NULL,
  production_id UUID REFERENCES productions(id) ON DELETE CASCADE,
  venue_id UUID REFERENCES venues(id) ON DELETE CASCADE,
  granted_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS permission_grants_active_unique
  ON permission_grants(location_id, grantee_user_id, capability, COALESCE(production_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(venue_id, '00000000-0000-0000-0000-000000000000'::uuid))
  WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_permission_grants_grantee
  ON permission_grants(grantee_user_id);

CREATE TABLE IF NOT EXISTS character_cards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  workbook_status TEXT NOT NULL DEFAULT 'draft',
  workbook_context JSONB NOT NULL DEFAULT '{}'::jsonb,
  name TEXT NOT NULL,
  pronouns TEXT NOT NULL DEFAULT '',
  portrait_url TEXT NOT NULL DEFAULT '',
  token_aura TEXT,
  color TEXT NOT NULL DEFAULT '#d9c7a6',
  tagline TEXT NOT NULL DEFAULT '',
  public_description TEXT NOT NULL DEFAULT '',
  private_notes TEXT NOT NULL DEFAULT '',
  sheet_links JSONB NOT NULL DEFAULT '[]'::jsonb,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS sheet_links JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS workbook_status TEXT NOT NULL DEFAULT 'draft';
ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS workbook_context JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE character_cards
  ADD COLUMN IF NOT EXISTS token_aura TEXT;
UPDATE character_cards
   SET token_aura = color
 WHERE token_aura IS NULL
   AND color IS NOT NULL
   AND color <> ''
   AND lower(color) <> '#d9c7a6';

CREATE INDEX IF NOT EXISTS idx_character_cards_owner
  ON character_cards(owner_user_id);

CREATE INDEX IF NOT EXISTS idx_character_cards_location
  ON character_cards(location_id);

CREATE TABLE IF NOT EXISTS current_session_personas (
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  equipped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (session_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_current_session_personas_card
  ON current_session_personas(character_card_id);

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

CREATE TABLE IF NOT EXISTS character_workbook_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  module_instance_id UUID REFERENCES character_workbook_modules(id) ON DELETE SET NULL,
  author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  page_key TEXT NOT NULL DEFAULT 'history',
  entry_type TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  stage_number INT,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_character_workbook_entries_character
  ON character_workbook_entries(character_card_id, sort_order ASC, created_at ASC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_character_workbook_modules_unique
  ON character_workbook_modules(
    character_card_id,
    ruleset_key
  );

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
