BEGIN;

CREATE TABLE IF NOT EXISTS auth.discord_channel_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  discord_server_id TEXT NOT NULL,
  discord_channel_id TEXT NOT NULL,
  discord_channel_name TEXT NOT NULL,
  discord_channel_type TEXT NOT NULL,
  mapping_kind TEXT NOT NULL,
  victory_scope_kind TEXT NOT NULL,
  victory_scope_id UUID,
  victory_scope_slug TEXT,
  expected_name TEXT NOT NULL,
  parent_discord_channel_id TEXT,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  last_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (location_id, mapping_kind, victory_scope_kind, victory_scope_slug)
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_channel_mappings_location
  ON auth.discord_channel_mappings(location_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_channel_mappings_server
  ON auth.discord_channel_mappings(discord_server_id);

COMMIT;
