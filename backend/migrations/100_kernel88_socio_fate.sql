BEGIN;

-- Kernel 88: canonical live-play Fate Points. Fate is not new -- Chapter 2
-- of Character Creation already tracks a one-time creation-time FP balance
-- (backend/internal/characters/chapter2_stage.go, JSON-blob-backed inside
-- character_workbook_entries). This table is the *live-play continuation*
-- of that resource after creation ends: a single canonical balance a
-- Character carries through Shows, distinct from Chapter 2's own
-- bookkeeping, which is left untouched (see
-- characters/character_creation_fate_handoff.go for the one-time handoff at
-- creation completion).
--
-- Normal cap is 7. Character Creation may temporarily push the live balance
-- up to 12 while creation_mode is true; turning creation_mode back off is
-- the one point excess above 7 is clamped away (kernel-88 spec §4.3).
-- One row per Character, created lazily on first read/write, matching
-- kernel-85's character_socio_state convention.
CREATE TABLE IF NOT EXISTS character_socio_fate (
  character_card_id UUID PRIMARY KEY REFERENCES character_cards(id) ON DELETE CASCADE,
  balance INTEGER NOT NULL DEFAULT 0,
  creation_mode BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT character_socio_fate_balance_floor CHECK (balance >= 0),
  CONSTRAINT character_socio_fate_normal_cap CHECK (creation_mode OR balance <= 7),
  CONSTRAINT character_socio_fate_creation_cap CHECK (NOT creation_mode OR balance <= 12)
);

-- Fate must never be spent or awarded silently (kernel-88 spec §4.5): every
-- balance change is logged here with who did it and why, so award/spend/
-- clamp behavior is auditable and testable. Append-only, no updates/deletes.
-- show_id is nullable: Chapter 2 Character Creation (the source of the
-- one-time 'creation_handoff' entry) happens before a Character has ever
-- joined a live Show, so that entry has no Show to attribute to. Every
-- other reason is Show-scoped and always populates it.
CREATE TABLE IF NOT EXISTS character_socio_fate_ledger (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID REFERENCES shows(id) ON DELETE CASCADE,
  delta INTEGER NOT NULL,
  reason TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  balance_after INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT character_socio_fate_ledger_reason_check CHECK (reason IN (
    'player_spend', 'director_award', 'director_correction',
    'creation_guidance', 'debrief_guidance', 'creation_clamp', 'creation_handoff'
  ))
);

CREATE INDEX IF NOT EXISTS idx_character_socio_fate_ledger_card
  ON character_socio_fate_ledger(character_card_id, created_at DESC);

COMMIT;
