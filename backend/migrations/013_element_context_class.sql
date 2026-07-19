ALTER TABLE elements
  ADD COLUMN IF NOT EXISTS context_class TEXT NOT NULL DEFAULT '';

UPDATE elements
SET context_class = CASE
  WHEN element_type = 'index_card' THEN 'card'
  WHEN slug = 'first-fire' THEN 'scenery'
  ELSE COALESCE(NULLIF(context_class, ''), '')
END
WHERE context_class IS NULL OR context_class = '' OR slug = 'first-fire' OR element_type = 'index_card';

ALTER TABLE elements
  ALTER COLUMN context_class SET DEFAULT '';
