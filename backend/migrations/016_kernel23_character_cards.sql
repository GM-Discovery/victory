BEGIN;

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
  ON permission_grants (
    location_id,
    grantee_user_id,
    capability,
    COALESCE(production_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(venue_id, '00000000-0000-0000-0000-000000000000'::uuid)
  )
  WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_permission_grants_grantee
  ON permission_grants(grantee_user_id);

CREATE TABLE IF NOT EXISTS character_cards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  pronouns TEXT NOT NULL DEFAULT '',
  portrait_url TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '#d9c7a6',
  tagline TEXT NOT NULL DEFAULT '',
  public_description TEXT NOT NULL DEFAULT '',
  private_notes TEXT NOT NULL DEFAULT '',
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

COMMIT;
