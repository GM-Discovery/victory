-- Kernel 72: DDL extracted verbatim from identity.EnsureKernel39DiscordGatewaySurface.
-- The Go bootstrap now carries only the discord_bridge user seed.
CREATE TABLE IF NOT EXISTS auth.discord_gateway_state (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  configured BOOLEAN NOT NULL DEFAULT FALSE,
  running BOOLEAN NOT NULL DEFAULT FALSE,
  connected BOOLEAN NOT NULL DEFAULT FALSE,
  session_id TEXT NOT NULL DEFAULT '',
  bot_user_id TEXT NOT NULL DEFAULT '',
  intents BIGINT NOT NULL DEFAULT 0,
  message_content_intent BOOLEAN NOT NULL DEFAULT FALSE,
  active_thread_count INTEGER NOT NULL DEFAULT 0,
  last_connected_at TIMESTAMPTZ,
  last_event_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth.discord_gateway_settings (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  debug_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth.discord_chat_imports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  venue_slug TEXT NOT NULL,
  session_id UUID,
  showing_id UUID,
  action_id UUID,
  discord_server_id TEXT NOT NULL,
  discord_thread_id TEXT NOT NULL,
  discord_channel_id TEXT,
  discord_message_id TEXT NOT NULL,
  discord_author_id TEXT NOT NULL,
  discord_author_username TEXT,
  discord_author_global_name TEXT,
  linked_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  import_status TEXT NOT NULL DEFAULT 'imported',
  message_created_at TIMESTAMPTZ,
  message_edited_at TIMESTAMPTZ,
  edit_action_id UUID REFERENCES actions(id) ON DELETE SET NULL,
  imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (discord_message_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_imports_location
  ON auth.discord_chat_imports(location_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_imports_session
  ON auth.discord_chat_imports(session_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_imports_thread
  ON auth.discord_chat_imports(discord_thread_id);

ALTER TABLE auth.discord_chat_imports
  ADD COLUMN IF NOT EXISTS edit_action_id UUID REFERENCES actions(id) ON DELETE SET NULL;

INSERT INTO users (handle, display_name)
SELECT 'discord_bridge', 'Discord Bridge'
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE handle = 'discord_bridge'
);
