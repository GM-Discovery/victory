-- Kernel 72: DDL + idempotent backfill extracted verbatim from showings.EnsureKernel22ShowingSurface (function removed).
CREATE TABLE IF NOT EXISTS showings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE UNIQUE,
  production_id UUID NOT NULL REFERENCES productions(id) ON DELETE RESTRICT,
  venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE RESTRICT,
  run_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  status session_status NOT NULL DEFAULT 'rehearsal',
  audience_view_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_showings_production_id
  ON showings(production_id);

CREATE INDEX IF NOT EXISTS idx_showings_venue_id
  ON showings(venue_id);

CREATE INDEX IF NOT EXISTS idx_showings_status
  ON showings(status);

ALTER TABLE actions
  ADD COLUMN IF NOT EXISTS showing_id UUID REFERENCES showings(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_actions_showing_id
  ON actions(showing_id);

WITH eligible_sessions AS (
  SELECT
    s.id AS session_id,
    s.venue_id,
    s.status,
    s.started_at,
    (
      SELECT p.id
      FROM productions p
      WHERE p.location_id = l.id
      ORDER BY p.created_at ASC
      LIMIT 1
    ) AS production_id,
    (
      SELECT sp.user_id
      FROM session_participants sp
      WHERE sp.session_id = s.id
      ORDER BY sp.joined_at ASC
      LIMIT 1
    ) AS created_by
  FROM sessions s
  JOIN venues v ON v.id = s.venue_id
  JOIN lots lo ON lo.id = v.lot_id
  JOIN locations l ON l.id = lo.location_id
  WHERE s.status IN ('rehearsal', 'live')
),
inserted_showings AS (
  INSERT INTO showings (
    session_id,
    production_id,
    venue_id,
    run_id,
    status,
    audience_view_enabled,
    started_at,
    ended_at,
    created_by
  )
  SELECT
    session_id,
    production_id,
    venue_id,
    NULL,
    status,
    (status = 'live'),
    started_at,
    CASE WHEN status = 'closed' THEN NOW() ELSE NULL END,
    created_by
  FROM eligible_sessions
  WHERE production_id IS NOT NULL
    AND created_by IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM showings sh
      WHERE sh.session_id = eligible_sessions.session_id
    )
  RETURNING id, session_id
)
UPDATE actions a
SET showing_id = sh.id
FROM showings sh
WHERE sh.session_id = a.session_id
  AND a.showing_id IS NULL;
