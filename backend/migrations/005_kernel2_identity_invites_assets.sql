-- database/migrations/005_kernel2_identity_invites_assets.sql
BEGIN;

-- Keep existing pgcrypto usage (already enabled in 001), but harmless if repeated.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Auth/local schema to avoid collisions with public.sessions (venue sessions).
CREATE SCHEMA IF NOT EXISTS auth;

-- ---------------------------------------------------------------------------
-- Users: add email + updated_at (email nullable to avoid breaking legacy join flow)
-- ---------------------------------------------------------------------------
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email TEXT,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Case-insensitive uniqueness for email when present.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_ci
  ON users ((lower(email)))
  WHERE email IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Auth: password credentials
-- password_hash stores a PHC-style Argon2id string (recommended for portability).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth.password_credentials (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Auth: server-side sessions
-- token_hash stores SHA-256(raw_token) as BYTEA.
-- cookie stores raw_token; DB never stores raw token.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth.sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  token_hash BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,

  revoked_at TIMESTAMPTZ,
  ip INET,
  user_agent TEXT,

  CONSTRAINT auth_sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id
  ON auth.sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at
  ON auth.sessions(expires_at);

-- ---------------------------------------------------------------------------
-- Auth: password reset tokens (email reset link flow)
-- token_hash stores SHA-256(raw_token) as BYTEA.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth.password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  token_hash BYTEA NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,

  request_ip INET,
  request_user_agent TEXT,

  CONSTRAINT password_reset_tokens_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id
  ON auth.password_reset_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at
  ON auth.password_reset_tokens(expires_at);

-- ---------------------------------------------------------------------------
-- Productions: minimal table to support director-scoped authority.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS productions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,

  name TEXT NOT NULL,
  slug TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_productions_location_id
  ON productions(location_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'productions_location_id_fkey'
  ) THEN
    ALTER TABLE productions
      ADD CONSTRAINT productions_location_id_fkey
      FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE CASCADE;
  END IF;
END
$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'productions_location_slug_unique'
  ) THEN
    ALTER TABLE productions
      ADD CONSTRAINT productions_location_slug_unique UNIQUE (location_id, slug);
  END IF;
END
$$;

-- ---------------------------------------------------------------------------
-- Invites: general access mechanism with scoped role assignment.
-- token_hash stores SHA-256(raw_token).
-- usage controls: max_uses + uses_count, with optional expiry and revocation.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  inviter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  target_role location_role NOT NULL,

  target_email TEXT,
  target_handle TEXT,

  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,

  token_hash BYTEA NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,

  max_uses INTEGER NOT NULL DEFAULT 1,
  uses_count INTEGER NOT NULL DEFAULT 0,

  revoked_at TIMESTAMPTZ,
  accepted_first_at TIMESTAMPTZ,
  accepted_last_at TIMESTAMPTZ,

  CONSTRAINT invites_token_hash_unique UNIQUE (token_hash),
  CONSTRAINT invites_max_uses_gt_zero CHECK (max_uses > 0),
  CONSTRAINT invites_uses_in_range CHECK (uses_count >= 0 AND uses_count <= max_uses),

  -- Kernel rule: director authority is production-scoped.
  CONSTRAINT invites_director_requires_production CHECK (
    (target_role <> 'director') OR (production_id IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_invites_location_id
  ON invites(location_id);

CREATE INDEX IF NOT EXISTS idx_invites_expires_at
  ON invites(expires_at);

CREATE INDEX IF NOT EXISTS idx_invites_inviter_user_id
  ON invites(inviter_user_id);

-- ---------------------------------------------------------------------------
-- Memberships: role assignment, scoped to location + optional production/venue.
-- active flag allows soft-disable without deleting history.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  role location_role NOT NULL,

  production_id UUID REFERENCES productions(id) ON DELETE CASCADE,
  venue_id UUID REFERENCES venues(id) ON DELETE CASCADE,

  granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Kernel rule: director, cast, crew should be production-scoped
  CONSTRAINT memberships_director_requires_production CHECK (
    (role <> 'director') OR (production_id IS NOT NULL)
  ),
  CONSTRAINT memberships_cast_requires_production CHECK (
    (role <> 'cast') OR (production_id IS NOT NULL)
  ),
  CONSTRAINT memberships_crew_requires_production CHECK (
    (role <> 'crew') OR (production_id IS NOT NULL)
  )
);

-- Uniqueness with NULL-safe semantics via partial unique indexes:
CREATE UNIQUE INDEX IF NOT EXISTS memberships_unique_location_role
  ON memberships(location_id, user_id, role)
  WHERE production_id IS NULL AND venue_id IS NULL AND active = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS memberships_unique_production_role
  ON memberships(location_id, production_id, user_id, role)
  WHERE production_id IS NOT NULL AND venue_id IS NULL AND active = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS memberships_unique_venue_role
  ON memberships(location_id, venue_id, user_id, role)
  WHERE venue_id IS NOT NULL AND active = TRUE;

CREATE INDEX IF NOT EXISTS idx_memberships_user_id
  ON memberships(user_id);

CREATE INDEX IF NOT EXISTS idx_memberships_location_id
  ON memberships(location_id);

-- ---------------------------------------------------------------------------
-- Access grants: explicit venue or production grants used by map visibility resolver.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'access_grant_type') THEN
    CREATE TYPE access_grant_type AS ENUM (
      'venue_access',
      'map_visibility'
    );
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS access_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  grant_type access_grant_type NOT NULL,

  venue_id UUID REFERENCES venues(id) ON DELETE CASCADE,
  production_id UUID REFERENCES productions(id) ON DELETE CASCADE,

  granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,

  CONSTRAINT access_grants_requires_scope CHECK (
    venue_id IS NOT NULL OR production_id IS NOT NULL
  )
);

