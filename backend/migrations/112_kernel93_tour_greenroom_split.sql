BEGIN;

-- Kernel 93 Pass C: campus_continuation bundled the Catharsis pin and the
-- Greenroom pin as one atomic tour, eligible the instant campus_mandatory
-- finished -- with no check that the user could actually enter Catharsis
-- yet, and no check that character creation had even started. Both pins
-- fired back-to-back regardless, pointing a user at places/features they
-- couldn't act on. Splitting the Greenroom pin into its own tour key lets
-- eligibility.go gate each pin independently: campus_continuation now
-- requires actual Catharsis access, and the new greenroom_intro requires an
-- in-progress character_cards row at Catharsis.
ALTER TABLE tour_completions DROP CONSTRAINT tour_completions_tour_key_check;
ALTER TABLE tour_completions ADD CONSTRAINT tour_completions_tour_key_check CHECK (tour_key IN (
  'campus_mandatory',
  'campus_continuation',
  'greenroom_intro',
  'catharsis_cast',
  'directors_chair_director_toolbox'
));

ALTER TABLE tour_progress DROP CONSTRAINT tour_progress_tour_key_check;
ALTER TABLE tour_progress ADD CONSTRAINT tour_progress_tour_key_check CHECK (tour_key IN (
  'campus_mandatory',
  'campus_continuation',
  'greenroom_intro',
  'catharsis_cast',
  'directors_chair_director_toolbox'
));

COMMIT;
