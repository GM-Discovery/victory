BEGIN;

-- Keep the physical reveal, but do not make the Player click through four
-- separate door actions before reaching the tutorial completion handoff.
UPDATE dialogue_packets
SET closing_beats = jsonb_build_array(
  'Ra finds the concealed plate, turns the key, and the heavy bolt withdraws. The gate swings inward; beyond it, the road goes on.'
  ),
  updated_at = NOW()
WHERE slug = 'ra'
  AND content_origin = 'seed';

COMMIT;
