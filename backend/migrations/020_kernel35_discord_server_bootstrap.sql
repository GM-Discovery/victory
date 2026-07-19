BEGIN;

CREATE TABLE IF NOT EXISTS auth.discord_server_link_settings (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  application_id TEXT,
  bot_token TEXT,
  redirect_url TEXT,
  permissions TEXT,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_server_link_settings_enabled
  ON auth.discord_server_link_settings(enabled);

COMMIT;
