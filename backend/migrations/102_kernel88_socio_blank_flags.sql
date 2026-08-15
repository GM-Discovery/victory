BEGIN;

-- Kernel 88: Director-authored short-text temporary state labels ("Holding
-- the eastern gate", "Waiting on Kessa"), distinct from socio_statuses'
-- canonical registry (kernel-88 spec §9). Rather than a parallel table,
-- reuse character_socio_status_effects (migration 097) with status_key
-- left NULL and a free-text custom_label instead -- the nullability split
-- is what keeps "canonical tag" and "blank state" visibly distinct, without
-- a second storage mechanism to keep in sync.
ALTER TABLE character_socio_status_effects
  ALTER COLUMN status_key DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS custom_label TEXT;

ALTER TABLE character_socio_status_effects
  DROP CONSTRAINT IF EXISTS character_socio_status_effects_kind_check;
ALTER TABLE character_socio_status_effects
  ADD CONSTRAINT character_socio_status_effects_kind_check CHECK (
    (status_key IS NOT NULL AND custom_label IS NULL)
    OR (status_key IS NULL AND custom_label IS NOT NULL AND char_length(custom_label) <= 60)
  );

COMMIT;
