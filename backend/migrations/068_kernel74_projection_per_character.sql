BEGIN;

-- Kernel 74 fix: a participant-local projection belongs to a (user,
-- CHARACTER, show) participation, not to (user, show).
--
-- Migration 066's partial unique index keyed only (user_id, show_id), and
-- projection.LoadActiveForViewer matched the same way. The consequence was a
-- real bug found in live play: a Player who finished the tutorial with one
-- Character, then switched to another, stayed stranded on the Outside the
-- Courtyard handoff map -- with a Character who had never met Kessa, never
-- tried the door, and never spoke to Ra.
--
-- That contradicts S5.2, which is explicit that progress belongs to the
-- selected Character participation context and must not transfer when the
-- selection changes. Every other Kernel 74 table already got this right
-- (participant_tutorial_progress, participant_freeform_submissions,
-- participant_dialogue_topic_views are all keyed on character_card_id);
-- the projection was the one that was not, despite carrying the column.
--
-- Widening the key is safe for existing rows: it can only ever split one
-- constraint group into several, never merge two.
DROP INDEX IF EXISTS uq_participant_local_projections_active;

CREATE UNIQUE INDEX IF NOT EXISTS uq_participant_local_projections_active
  ON participant_local_projections(user_id, character_card_id, show_id)
  WHERE cleared_at IS NULL;

COMMIT;
