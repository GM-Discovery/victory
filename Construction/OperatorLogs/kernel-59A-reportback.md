# Kernel Report Back - Kernel 59A (Phase 1: Backend + Shared Projector)

## 1. Status
PARTIAL PASS — SCOPED TO BACKEND + SHARED PROJECTOR BY DESIGN

This is a deliberately scoped first phase of Kernel 59A, agreed with the owner after an audit showed the doc's "repair/completion" framing was inaccurate (see §2). This phase built the owner-curation domain layer, the persistence Kernel 58 never had, and a real shared projector consumed by both Greenroom and the venue right tray. It explicitly excludes the Greenroom "Arrange Face" UI, Director override/lock controls, and the mandatory browser-verification walkthrough — those remain for a follow-up phase and must not be claimed as done.

## 2. Audit: What Kernel 58 Actually Shipped

The Kernel 59A spec assumes Kernel 58 already built "Face fact contracts, priority, Token Aura, and Director locks." Before writing any code, I audited the actual codebase against that claim. Verified findings:

- `WorkbookPageField` (`workbook_pages.go`) already had `priority_mode`, `priority_score`, `priority_band`, `source_kind`, `value_locked`, `priority_locked` JSON fields, and a `priorityBandForScore()` helper.
- Every one of the ~30 call sites constructing these fields in `buildWorkbookPages` hardcoded `PriorityMode: "inferred"` with a priority score baked directly into Go source — there was no persistence and no manual mode anywhere.
- `value_locked`/`priority_locked` were declared but never set `true` anywhere in the codebase — completely inert.
- There was no `face_visibility_mode` concept (inferred/shown/hidden) anywhere, frontend or backend.
- There were no Director override/lock domain functions at all.
- Greenroom's frontend (`frontend/venues/greenroom/index.html`) read `priority_score` only to sort fields for display — no editing UI existed.
- Greenroom's Mechanics page and the venue right tray (Kernel 60) were two fully independent projections: Greenroom's Mechanics page never showed `character_skills` at all, while the venue sheet did.
- One real asset did exist: character-level Director authority. `CanEditCard` already grants edit access to any location Director/Producer via `hasAuthorityInLocation`, reusable as-is for a later override/lock kernel.

Conclusion reported to the owner: Kernel 59A as scoped was not a "finish the last 20%" repair — it was the entire Kernel 58 curation/lock system from scratch, plus a shared-projector refactor merging two independent systems, plus a new Greenroom UI, plus mandatory screenshot-backed browser verification. The owner chose to scope this pass to backend + shared projector only, deferring the rest.

## 3. What Was Built

