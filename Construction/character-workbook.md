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
