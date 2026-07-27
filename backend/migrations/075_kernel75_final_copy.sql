BEGIN;

-- Kernel 75 S3.1 and S11.5: the operator-facing copy pass.
--
-- EVERY STATEMENT BELOW IS GUARDED. Migration 072 added content_origin, and
-- the rule this kernel commits to is:
--
--   a content migration touches a seeded field ONLY WHERE content_origin =
--   'seed'.
--
-- Once a Director edits a packet through the new authoring API its
-- content_origin flips to 'authored', and from then on no migration --
-- including this one, replayed on a fresh install after an edit -- will
-- overwrite their words. That is what makes shipping an editor and a content
-- migration in the same kernel safe.

-- --- Ra's gate-opening beats ------------------------------------------------
--
-- Migration 066 already wrote a good single-paragraph closing narration: Ra
-- lifts a concealed iron plate, exposes a recessed lock worn smooth with
-- use, and turns a key. S3.1 requires the same content as an ordered
-- sequence so the Program reveals it a beat at a time -- the Player should
-- watch the gate open, not read a paragraph about it having opened.
--
-- The four beats deliberately preserve migration 066's interpretation
-- exactly, because it is the one the operator settled on and because it is
-- what keeps the door's "no obvious lock or keyhole" opening description
-- honest rather than contradicted:
--
--   1. no lock was visible, because the plate reads as reinforcement
--   2. the mechanism is recessed and concealed, on the courtyard side
--   3. a real key, a real bolt -- physical, not magical
--   4. the gate visibly opens and the threshold beyond becomes visible
--
-- closing_narration is left as-is on purpose: it remains the fallback for
-- any client that has not learned about closing_beats.
UPDATE dialogue_packets SET
  closing_beats = jsonb_build_array(
    'Ra crouches at the foot of the gate and works his fingers beneath an iron plate you had taken for part of the door''s reinforcement.',
    'It lifts. Set into the stone behind it is a recessed lock, worn smooth with use — nothing you would have found by looking.',
    'He turns a key in it. Somewhere inside the wall a heavy bolt withdraws, and the sound of it carries.',
    'The gate swings inward under its own weight. Beyond the threshold, the road goes on.'
  ),
  updated_at = NOW()
WHERE slug = 'ra'
  AND content_origin = 'seed'
  AND closing_beats = '[]'::jsonb;

-- --- The tutorial-handoff Scene's index card --------------------------------
--
-- Migration 066 seeded placeholder wording ("The Narrator will take it from
-- here"), which S1.4 now specifically warns against: it implies an
-- unattended Director is about to appear. The replacement says plainly that
-- continuation happens with real people at a SCHEDULED Session.
--
-- scene_stage_elements has no content_origin column, so the guard here is
-- the exact placeholder string. A Director who has already re-authored this
-- card in Scene Setup keeps their version untouched.
UPDATE scene_stage_elements SET
  data = jsonb_set(
    data,
    '{text}',
    to_jsonb('The automated beginning is complete. Your Character and progress are saved. The story continues with real people during a scheduled Session.'::text),
    TRUE
  )
WHERE kind = 'index_card'
  AND data ->> 'text' = 'Tutorial complete. Your Character is equipped and ready. The Narrator will take it from here.'
  AND scene_id IN (SELECT id FROM scenes WHERE slug = 'tutorial-handoff');

COMMIT;
