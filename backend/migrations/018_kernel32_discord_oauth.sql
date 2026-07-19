BEGIN;

CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.discord_identities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  discord_user_id TEXT NOT NULL UNIQUE,
  username TEXT,
  global_name TEXT,
  discriminator TEXT,
  avatar TEXT,
  email TEXT,
  email_verified BOOLEAN,
  locale TEXT,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_discord_identities_user_id
  ON auth.discord_identities(user_id);

CREATE TABLE IF NOT EXISTS auth.oauth_states (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider TEXT NOT NULL,
  state_hash BYTEA NOT NULL UNIQUE,
  return_to TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_oauth_states_provider
  ON auth.oauth_states(provider);

CREATE INDEX IF NOT EXISTS idx_auth_oauth_states_expires_at
  ON auth.oauth_states(expires_at);

COMMIT;
