-- Kernel 79 Goal C: extend the reusable ewrite_object_links model (spec 8.4,
-- "prefer extending the existing ewrite_object_links model; do not create
-- separate unrelated link systems for each object type") to four more
-- existing Victory objects, exactly the same idiom migration 084 set up for
-- equipment_item: one nullable FK column per object type, a CHECK arm, and
-- a matching fk-shape CHECK so exactly one typed column is set per row.
--
-- object_type coverage added this migration:
--   - 'cue'            -> cues(id)                    (spec 8.2)
--   - 'index_card'      -> elements(id)                (spec 8.1; index
--                          cards are element_type='index_card' rows, no
--                          separate index_cards table exists)
--   - 'scene_element'   -> scene_stage_elements(id)     (spec 8.3)
--   - 'dialogue_topic'  -> dialogue_topics(id)          (spec 7; the one
--                          concrete, stable "tutorial choice" object that
--                          exists today -- Kernel 74's guided-dialogue
--                          topics. Health systems/stances/actions named in
--                          spec 7 have no data model yet, same deferral
--                          Kernel 79 Phase 1 already recorded.)
--
-- Every new FK is ON DELETE CASCADE, matching equipment_item_id's choice:
-- an object link is a satellite row of the object it annotates, meaningless
-- once that object is gone.

ALTER TABLE ewrite_object_links
  ADD COLUMN IF NOT EXISTS cue_id UUID REFERENCES cues(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS index_card_element_id UUID REFERENCES elements(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS scene_element_id UUID REFERENCES scene_stage_elements(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS dialogue_topic_id UUID REFERENCES dialogue_topics(id) ON DELETE CASCADE;

ALTER TABLE ewrite_object_links
  DROP CONSTRAINT IF EXISTS ewrite_object_links_type_check;
ALTER TABLE ewrite_object_links
  ADD CONSTRAINT ewrite_object_links_type_check
    CHECK (object_type IN ('equipment_item', 'cue', 'index_card', 'scene_element', 'dialogue_topic'));

ALTER TABLE ewrite_object_links
  DROP CONSTRAINT IF EXISTS ewrite_object_links_fk_shape;
ALTER TABLE ewrite_object_links
  ADD CONSTRAINT ewrite_object_links_fk_shape
    CHECK (
      (object_type = 'equipment_item') = (equipment_item_id IS NOT NULL) AND
      (object_type = 'cue') = (cue_id IS NOT NULL) AND
      (object_type = 'index_card') = (index_card_element_id IS NOT NULL) AND
      (object_type = 'scene_element') = (scene_element_id IS NOT NULL) AND
      (object_type = 'dialogue_topic') = (dialogue_topic_id IS NOT NULL)
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_ewrite_object_links_cue
  ON ewrite_object_links(cue_id) WHERE object_type = 'cue';
CREATE UNIQUE INDEX IF NOT EXISTS uq_ewrite_object_links_index_card
  ON ewrite_object_links(index_card_element_id) WHERE object_type = 'index_card';
CREATE UNIQUE INDEX IF NOT EXISTS uq_ewrite_object_links_scene_element
  ON ewrite_object_links(scene_element_id) WHERE object_type = 'scene_element';
CREATE UNIQUE INDEX IF NOT EXISTS uq_ewrite_object_links_dialogue_topic
  ON ewrite_object_links(dialogue_topic_id) WHERE object_type = 'dialogue_topic';
