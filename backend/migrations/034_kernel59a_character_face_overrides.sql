BEGIN;

CREATE TABLE IF NOT EXISTS character_face_overrides (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  fact_key TEXT NOT NULL,
  visibility_mode TEXT NOT NULL DEFAULT 'inferred',
  visibility_locked BOOLEAN NOT NULL DEFAULT FALSE,
  priority_mode TEXT NOT NULL DEFAULT 'inferred',
  priority_score INT,
  priority_locked BOOLEAN NOT NULL DEFAULT FALSE,
  value_locked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (character_card_id, fact_key)
);

CREATE INDEX IF NOT EXISTS idx_character_face_overrides_card
  ON character_face_overrides(character_card_id);

COMMIT;
