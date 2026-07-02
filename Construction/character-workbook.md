# Character Workbook

## Canonical Root
- The canonical character root is `character_cards`.
- Kernel 53 extends that root instead of introducing a parallel workbook table.
- Workbook metadata now lives on the card row through `workbook_status` and `workbook_context`.

## Ownership And Active State
- A workbook belongs to its creator.
- The current active workbook for a user is tracked in `active_user_characters`.
- Catharsis creation now writes the new workbook into the active slot so the shell can project it immediately.

## Workbook Modules
- Each workbook can have module instances in `character_workbook_modules`.
- Kernel 53 seeds a Catharsis `socio` module with draft status and Stage 1 parentage metadata.
- Later productions can add their own module rows without changing the canonical root.

## History And Journal
- Append-only workbook facts now land in `character_workbook_entries`.
- Private player notes belong in `character_journals`.
- Journals do not become canonical history automatically.

## Greenroom
- Greenroom now renders workbook pages instead of the old card editor.
- The page set is `Face`, `History`, `Mechanics`, `Journal`, and the active module face page.
- Draft workbooks stay visible and resumable through the workbook view.

## Active Character Surface
- Catharsis and First Theater now show the active workbook in shell chrome.
- Clicking the shell chip opens Greenroom on the loaded workbook when one is active.

## Catharsis Draft Flow
- The Catharsis start button creates or resumes the onboarding draft.
- The draft is keyed by workbook context so repeated clicks do not duplicate the draft.
- The flow stays resumable instead of requiring a finished character.
- Stage 1 rolls now append workbook entries and persist progress through the workbook event route.
- Operator test accounts can now clear the Catharsis first-appearance browser flags from the account hub to replay the onboarding flow without changing production permissions.

## Implementation-Ready Next Slice
- Surface a real Socio ruleset selector when the draft flow asks for one.
- Move the staged parentage/build sequence onto a reusable ruleset model rather than hardcoding it into Catharsis UI.
- Keep the operator reset as a browser-only test utility until the ruleset model exists.

## Journal Command
- `/journal <text>` bypasses venue chat.
- The command writes to the private workbook journal for the active character.

## Deferred Work
- The Stage 1 Parentage chart is now supplied in `backend/internal/characters/parentage_chart.go`.
- Rolls 100-120 are normalized into the `organizational` band with the shared surrogate floor value `300000`.
- Stage 2 mechanics are still out of scope for this kernel; the Stage 2 panel is a shell only (`Stages of Childhood → Conception`) with no rollable controls. (Superseded by Kernel 54, below — Stage 2 is now the second of ten live Chapter 2 stages.)

## Kernel 54 — Chapter 2 Lifepath System
- Chapter 2 ("Lifepath") is 10 stages (Preconception, Genetics, Infancy, Toddlerhood, Childhood, School Age, Upper School, Teen/Middle, High School, Young Adult), each mapped to one of the 10 attributes (Spirit, Might, Empathy, Grace, Awareness, Intellect, Lore, Presence, Craft, Resolve). Rules data lives in `backend/internal/characters/chapter2_rules.go`, version `0.3`.
- Characters start with 12 Fate Points (FP); unspent FP carries into play. Each stage: roll a server-locked d4 → optional 3 FP enhancement (roll d6, keep `max(d4, d6)`) → the roll applies to the stage's primary attribute (Stage 9 is the exception, see below) → choose exactly 1 of 4 bonus choices (+1 to a named attribute) → optional companion purchase (1-2 FP) → 0-2 trait purchases (4-5 FP each, descriptive only) → complete the stage. Stage 2 additionally has an optional freeform Representation field with no mechanical effect.
- Stage 9 (High School) is special: 1 point of the final roll must go to Craft; the rest is freely distributed by the player across any non-Craft attributes (stacking on one attribute is allowed), all subject to the hard attribute cap.
- Hard attribute cap is 10 per attribute, enforced server-side by blocking the stage commit (not truncating); there is no separate Bonus Point cap.
- Dice are server-authoritative and idempotent (`POST /api/character-cards/chapter2-roll`, backed by the same `character_workbook_rolls` table Kernel 53 introduced) — repeated requests for the same stage/die never reroll. A companion `GET /api/character-cards/chapter2-roll` status endpoint lets the frontend restore an in-progress (rolled-but-not-yet-completed) stage's locked roll on page refresh.
- Stage completion is append-only and idempotent: `POST /api/character-cards/chapter2-stage` re-validates everything server-side (bonus choice, trait ownership, FP balance, attribute cap, Stage 9 allocation) and re-posting an already-completed stage returns the existing result rather than re-applying deltas.
- Chapter 2 completes after Stage 10, transitioning `workbook_context.current_stage` to `3` (Chapter 3 — out of scope for this kernel, shown as a mechanics-free interstitial shell).
- The venue (Catharsis) fixes the active ruleset; the player does not select a ruleset inside Chapter 2.