CREATE INDEX IF NOT EXISTS idx_access_grants_user_id
  ON access_grants(user_id);

CREATE INDEX IF NOT EXISTS idx_access_grants_location_id
  ON access_grants(location_id);

CREATE INDEX IF NOT EXISTS idx_access_grants_venue_id
  ON access_grants(venue_id);

CREATE INDEX IF NOT EXISTS idx_access_grants_production_id
  ON access_grants(production_id);

-- ---------------------------------------------------------------------------
-- Venues: map visibility and workshop identification
-- ---------------------------------------------------------------------------
ALTER TABLE venues
  ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_workshop BOOLEAN NOT NULL DEFAULT FALSE;

-- Seed: ensure InfoBooth and Workshop exist at the welcome gate / main-lot
-- using the known seed world location/lot slugs.
WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
),
lot_row AS (
  SELECT id FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'InfoBooth',
  'info-booth',
  'info',
  '{ "map": { "always_visible": true }, "staffed_by": "daemon", "open": true }'::jsonb,
  TRUE,
  FALSE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row) AND slug = 'info-booth'
);

WITH location_row AS (
  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
),
lot_row AS (
  SELECT id FROM lots
  WHERE location_id = (SELECT id FROM location_row)
    AND slug = 'main-lot'
  LIMIT 1
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  (SELECT id FROM lot_row),
  'Workshop',
  'workshop',
  'workshop',
  '{ "workshop": { "asset_ingest": true } }'::jsonb,
  FALSE,
  TRUE
WHERE NOT EXISTS (
  SELECT 1 FROM venues
  WHERE lot_id = (SELECT id FROM lot_row) AND slug = 'workshop'
);

-- ---------------------------------------------------------------------------
-- Assets: ownership + storage metadata
-- ---------------------------------------------------------------------------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'asset_owner_state') THEN
    CREATE TYPE asset_owner_state AS ENUM (
      'personal',
      'uploader_owned',
      'lent_for_use',
      'production_licensed',
      'production_usable',
      'library_owned'
    );
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- "Producer-owned server instance" storage owner
  producer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,

  uploader_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  owner_state asset_owner_state NOT NULL,

  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,

  original_filename TEXT,
  source_ext TEXT,
  source_mime TEXT,
  sniffed_mime TEXT,

  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  byte_size INTEGER NOT NULL,

  checksum_sha256 BYTEA NOT NULL,

  storage_root TEXT NOT NULL,
  original_path TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_assets_location_id
  ON assets(location_id);

CREATE INDEX IF NOT EXISTS idx_assets_producer_user_id
  ON assets(producer_user_id);

CREATE INDEX IF NOT EXISTS idx_assets_uploader_user_id
  ON assets(uploader_user_id);

CREATE INDEX IF NOT EXISTS idx_assets_production_id
  ON assets(production_id);

CREATE INDEX IF NOT EXISTS idx_assets_checksum
  ON assets(checksum_sha256);

CREATE TABLE IF NOT EXISTS asset_derivatives (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,

  -- For this kernel: "512", "1024", "2048", and optionally "original"
  variant_key TEXT NOT NULL,

  width INTEGER NOT NULL,
  height INTEGER NOT NULL,

  mime TEXT NOT NULL,
  byte_size INTEGER NOT NULL,
  checksum_sha256 BYTEA NOT NULL,

  path TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (asset_id, variant_key)
);

CREATE INDEX IF NOT EXISTS idx_asset_derivatives_asset_id
  ON asset_derivatives(asset_id);

COMMIT;
