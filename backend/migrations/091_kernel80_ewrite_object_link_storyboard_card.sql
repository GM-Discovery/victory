-- Kernel 80: extend the reusable ewrite_object_links model (same idiom
-- migration 089 used for cue/index_card/scene_element/dialogue_topic, and
-- 084 originally set up for equipment_item) to a sixth object type:
-- storyboard cards, so a card's optional eWrite link (spec 1.12, 10) reuses
-- the existing link table and frontend/lib/ewrite-rule-link.js widget
-- rather than a parallel link system.
--
-- Kept as its own migration file rather than folded into 090 because it
-- ALTERs a table owned by the ewrite package, not storyboards -- a future
-- reader auditing ewrite_object_links's shape should find every change to
-- it under a migration named for what it added, not buried inside an
-- unrelated domain's schema file. Still one inseparable, sequential,
-- single-PR family with 090.
--
-- ON DELETE CASCADE matches every other object_type column here: a link is
-- a satellite row of the object it annotates, meaningless once that object
-- is gone.

BEGIN;

ALTER TABLE ewrite_object_links
  ADD COLUMN IF NOT EXISTS storyboard_card_id UUID REFERENCES storyboard_cards(id) ON DELETE CASCADE;

ALTER TABLE ewrite_object_links
  DROP CONSTRAINT IF EXISTS ewrite_object_links_type_check;
ALTER TABLE ewrite_object_links
  ADD CONSTRAINT ewrite_object_links_type_check
    CHECK (object_type IN ('equipment_item', 'cue', 'index_card', 'scene_element', 'dialogue_topic', 'storyboard_card'));

ALTER TABLE ewrite_object_links
  DROP CONSTRAINT IF EXISTS ewrite_object_links_fk_shape;
ALTER TABLE ewrite_object_links
  ADD CONSTRAINT ewrite_object_links_fk_shape
    CHECK (
      (object_type = 'equipment_item') = (equipment_item_id IS NOT NULL) AND
      (object_type = 'cue') = (cue_id IS NOT NULL) AND
      (object_type = 'index_card') = (index_card_element_id IS NOT NULL) AND
      (object_type = 'scene_element') = (scene_element_id IS NOT NULL) AND
      (object_type = 'dialogue_topic') = (dialogue_topic_id IS NOT NULL) AND
      (object_type = 'storyboard_card') = (storyboard_card_id IS NOT NULL)
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_ewrite_object_links_storyboard_card
  ON ewrite_object_links(storyboard_card_id) WHERE object_type = 'storyboard_card';

COMMIT;
