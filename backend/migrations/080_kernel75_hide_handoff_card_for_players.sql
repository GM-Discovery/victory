BEGIN;

-- The handoff scene already has a Player-facing completion panel. Keep the
-- authored index card available to backstage editors, but do not project it
-- as a floating card over the handoff artwork for Players.
UPDATE scene_stage_elements
SET visibility = '{"toRoles":["director","producer","crew","operator"],"privateTo":[]}'::jsonb
WHERE kind = 'index_card'
  AND label = 'Tutorial complete'
  AND scene_id IN (SELECT id FROM scenes WHERE slug = 'tutorial-handoff');

COMMIT;