## Kernel 53 Correction Pass
- Starting Credit retention is strictly `> 50` (confirmed already correct; boundary tests for 49/50/51 exist in `socio_build_test.go`).
- The Stage 2 d4 "childhood die" rolling mechanic was removed from `onboarding.js`/`index.html`; only the interstitial shell remains.
- The donor `3d20` roll is now server-authoritative: `POST /api/character-cards/parentage-roll` (`backend/internal/characters/parentage_roll.go`) generates and locks the roll per `(owner, draft_token, event_key)` in `character_workbook_rolls`; `CreateCard` rebuilds `socio_parentage_parents` solely from these locked rolls before seeding, ignoring any client-supplied roll/credit fields.
- `character_workbook_entries` and `character_workbook_rolls` are now created by migration (`031_kernel53_character_workbook_foundation.sql`, renumbered from a colliding `018`); they previously existed on the live DB only via undocumented manual creation.
- `RecordWorkbookEvents` now dedupes identical (character, entry_type, title, body) entries to prevent history stacking on resume.
- `/journal` now supports PATCH (edit) and DELETE (archive), author-scoped; these were previously stubbed `501`.
- New characters default to the lowest unused Roman numeral name (`I`, `II`, ...); Greenroom and the active-character chip mark auto-named characters so it's visible the name hasn't been chosen yet.
- The Greenroom "New Workbook" dead end (it could never actually start a second character) is fixed via a `?new_character=1` signal that forces Catharsis to reopen the builder.

## Kernel 55 — Chapter 3: Character Archetypes
- Chapter 3 offers a 14-archetype catalog (`backend/internal/characters/chapter3_archetypes.go`, dataset version `1.0.0`, ported verbatim from the canonical reference implementation) via two paths: direct dropdown selection with an echo preview, or a full 15-question/89-answer reflective quiz (`frontend/venues/catharsis/chapter3-quiz-data.js`, client-only — never sent to or stored by the server).
- Quiz scoring: raw percentage per archetype, then a flat `+12` "Resonance" display bump (never labeled probability/confidence/accuracy); ranking is by raw percentage, then points, then archetype key alphabetically. Question 15 is unscored and only feeds a self-prediction "How You See Yourself" comparison.
- Either path ends at one explicit confirmation screen; only confirmation writes anything canonical. `POST /api/character-cards/chapter3-confirm` is idempotent, requires Chapter 2 Stage 10 complete, and stores a `Chapter3Fact` (stable archetype ID, key, title, code, primary/secondary attribute, key skill) in `workbook_context["chapter3"]`.
- History gets exactly one minimal event ("[Name] selected [Archetype] for their archetype.") with no quiz details; Mechanics gets the archetype's primary/secondary attribute and key skill (the stable inputs Kernel 56 consumes); Face gets a compact archetype widget (title + motto).
- Chapter 3 completion advances `current_stage` to `4`. A Director/permitted-actor `override=true` flag appends a new selection as `archetype_override` without destroying the original History event.

## Kernel 56 — Chapter 4: First Skill and Courtyard Entry
- Chapter 4 presents the ten skills belonging to the confirmed archetype's key-skill attribute group (routing decision: by the key skill's governing attribute, not the archetype's listed primary attribute — see the "Chapter 4 routing rule" note below), highlights the key skill, and lets the player confirm exactly one first trained skill at `d6`.
- The full versioned 100-skill catalogue (10 attributes × 10 skills, `backend/internal/characters/chapter4_skills.go`, catalogue version `0.9-draft`) was ported from `chapter-4-skill-catalogue-v0.9-draft.json` per the reconciliation policy in `socio-skill-catalogue-audit-v0.1.md` (Aesthetic Design added to Grace, Occultism excluded from Lore, Recovery retained in Resolve without an invented expanded description).
- **Chapter 4 routing rule**: the kernel spec required an explicit author decision between routing by the archetype's listed primary attribute vs. by its key skill's governing attribute, since 4 of 14 archetypes (Architect, Aesthetician, Steward, Performer) have these differ. Key-skill-attribute routing was selected, guaranteeing the key skill always appears in its own displayed group for all 14 archetypes with zero edits to Chapter 3's shipped data.
- Attribute capacity equals the character's effective Chapter 2 attribute total for that attribute (e.g. Craft 7 → up to 7 selected Craft skills); Kernel 56 only ever selects one skill, so capacity is always checked as 0-or-1-of-N.
- Confirming a skill (`POST /api/character-cards/chapter4-confirm`) is the terminal onboarding event: it locks the character (`workbook_status = "complete"`, `current_stage = 5`), equips the character as the user's active persona via the existing `setActiveCharacter` primitive, and is idempotent. Normal player flow cannot reopen Chapters 2/3/4 afterward — further changes require a Director `/override`, whose command surface is deferred to a later kernel.
- History gets one minimal event ("[Name] selected [Skill] as their first skill."); Mechanics gets the skill's stable ID, name, attribute, training state, and die size; Face gets the first trained skill name and die size.
- After confirmation, a brief curtain-opening transition leads into the initial Locked Courtyard scene (`GET /api/characters/chapter4-courtyard`) — a static, versioned scene descriptor (walls, market booths/crowd, Kessa present, door locked) with no invented dialogue or encounter mechanics, since no Courtyard/venue-scene infrastructure existed in the repo prior to this kernel and building a full venue/map/element system for one static entry point was out of this kernel's scope.
