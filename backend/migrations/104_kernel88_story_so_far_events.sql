BEGIN;

-- Kernel 88: widen Story So Far's fixed event_type/source_kind vocabularies
-- (migration 070) to admit two new deterministic template clauses --
-- Director-awarded Fate and resolved Help/interrupt outcomes. Additive
-- only, matching kernel-75's closed-vocabulary design: Postgres enforces
-- that storysofar's template library (backend/internal/storysofar/
-- templates.go) can never silently invent an event_type/source_kind that
-- isn't one of these two lists.
ALTER TABLE character_story_events
  DROP CONSTRAINT character_story_events_type_check;
ALTER TABLE character_story_events
  ADD CONSTRAINT character_story_events_type_check CHECK (event_type IN (
    'arrival',
    'merchant_met',
    'merchant_stance',
    'merchant_haggle',
    'equipment_acquired',
    'door_intention',
    'dialogue_learned',
    'gate_opened',
    'face_sheet_echo',
    'director_moment',
    'aftercare',
    'tutorial_completed',
    'fate_points_awarded',
    'help_resolved'
  ));

ALTER TABLE character_story_events
  DROP CONSTRAINT character_story_events_source_kind_check;
ALTER TABLE character_story_events
  ADD CONSTRAINT character_story_events_source_kind_check CHECK (source_kind IN (
    '',
    'tutorial_milestone',
    'inventory_item',
    'freeform_submission',
    'interaction_attempt',
    'dialogue_topic',
    'workbook_entry',
    'character_journal',
    'aftercare',
    'socio_fate_ledger',
    'socio_pending_action'
  ));

COMMIT;
