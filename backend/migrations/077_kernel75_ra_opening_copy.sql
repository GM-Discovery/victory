BEGIN;

-- Ra's first beat should announce why he is there before the Player chooses
-- a topic. The authored topic responses remain in 066; this migration only
-- updates the opening copy for existing and fresh databases.
UPDATE dialogue_packets
SET opening_narration = 'Ra is looking for you. He steps into the courtyard behind you, already calling your name.',
    opening_line = 'There you are. I have been looking for you.',
    updated_at = NOW()
WHERE slug = 'ra';

COMMIT;
