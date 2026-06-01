BEGIN;

ALTER TABLE auth.discord_server_link_settings
  ADD COLUMN IF NOT EXISTS public_key TEXT NOT NULL DEFAULT '';

UPDATE auth.discord_server_link_settings
SET public_key = COALESCE(public_key, '')
WHERE public_key IS NULL;

CREATE TABLE IF NOT EXISTS auth.discord_session_threads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,
  venue_slug TEXT NOT NULL,
  session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  showing_id UUID REFERENCES showings(id) ON DELETE SET NULL,
  discord_server_id TEXT NOT NULL,
  parent_channel_id TEXT NOT NULL,
  thread_id TEXT NOT NULL,
  thread_name TEXT NOT NULL,
  started_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  started_by_discord_user_id TEXT,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  showtime_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT discord_session_threads_unique_venue UNIQUE (location_id, venue_slug)
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_session_threads_location
  ON auth.discord_session_threads(location_id);

CREATE INDEX IF NOT EXISTS idx_auth_discord_session_threads_status
  ON auth.discord_session_threads(status);

CREATE INDEX IF NOT EXISTS idx_auth_discord_session_threads_thread
  ON auth.discord_session_threads(thread_id);

COMMIT;
