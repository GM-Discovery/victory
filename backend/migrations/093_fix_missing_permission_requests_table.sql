-- Fixes a real bug: `permission_requests` has been read/written by
-- backend/internal/identity/requests.go and permissions.go (and read by
-- access/visibility.go's notification-count query) since this feature was
-- first built (commit 35348da, "Kernel 3" era), but no migration ever
-- created the table. Every /api/requests/create call -- including the
-- Audition Hall "Request Access" form's default Catharsis/cast
-- auto-approve path -- has been failing with a 500 the whole time. This
-- had gone unnoticed because the one account that predates this fix
-- (Operator) never needed to submit a request themselves, and
-- visibility.go's notification-count read tolerates the missing table
-- silently (a caught error just skips badge counts), while
-- requests.go/permissions.go's writes do not.
--
-- Schema derived directly from every column the existing Go code already
-- references: requested_role reuses the location_role enum (001_init.sql)
-- since every read casts it `::text` and permissions.go's manual-approval
-- path inserts it straight into memberships.role (also location_role) --
-- the same enum-reuse precedent Kernel 80's storyboard_grants.granted_role
-- established for a different table.
BEGIN;

CREATE TABLE IF NOT EXISTS permission_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  venue_slug TEXT NOT NULL,
  requested_role location_role NOT NULL,
  note TEXT,

  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'approved', 'denied')),

  reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_permission_requests_user_id
  ON permission_requests(user_id);

CREATE INDEX IF NOT EXISTS idx_permission_requests_status
  ON permission_requests(status);

CREATE INDEX IF NOT EXISTS idx_permission_requests_venue_slug
  ON permission_requests(venue_slug);

COMMIT;
