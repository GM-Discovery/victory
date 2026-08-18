BEGIN;

-- Kernel 91: Victory Campus Tours & Guided Onboarding.
--
-- tour_completions is the durable, server-authoritative record of which
-- guided tours an account has completed or skipped (kernel-91 S12: "Do not
-- rely on localStorage as the only source of truth"). It replaces the old
-- map welcome modal's localStorage-only flag (victory:map-onboarding-v1)
-- with a real account-scoped fact.
--
-- Deliberately bounded like participant_tutorial_progress (migration 066):
-- tour_key is CHECK-constrained rather than a free-text/config-driven
-- column, so "which tours exist" stays a schema fact a reviewer can read,
-- and adding a tour is a migration on purpose (kernel-91 S30: this is not a
-- generic course-builder/LMS).
--
-- Unlike participant_tutorial_progress, a tour has no Character or Show
-- scope (kernel-91 S11-S13: tours are per-user, optionally per-venue and
-- per-role, not per-Character-participation) -- hence the different key
-- shape: (user_id, tour_key, venue_slug, role_key) rather than (user_id,
-- character_card_id, show_id, milestone_key).
CREATE TABLE IF NOT EXISTS tour_completions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tour_key TEXT NOT NULL,
  venue_slug TEXT,          -- NULL for campus/map-scoped tours
  role_key TEXT,            -- NULL for tours with no specific role gate
  status TEXT NOT NULL CHECK (status IN ('completed', 'skipped')),
  step_reached TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT tour_completions_tour_key_check CHECK (tour_key IN (
    'campus_mandatory',
    'campus_continuation',
    'catharsis_cast',
    'directors_chair_director_toolbox'
  ))
);

-- A table-level UNIQUE(...) cannot contain an expression like COALESCE(), so
-- the idempotency floor is a unique expression index instead. COALESCE is
-- required here, not decorative: Postgres treats a bare NULL as distinct
-- from every other NULL in a unique index, so two skip/complete calls on a
-- campus-scoped tour (venue_slug and role_key both NULL) would not collide
-- without it and would silently double-insert. This is the one deliberate
-- schema deviation from participant_tutorial_progress, which has no
-- nullable column in its own UNIQUE key -- do not copy a bare-NULL
-- UNIQUE(...) pattern onto a nullable-scoped table without this.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tour_completions_scope
  ON tour_completions (user_id, tour_key, COALESCE(venue_slug, ''), COALESCE(role_key, ''));

CREATE INDEX IF NOT EXISTS idx_tour_completions_user ON tour_completions(user_id);

COMMIT;
