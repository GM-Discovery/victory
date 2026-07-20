-- Kernel 73: participant_interactions is Kessa's own table, deliberately NOT
-- folded into show_scene_placements.config_json -- Kernel 69's Scenes/
-- Placements already carry a config_json column, but Kernel 70's Cues chose
-- a dedicated first-class table (cues, migration 045) referencing
-- show_scene_placement_id rather than overloading it, and this follows that
-- same precedent for the same reason: a participant interaction has its own
-- authority/enablement/ordering lifecycle distinct from placement config.
--
-- The structural difference from a Cue: a Cue is shared/role-scoped (every
-- eligible viewer sees and can press the SAME button, authority.go's
-- trigger_scope). A participant interaction opens a Program for ONLY the
-- triggering Player -- interaction_type is the only thing that varies what
-- opens; Kernel 73 ships exactly one type, open_equip_mode.
CREATE TABLE IF NOT EXISTS participant_interactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_scene_placement_id UUID NOT NULL REFERENCES show_scene_placements(id) ON DELETE CASCADE,
  internal_name TEXT NOT NULL,
  stage_button_label TEXT NOT NULL,
  interaction_type TEXT NOT NULL,
  configuration_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT participant_interactions_type_check CHECK (interaction_type IN ('open_equip_mode'))
);

CREATE INDEX IF NOT EXISTS idx_participant_interactions_placement
  ON participant_interactions(show_scene_placement_id, sort_order);

-- Seed Kessa's packet + starting stock (spec S5.4: seeded stock proves the
-- alpha path immediately; the editor -- Phase E -- proves the model is
-- reusable). This does NOT create a participant_interactions row: that
-- requires a real show_scene_placement_id, which only exists once a
-- Director actually places the Courtyard Scene into a real Show (Kernel 69's
-- existing "Add Scene to Show" flow) -- attaching Kessa to that placement is
-- exactly what Kernel 73's own Director authoring UI (Phase E) is for.
INSERT INTO equipment_items (location_id, name, slug, short_description, descriptors_json, quantity_mode, active)
SELECT l.id, v.name, v.slug, v.description, v.descriptors::jsonb, v.mode, TRUE
FROM locations l
JOIN (VALUES
  ('Traveler''s Cloak', 'travelers-cloak', 'A well-worn cloak that keeps off the worst of the weather.', '["clothing","weatherproof"]', 'unique'),
  ('Coil of Rope', 'coil-of-rope', 'Fifty feet of sturdy hemp rope.', '["tool","utility"]', 'stackable'),
  ('Traveling Rations', 'traveling-rations', 'A few days'' worth of dried food.', '["consumable","food"]', 'stackable'),
  ('Sturdy Boots', 'sturdy-boots', 'Boots built for long roads.', '["clothing","footwear"]', 'unique'),
  ('Simple Dagger', 'simple-dagger', 'A plain but reliable blade.', '["weapon","light"]', 'unique')
) AS v(name, slug, description, descriptors, mode) ON TRUE
WHERE l.slug = 'amurray-family'
  AND NOT EXISTS (SELECT 1 FROM equipment_items e WHERE e.location_id = l.id AND e.slug = v.slug);

INSERT INTO merchant_packets (
  location_id, slug, display_name, intro_text, stance_dispositions_json,
  haggle_success_text, haggle_failure_text
)
SELECT
  l.id, 'kessa', 'Kessa',
  'Kessa looks up from her ledger as you approach her stall.',
  '{
    "command": {"disposition": "reject", "responses": ["Kessa''s eyes narrow. \"I don''t take orders in my own shop, friend.\""]},
    "convince": {"disposition": "reject", "responses": ["She waves a hand. \"I know my stock better than any stranger could explain it to me.\""]},
    "insight": {"disposition": "positive", "responses": ["Kessa brightens. \"Sharp eye. Here -- the roads north have been rough lately. Travel prepared.\""]},
    "follow": {"disposition": "neutral", "responses": ["\"Standard gear, standard prices,\" she says, gesturing at the stall. \"Take a look.\""]},
    "sympathize": {"disposition": "neutral", "responses": ["Kessa softens slightly. \"Kind of you to ask. The terms stay the same, but -- welcome.\""]}
  }'::jsonb,
  'Kessa grins. "I like your nerve. Twenty percent off, just for that."',
  'Kessa shakes her head. "Nice try. Full price stands."'
FROM locations l
WHERE l.slug = 'amurray-family'
  AND NOT EXISTS (SELECT 1 FROM merchant_packets m WHERE m.location_id = l.id AND m.slug = 'kessa');

INSERT INTO merchant_packet_equipment_items (packet_id, equipment_item_id, sort_order)
SELECT mp.id, ei.id, v.ord
FROM merchant_packets mp
JOIN locations l ON l.id = mp.location_id AND l.slug = 'amurray-family'
JOIN (VALUES
  ('travelers-cloak', 1), ('coil-of-rope', 2), ('traveling-rations', 3),
  ('sturdy-boots', 4), ('simple-dagger', 5)
) AS v(slug, ord) ON TRUE
JOIN equipment_items ei ON ei.location_id = l.id AND ei.slug = v.slug
WHERE mp.slug = 'kessa'
  AND NOT EXISTS (
    SELECT 1 FROM merchant_packet_equipment_items x
    WHERE x.packet_id = mp.id AND x.equipment_item_id = ei.id
  );
