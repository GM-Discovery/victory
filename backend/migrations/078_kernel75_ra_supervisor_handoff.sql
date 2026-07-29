BEGIN;

-- Ra's opening conversation ends at the supervisor handoff. The Narrator,
-- not Ra, introduces any later Crown Bet in a subsequent session.
UPDATE dialogue_topics
SET response_text = '"Because someone told me to find people like you." Ra glances toward the road, then back. "People with enough nerve to leave this place and see what comes next. My supervisor wants to meet you at the training camps outside town. I was told to send you there. That is the whole of my brief."',
    updated_at = NOW()
WHERE topic_key = 'why-looking'
  AND packet_id IN (SELECT id FROM dialogue_packets WHERE slug = 'ra');

-- Keep the old authored rows for history, but remove them from the Player
-- conversation so Ra cannot explain a later campaign quest prematurely.
UPDATE dialogue_topics
SET active = FALSE,
    required_for_completion = FALSE,
    updated_at = NOW()
WHERE topic_key IN ('crown-bet', 'what-if-succeed')
  AND packet_id IN (SELECT id FROM dialogue_packets WHERE slug = 'ra');

DELETE FROM dialogue_topic_prerequisites
WHERE topic_id IN (
  SELECT t.id
  FROM dialogue_topics t
  JOIN dialogue_packets p ON p.id = t.packet_id
  WHERE p.slug = 'ra' AND t.topic_key IN ('crown-bet', 'what-if-succeed')
)
OR requires_topic_id IN (
  SELECT t.id
  FROM dialogue_topics t
  JOIN dialogue_packets p ON p.id = t.packet_id
  WHERE p.slug = 'ra' AND t.topic_key = 'crown-bet'
);

COMMIT;
