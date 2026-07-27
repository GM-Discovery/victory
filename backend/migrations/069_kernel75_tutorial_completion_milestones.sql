BEGIN;

-- Kernel 75 S3.1/S3.2: the Locked Courtyard tutorial gains a real ending.
--
-- Kernel 74 stopped at tutorial_handoff_entered -- the Player was moved to a
-- handoff Scene and that was the whole ending. K75 splits the last beat in
-- two around the Player's own Continue press, so two new milestones are
-- needed:
--
--   tutorial_gate_opened  -- Ra exposes the concealed lock, works it, the
--                            bolt withdraws, the gate opens. A narrative
--                            fact, recorded when the closing beats are
--                            generated so that reopening the completion
--                            Program later replays the same ending rather
--                            than re-deriving it.
--
--   tutorial_completed    -- the Player pressed Continue. This is the
--                            Player's acknowledgement, and it is the
--                            idempotency anchor for Story So Far generation
--                            (migration 070) and one-time Player
--                            recognition: both are keyed off "has this
--                            milestone already been recorded", so a
--                            double-clicked Continue converges instead of
--                            duplicating a Player's history.
--
-- Extending the CHECK in place rather than adding a parallel progress table
-- is deliberate. participant_tutorial_progress is read by exactly three
-- things -- the reveal gate (world.milestoneGate), the invocation gate
-- (merchant.requireBindingMilestone), and the Directors+ readiness list
-- (tutorial.ListShowProgress). A second table would fork all three and force
-- a UNION into the one function the Director's Chair reads. Migration 066
-- already established this extension pattern for
-- participant_interactions_type_check.
--
-- tutorial/progress.go mirrors this list by hand in allMilestones. The two
-- must be changed together; IsMilestone is the Go-side gate that turns a
-- typo into a clean error instead of a constraint violation surfacing as a
-- 500.
ALTER TABLE participant_tutorial_progress
  DROP CONSTRAINT IF EXISTS participant_tutorial_progress_milestone_check;

ALTER TABLE participant_tutorial_progress
  ADD CONSTRAINT participant_tutorial_progress_milestone_check CHECK (milestone_key IN (
    'kessa_intro_completed',
    'door_intention_submitted',
    'ra_intro_started',
    'ra_intro_completed',
    'tutorial_gate_opened',
    'tutorial_handoff_entered',
    'tutorial_completed'
  ));

COMMIT;
