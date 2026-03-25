CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'location_role'
  ) THEN
    CREATE TYPE location_role AS ENUM (
      'producer',
      'director',
      'cast',
      'crew',
      'audience'
    );
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'session_status'
  ) THEN
    CREATE TYPE session_status AS ENUM (
      'rehearsal',
      'live',
      'closed'
    );
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'element_state'
  ) THEN
    CREATE TYPE element_state AS ENUM (
      'library',
      'production',
      'session'
    );
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  display_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS locations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS location_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role location_role NOT NULL,
  granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (location_id, user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_location_memberships_location_id
  ON location_memberships(location_id);

CREATE INDEX IF NOT EXISTS idx_location_memberships_user_id
  ON location_memberships(user_id);

CREATE TABLE IF NOT EXISTS lots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_lots_location_id
  ON lots(location_id);

CREATE TABLE IF NOT EXISTS venues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lot_id UUID NOT NULL REFERENCES lots(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'presentation',
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (lot_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_venues_lot_id
  ON venues(lot_id);

CREATE TABLE IF NOT EXISTS libraries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_libraries_location_id
  ON libraries(location_id);

CREATE TABLE IF NOT EXISTS elements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  library_id UUID NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  element_type TEXT NOT NULL,
  state element_state NOT NULL DEFAULT 'library',
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (library_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_elements_library_id
  ON elements(library_id);

CREATE TABLE IF NOT EXISTS venue_layout_elements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
  element_id UUID NOT NULL REFERENCES elements(id) ON DELETE CASCADE,
  surface TEXT NOT NULL DEFAULT 'stage',
  position JSONB NOT NULL DEFAULT '{}'::jsonb,
  visibility JSONB NOT NULL DEFAULT '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}'::jsonb,
  is_default BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (venue_id, element_id, surface)
);

CREATE INDEX IF NOT EXISTS idx_venue_layout_elements_venue_id
  ON venue_layout_elements(venue_id);

CREATE INDEX IF NOT EXISTS idx_venue_layout_elements_element_id
  ON venue_layout_elements(element_id);

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE RESTRICT,
  status session_status NOT NULL DEFAULT 'rehearsal',
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sessions_venue_id
  ON sessions(venue_id);

CREATE INDEX IF NOT EXISTS idx_sessions_status
  ON sessions(status);

CREATE TABLE IF NOT EXISTS session_participants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role location_role NOT NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at TIMESTAMPTZ,
  UNIQUE (session_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_session_participants_session_id
  ON session_participants(session_id);

CREATE INDEX IF NOT EXISTS idx_session_participants_user_id
  ON session_participants(user_id);

CREATE TABLE IF NOT EXISTS actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  moment_id BIGINT NOT NULL,
  actor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  type TEXT NOT NULL,
  target JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  scope JSONB NOT NULL DEFAULT '{"surfaces":["stage"],"audienceSegments":["all"]}'::jsonb,
  visibility JSONB NOT NULL DEFAULT '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}'::jsonb,
  recorded BOOLEAN NOT NULL DEFAULT TRUE,
  ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (session_id, moment_id)
);

CREATE INDEX IF NOT EXISTS idx_actions_session_id
  ON actions(session_id);

CREATE INDEX IF NOT EXISTS idx_actions_actor_id
  ON actions(actor_id);

CREATE INDEX IF NOT EXISTS idx_actions_type
  ON actions(type);

CREATE INDEX IF NOT EXISTS idx_actions_ts
  ON actions(ts);