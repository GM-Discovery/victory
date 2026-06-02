BEGIN;

CREATE TABLE IF NOT EXISTS auth.discord_chat_message_bridges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action_id UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  showing_id UUID REFERENCES showings(id) ON DELETE SET NULL,
  venue_slug TEXT NOT NULL,
  discord_server_id TEXT NOT NULL,
  discord_thread_id TEXT NOT NULL,
  discord_message_id TEXT,
  bridge_status TEXT NOT NULL DEFAULT 'pending',
  error_code TEXT,
  error_message TEXT,
  attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  mirrored_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT discord_chat_message_bridges_action_unique UNIQUE (action_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_message_bridges_location
  ON auth.discord_chat_message_bridges(location_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_message_bridges_venue
  ON auth.discord_chat_message_bridges(venue_slug);

CREATE INDEX IF NOT EXISTS idx_auth_discord_chat_message_bridges_status
  ON auth.discord_chat_message_bridges(bridge_status);

COMMIT;
