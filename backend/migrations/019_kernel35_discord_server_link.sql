BEGIN;

CREATE TABLE IF NOT EXISTS auth.discord_server_links (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  discord_guild_id TEXT NOT NULL,
  discord_guild_name TEXT,
  system_channel_id TEXT,
  system_channel_name TEXT,
  bot_verified BOOLEAN NOT NULL DEFAULT FALSE,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  linked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  linked_at TIMESTAMPTZ,
  removed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_discord_server_links_guild_id
  ON auth.discord_server_links(discord_guild_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_server_links_active
  ON auth.discord_server_links(active);

COMMIT;
