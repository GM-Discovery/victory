BEGIN;

-- Kernel 85: Socio Game Status mechanical state -- eight attribute-based HP
-- pools (Might/Health, Intellect/Psyche, Grace/Motion, Presence/Will,
-- Spirit/Essence, Resolve/Focus, Awareness/Perception, Empathy/Heart) plus a
-- status-effect registry. Neither existed anywhere before this migration --
-- character_cards carries no attribute/HP fields (kernel-85 repo audit
-- S13.7). This is the one canonical table going forward; Game Status must
-- never write anywhere else (kernel-85 S14, "do not create duplicate
-- Character-health tables").
--
-- Deliberately scoped narrow per operator decision: paired current/maximum
-- integer columns with labels, no character-sheet UI wiring, no derived
-- fields. One row per Character, created lazily on first read/write rather
-- than backfilled for every existing character_cards row.

CREATE TABLE IF NOT EXISTS character_socio_state (
  character_card_id UUID PRIMARY KEY REFERENCES character_cards(id) ON DELETE CASCADE,

  health_current INTEGER NOT NULL DEFAULT 0,
  health_max INTEGER NOT NULL DEFAULT 0,
  psyche_current INTEGER NOT NULL DEFAULT 0,
  psyche_max INTEGER NOT NULL DEFAULT 0,
  motion_current INTEGER NOT NULL DEFAULT 0,
  motion_max INTEGER NOT NULL DEFAULT 0,
  will_current INTEGER NOT NULL DEFAULT 0,
  will_max INTEGER NOT NULL DEFAULT 0,
  essence_current INTEGER NOT NULL DEFAULT 0,
  essence_max INTEGER NOT NULL DEFAULT 0,
  focus_current INTEGER NOT NULL DEFAULT 0,
  focus_max INTEGER NOT NULL DEFAULT 0,
  perception_current INTEGER NOT NULL DEFAULT 0,
  perception_max INTEGER NOT NULL DEFAULT 0,
  heart_current INTEGER NOT NULL DEFAULT 0,
  heart_max INTEGER NOT NULL DEFAULT 0,

  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT character_socio_state_health_bounds CHECK (health_current >= 0 AND health_current <= health_max),
  CONSTRAINT character_socio_state_psyche_bounds CHECK (psyche_current >= 0 AND psyche_current <= psyche_max),
  CONSTRAINT character_socio_state_motion_bounds CHECK (motion_current >= 0 AND motion_current <= motion_max),
  CONSTRAINT character_socio_state_will_bounds CHECK (will_current >= 0 AND will_current <= will_max),
  CONSTRAINT character_socio_state_essence_bounds CHECK (essence_current >= 0 AND essence_current <= essence_max),
  CONSTRAINT character_socio_state_focus_bounds CHECK (focus_current >= 0 AND focus_current <= focus_max),
  CONSTRAINT character_socio_state_perception_bounds CHECK (perception_current >= 0 AND perception_current <= perception_max),
  CONSTRAINT character_socio_state_heart_bounds CHECK (heart_current >= 0 AND heart_current <= heart_max)
);

-- Status registry is deliberately data, not a Go const list (kernel-85
-- S2.3, "do not assume this list is exhaustive or final") -- adding a status
-- is a seed-row INSERT, not a code change. Seeded with the Socio source's
-- named statuses; a Director/Producer can still apply any status_key that
-- exists in this table, and this migration is additive-only for future
-- entries.
CREATE TABLE IF NOT EXISTS socio_statuses (
  key TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

INSERT INTO socio_statuses (key, label) VALUES
  ('raw', 'Raw'),
  ('winded', 'Winded'),
  ('wounded_1', 'Wounded 1'),
  ('wounded_2', 'Wounded 2'),
  ('wounded_3', 'Wounded 3'),
  ('limping', 'Limping'),
  ('burning', 'Burning'),
  ('haunted', 'Haunted'),
  ('numb', 'Numb'),
  ('obsessed', 'Obsessed'),
  ('fragmented', 'Fragmented'),
  ('overstimulated', 'Overstimulated'),
  ('unmoored', 'Unmoored'),
  ('inspired', 'Inspired'),
  ('shadowbound', 'Shadowbound'),
  ('sanctified', 'Sanctified'),
  ('radiant', 'Radiant')
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS character_socio_status_effects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  status_key TEXT NOT NULL REFERENCES socio_statuses(key) ON DELETE RESTRICT,
  intensity INTEGER,
  applied_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cleared_at TIMESTAMPTZ,
  cleared_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_character_socio_status_effects_active
  ON character_socio_status_effects(character_card_id)
  WHERE cleared_at IS NULL;

-- At most one active (uncleared) row per (character, status_key) -- applying
-- an already-active status again is "adjust intensity", not a second row.
CREATE UNIQUE INDEX IF NOT EXISTS idx_character_socio_status_effects_one_active
  ON character_socio_status_effects(character_card_id, status_key)
  WHERE cleared_at IS NULL;

COMMIT;
