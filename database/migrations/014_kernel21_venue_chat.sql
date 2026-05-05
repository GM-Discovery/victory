UPDATE venues
SET config = jsonb_set(
  jsonb_set(config, '{chat_enabled}', 'true'::jsonb, true),
  '{talking_enabled}',
  'true'::jsonb,
  true
)
WHERE slug = 'the-cave';
