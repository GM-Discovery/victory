BEGIN;

-- Kernel 91 follow-up: tour_progress is a resumable cursor, distinct from
-- tour_completions (migration 107). A click-gated step whose target is a
-- venue pin (Audition Hall, Trailer) causes a real page navigation the
-- instant it's clicked (kernel-91 S5: "prefer click-through over
-- teleporting"), which can race and cancel an ordinary fetch() before it
-- lands. Without a durable mid-tour cursor, every return to the map after
-- such a click restarted the mandatory tour from step 0 -- a real bug found
-- in production (Grant's report: "stuck in a loop of resetting the tour to
-- audition hall").
--
-- This table only ever holds "how far did they get," is overwritten in
-- place (not an append-only historical fact like tour_completions), and is
-- deleted once the tour is actually completed or skipped -- see
-- tour.RecordCompletion's cleanup. It is written via navigator.sendBeacon,
-- which is specifically designed to survive the page-unload race this
-- table exists to close.
CREATE TABLE IF NOT EXISTS tour_progress (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tour_key TEXT NOT NULL,
  venue_slug TEXT,
  role_key TEXT,
  step_reached TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT tour_progress_tour_key_check CHECK (tour_key IN (
    'campus_mandatory',
    'campus_continuation',
    'catharsis_cast',
    'directors_chair_director_toolbox'
  ))
);

-- Same COALESCE rationale as uq_tour_completions_scope in migration 107.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tour_progress_scope
  ON tour_progress (user_id, tour_key, COALESCE(venue_slug, ''), COALESCE(role_key, ''));

COMMIT;
