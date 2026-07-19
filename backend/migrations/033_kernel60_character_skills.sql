BEGIN;

CREATE TABLE IF NOT EXISTS character_skills (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  skill_id TEXT NOT NULL,
  skill_name TEXT NOT NULL,
  attribute_name TEXT NOT NULL,
  ladder_step INT NOT NULL DEFAULT 0,
  is_helper BOOLEAN NOT NULL DEFAULT FALSE,
  source TEXT NOT NULL DEFAULT 'player_added',
  improvement_count INT NOT NULL DEFAULT 0,
  level_checkboxes JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (character_card_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_character_skills_card
  ON character_skills(character_card_id);

COMMIT;
