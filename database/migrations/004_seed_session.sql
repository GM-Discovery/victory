INSERT INTO sessions (venue_id, status)
SELECT v.id, 'live'
FROM venues v
WHERE v.slug = 'the-cave'
  AND NOT EXISTS (
    SELECT 1
    FROM sessions s
    WHERE s.venue_id = v.id
      AND s.status IN ('rehearsal', 'live')
  );