- **Migration `034_kernel59a_character_face_overrides.sql`**: `character_face_overrides` table (`visibility_mode`, `priority_mode`, `priority_score`, plus `visibility_locked`/`priority_locked`/`value_locked` columns reserved — never set `true` this phase — for a later Director-lock kernel, matching Kernel 60's `is_helper` reservation precedent).
- **`backend/internal/characters/character_face_overrides.go`**: the owner-facing domain layer.
  - `SetCharacterFactFaceVisibility` / `ReturnCharacterFactFaceVisibilityToInferred` — explicit show/hide and reset, validated against the character's actual current Face-eligible fields (rejects unknown fact keys).
  - `SetCharacterFactPriority` / `ReturnCharacterFactPriorityToInferred` — manual signed-integer priority and reset.
  - Every mutation appends a typed, append-only History event (`character_face_visibility_set`, `character_face_visibility_returned_to_inferred`, `character_face_priority_set`, `character_face_priority_returned_to_inferred`) via the existing `RecordWorkbookEvents` — no event mutates or deletes prior history.
- **`backend/internal/characters/character_sheet_projection.go`**: the shared projector.
  - `applyFaceOverrides` — layers persisted overrides onto contract-inferred fields (owner-manual overrides inferred; no Director-lock precedence layer exists yet since that's deferred).
  - `EffectiveFaceFields` / `resolveEffectiveFaceFields` — the one function both Greenroom and the venue sheet call for "what's visible on Face right now, in what order."
  - `ProjectCharacterSheet` — groups effective fields into Identity / At a Glance (capped at 6, per spec §7.3) / Detail Sections, and attaches Mechanics data (attributes + Kernel 60 skills).
- **Rewired `LoadCharacterWorkbookView`** (Greenroom): the "face" page's fields now pass through `resolveEffectiveFaceFields` instead of being returned as raw contract output. The "mechanics" page now carries the character's actual `character_skills` list (`CharacterWorkbookPage.Skills`, new field) — fixing the real drift identified in the audit, where Greenroom never showed Kernel 60 skills at all.
- **Rewired `venue_sheet.go`** (Kernel 60): `BuildVenueCharacterSheet` now derives from `ProjectCharacterSheet` instead of independently reading `card.Tagline`/attributes/skills. Added `AtAGlance []VenueFactEntry` to the venue response (a trimmed `{key, label, value}` shape, stripped of Greenroom-only editing metadata like `Editable`/`InputType`) so the same visible narrative facts Greenroom shows now also appear in the venue right tray.

## 4. Evidence

Checks run:

- `cd backend && go build ./...` — clean.
- `cd backend && go vet ./...` — clean.
- `cd backend && go test ./internal/characters/... ./internal/commands/... ./internal/actions/...` — all pass, including the pre-existing Kernel 53/60 suite (confirms the rewire didn't regress existing Face-field consumers) and 22 new tests covering override guard clauses and the projector's pure logic (override application, visibility filtering, priority-sort ordering).
- `scripts/smoke/fresh-install.sh --local` — clean pass from an empty database, including migration `034` applying without error.
- **Live functional verification against a second isolated database** (separate from the smoke script's throwaway instance), calling the real Go functions directly against a seeded character with a confirmed Chapter 4 skill and Chapter 2 attributes:
  1. Baseline: Greenroom's face fields and the venue sheet's `at_a_glance` both showed the same visible facts (`tagline` present in both) at the same priority scores.
  2. `SetCharacterFactFaceVisibility(..., "tagline", "hidden")` removed `tagline` from **both** Greenroom's face fields and the venue's `at_a_glance` in the same call cycle.
  3. The character's Mechanics skill list (`Artifice`, step 1) was **unaffected** by hiding `tagline` — confirming §7.4 ("hidden means hidden from Face, not erased").
  4. `SetCharacterFactPriority(..., "pronouns", 9999)` correctly reordered Greenroom's face field list so `pronouns` sorted first.
  5. `ReturnCharacterFactFaceVisibilityToInferred(..., "tagline")` correctly restored `tagline` to both projections.
  6. All three expected History event types were recorded, append-only, with readable bodies (e.g. "Featured Quote was hidden from Face.").
  7. Confirmed `SetCharacterFactPriority` correctly rejects a fact key that isn't Face-eligible (`public_description`, which lives on the separate "bio" page) with `unknown_fact_key` — the eligibility guard works as designed.

All throwaway databases and scratch backend processes from this verification were dropped/killed afterward; the always-running production backend (a separate, pre-existing process) was never touched.

## 5. Deviations from Kernel

- **Scope**: per owner decision, this phase excludes the Greenroom "Arrange Face" UI (§6), Director override/lock controls (§6.6, §9.2, and the `Director*` domain functions in §10), the websocket `character/projection_updated` invalidation signal (§8.7), and the mandatory browser-verification walkthrough (§13). None of these were attempted; they are not partially built.
- **UrgentState is absent from `CharacterSheetProjection`**, unlike the doc's §8.1 shape. No fact contract in this codebase currently produces an urgent-eligible fact, so there is nothing to project; adding an always-empty field for a section nothing populates would be speculative structure with no consumer. Add it when a fact contract actually needs it.
- **At a Glance is `Region == "glance"` fields capped at 6**, not a genuine cross-region top-N re-ranking as §7.3 describes ("highest-priority eligible enduring facts" drawn from anywhere). Today's `Region: "glance"` tagging already functions as this bucket by convention across all existing fields, so this is a faithful reading of current data, but a true cross-region "top N regardless of section" algorithm was not built.
- **`CanEditCard` is the only authority gate** in this phase — it grants any owner or location Director/Producer edit rights, which is broader than "owner curation" alone. Since Director-specific override/lock behavior is deferred, this phase does not yet distinguish "a Director is curating on the owner's behalf" from "the owner is curating" — both currently go through the same `SetCharacterFactFaceVisibility`/`SetCharacterFactPriority` path with identical effect. A later Director-lock phase needs to add that distinction, not assume this phase already has it.

## 6. Known Issues / Explicitly Not Done

- No Greenroom UI exists to call any of the new mutation functions — they are backend-only and currently unreachable from the frontend. (Deliberately deferred, not a bug.)
- No HTTP endpoints were added for the new domain functions, for the same reason — building an API surface with no consumer would be dead code until the UI phase needs it.
- No Director override/lock functions exist (`DirectorOverrideCharacterFact` etc. from §10) — `visibility_locked`/`priority_locked`/`value_locked` are reserved columns only, always `false`.
- No browser was driven for this phase; all verification is Go-level (unit tests + direct function calls against an isolated database), consistent with what the owner asked for in this phase, but **not** sufficient for a future PASS claim on the full Kernel 59A spec, which explicitly requires screenshot-backed browser evidence.

## 7. Next Recommended Step

- Phase 2 (deferred): Greenroom "Arrange Face" UI — two-pane layout, per-fact Show/Hide/Priority controls, calling the domain functions this phase built.
- Phase 3 (deferred): Director override/lock functions (`DirectorOverrideCharacterFact`, lock/unlock triplet) built on top of the existing `CanEditCard`-derived Director authority, plus the UI to drive them.
- Phase 4 (deferred): the websocket `character/projection_updated` invalidation signal, and the full 20-step real-browser verification walkthrough with screenshots, per §13 — required before any PASS claim on the complete Kernel 59A spec.

---

# Kernel Report Back - Kernel 59A (Phase 2: Presentation Completion Gate)

## 1. Status
PARTIAL PASS — GREENROOM OWNER CURATION UI + DIRECTOR OVERRIDES/LOCKS + READABLE HISTORY IMPROVEMENT

This pass removes the incorrect Face event-pinning model, exposes the Phase 1 owner curation operations through Greenroom, implements dimension-specific Director lock/unlock controls, and adds Director value overrides as a projection layer. It does not complete projection invalidation, canonical recent-roll querying, or browser multi-client proof. Those remain required before the full Kernel 59A spec can be marked PASS.

## 2. What Changed

- Removed the Greenroom Face page's primary "Recent Events / Pinned Highlights / Pin" presentation model.
- Greenroom Face now renders a visible Face preview from current projected facts, while hidden facts are omitted from the preview.
- Added an "Arrange Face" inventory grouped by semantic buckets: Identity, Current State, Attributes, Skills, Traits, Relationships, Quotes, Biography, Custom Information, and Hidden from Face.
- Added owner controls for:
  - Show on Face;
  - Hide from Face;
  - Use inferred visibility;
  - priority decrement;
  - priority increment;
  - direct signed integer priority entry;
  - Use inferred priority.
- Added HTTP routes under `/api/character-workbooks/{id}/face-visibility` and `/api/character-workbooks/{id}/face-priority`, backed by the existing Phase 1 domain functions.
- Added `/api/character-workbooks/{id}/face-lock`, backed by `SetDirectorCharacterFactLock`, for independent `value`, `visibility`, and `priority` lock/unlock.
- Director lock changes require a reason, are restricted to producer/director location authority, and append `character_fact_lock_changed` History events.
- Added migration `035_kernel59a_director_value_overrides.sql` for `value_override_active` and `value_override`.
- Added `/api/character-workbooks/{id}/face-value`, backed by `SetDirectorCharacterFactValueOverride` and `ClearDirectorCharacterFactValueOverride`.
- Director value overrides require a reason, append `character_fact_director_override` / `character_fact_director_override_cleared`, and win in the shared projector without mutating the underlying character card fact.
- Greenroom shows a separate Director Locks control block for producers/directors, distinct from owner curation controls.
- Greenroom workbook responses now return all Face-eligible facts with override metadata, not only currently visible facts, so hidden facts remain available to restore.
- Venue projection remains trimmed through `EffectiveFaceFields`, so hidden facts are still omitted from venue Face.
- Greenroom Mechanics now renders the `page.skills` list already attached by the shared projector path.
- Greenroom History now renders readable event type labels and formatted timestamps.
- Chapter II History rows render known `BONUS_S##_X` values as named Socio choices with attribute effects, using stored payload when available, without mutating stored event rows.
- Locked-dimension errors now surface as explicit client errors instead of generic `character_kernel_failed`.
- First Theater and Catharsis venue trays now render the shared projector's `at_a_glance` facts in a compact Face section, so curated visible facts are not dropped on the floor.
- First Theater and Catharsis venue sheets refetch on window focus / visible-tab return with debounce, reducing stale tray data after another-tab Greenroom edits.
- First Theater and Catharsis Dice trays now render controls and Recent Rolls as a desktop split pane, with mobile stacking fallback.
- Dice tray recent-roll depth was increased from 2 to 8 in both venue runtimes.

## 3. Validation

Passed:

- `tmp=/tmp/greenroom-inline.js; sed -n '/<script>/,/<\/script>/p' frontend/venues/greenroom/index.html | sed '1d;$d' > "$tmp" && node --check "$tmp"`
- `git diff --check -- backend/internal/characters/character_face_overrides.go backend/internal/characters/character_face_overrides_test.go backend/internal/characters/character_sheet_projection.go backend/internal/characters/character_sheet_projection_test.go backend/internal/characters/workbook_pages.go frontend/venues/greenroom/index.html database/migrations/035_kernel59a_director_value_overrides.sql scripts/smoke/fresh-install.sh`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/network`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/commands`
- `cd backend && GOCACHE=/tmp/victory-gocache go build ./...`
- `cd backend && GOCACHE=/tmp/victory-gocache go vet ./...`
- `node --check frontend/venues/first-theater/runtime.js`
- `node --check frontend/venues/catharsis/runtime.js`
- `node --check frontend/venues/first-theater/runtime/dice.js`
- `node --check frontend/venues/catharsis/runtime/dice.js`
- `git diff --check -- frontend/venues/first-theater/index.html frontend/venues/first-theater/runtime.js frontend/venues/first-theater/runtime/dice.js frontend/venues/catharsis/index.html frontend/venues/catharsis/runtime.js frontend/venues/catharsis/runtime/dice.js`
- Existing database migration proof:
  - `docker exec -i victory-postgres psql -v ON_ERROR_STOP=1 -U victory -d victory < database/migrations/035_kernel59a_director_value_overrides.sql` — clean.
  - Verified `character_face_overrides.value_override_active` as `boolean NOT NULL DEFAULT false`.
  - Verified `character_face_overrides.value_override` as `text NOT NULL DEFAULT ''`.
  - Re-applied migration 035; Postgres emitted `already exists, skipping` notices and committed cleanly.

Notes:

- The generic inline-script extraction command is not valid for First Theater/Catharsis because each page contains multiple `<script>` tags; concatenating them leaves `</script>` markup in the temporary JS file. The changed runtime code lives in external scripts and was checked directly.

Full suite:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./...` did not pass because of failures outside this change path:
  - `backend/internal/assets`: `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails` fails with invalid UUID input for `/path/that/does/not/exist`.
  - `backend/internal/identity`: Discord audio/channel mapping/bootstrap repair tests fail around Discord response shape and repeated mapping counts.

## 4. Still Missing From Full Kernel 59A

- Browser proof of Director locks against real producer/director authority.
- Projection privacy audit for Director-only reason text beyond the existing venue-sheet omission path.
- Projection invalidation signal and stale-response version guard.
- Explicit venue tray Face/Mechanics tabs beyond the existing compact Face facts + Mechanics sections.
- Canonical recent-roll HTTP query; the Dice pane still derives recent rolls from the existing action snapshot/live action path.
- Browser and two-client verification.

## 5. Follow-up Pass: Quote/Bio Presentation Alignment

After owner review, one presentation mismatch remained: the Face preview still hard-coded Bio, Quote, and Vital Stats widgets even when the corresponding facts were hidden or not Face-eligible. That meant hiding `tagline` from Arrange Face did not remove the visible quote block.

Changes made:

- `public_description` is now a Face-eligible `glance` fact, so Biography can appear in the shared venue tray projection and can be shown/hidden/reprioritized like Featured Quote.
- `tagline` remains the single existing `/quote set`-controlled Featured Quote fact, now with higher default Face priority.
- Greenroom Face now renders Bio, Quotes, and Vital Stats only from currently visible Face facts. Hidden `tagline` no longer appears in the hard-coded quote widget.
- The quote widget supports rendering up to 1-3 quote-like Face facts by length, while today's backend still only stores the single Featured Quote. A true `/quote add` multi-quote collection does not exist yet.
- First Theater and Catharsis tray quote facts render as compact quote blocks instead of plain stat rows.

Additional validation:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters` — clean.
- `awk "f && /<\/script>/{f=0; next} f{print} /<script>/{f=1; next}" frontend/venues/greenroom/index.html > /tmp/greenroom-inline.js && node --check /tmp/greenroom-inline.js` — clean.
- `node --check frontend/venues/first-theater/runtime.js` — clean.
- `node --check frontend/venues/catharsis/runtime.js` — clean.
- `git diff --check -- backend/internal/characters/workbook_pages.go backend/internal/characters/workbook_pages_test.go frontend/venues/greenroom/index.html frontend/venues/first-theater/runtime.js frontend/venues/first-theater/index.html frontend/venues/catharsis/runtime.js frontend/venues/catharsis/index.html` — clean.
