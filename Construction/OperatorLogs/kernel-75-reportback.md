# Kernel 75 Reportback — Tutorial Completion, Story So Far, Aftercare, MVP Proof

**Result:** PARTIAL — every automated criterion passes and the full journey is
proven live; PASS criteria 25 and 27 need the operator (placeholder art, and
the personal browser walk).

**Deployed:** live at `https://victory.amurray.family`.
**Migrations:** `069`–`076` (latest `076_kernel75_director_journal.sql`).
**Final commit SHA:** recorded in §13 below.

---

## 1. Pre-implementation audit

| Question | Answer |
|---|---|
| Next migration | `069`, confirmed against both `backend/migrations/` and the live ledger before naming. |
| K74 Ra closing path | `merchant.LeaveDialogue` did everything at once: narration, projection, two milestones. |
| Participant-local projection | `participant_local_projections`, opened only by `LeaveDialogue`, cleared by `projection.ClearForSharedSceneAdvance` inside both `SetCurrentScenePlacement` and `SetCurrentScenePlacementTrusted`. |
| Tutorial records | `participant_tutorial_progress`, CHECK-constrained to five milestone keys. |
| Archetype storage | **Not a column** — `character_cards.workbook_context->'chapter3'`. |
| Archetype→attribute map | **Already existed**: `characters.Chapter3Archetypes` (14 entries), `Chapter3ArchetypeByKey`. Reused, never copied. |
| Life-stage lines | `character_workbook_entries` where `page_key='history'`, `entry_type='chapter2_stage'`. |
| Kessa stance / Haggle | **Not persisted at all.** K73 wrote only ephemeral `actions` rows keyed to a user, not a Character. |
| `/journal` | `character_journals`, owner-of-active-character only, no `show_id`. |
| Trailer Face | A **Player** concept (`player_profile_*`), projected live, never stored. |
| Director's Chair | No Show/Scene awareness whatsoever; coarse global-role page guard. |
| Export utilities | **None existed** — no `encoding/csv`, no `Content-Disposition` anywhere. |
| Award/milestone system | **None existed** at Player level. `users` has no participation counter. |

**Reportback answers to §2:**

1. **Reused:** the archetype catalog, `participant_tutorial_progress` (extended, not forked), the projection machinery untouched, `program-panel.js`, the workbook page list, `showruns` authority helpers, `ResolveEligibleContext`.
2. **Story So Far needed a new table** — `character_workbook_entries` has no Show/Session/Scene columns, no privacy column, and `author_user_id NOT NULL` while these rows are system-generated.
3. **Reveal state** is `visibility_state ∈ private|table`. There is deliberately no `public`.
4. **Archetype → lens:** archetype's primary attribute wins; absent/unscored falls back to highest; ties prefer the archetype attribute; a missing Face Sheet line omits the quotation.
5. **Player vs Character recognition:** `player_recognition_grants` is UNIQUE on `(user_id, key)` alone; Character-level acknowledgement lives in `character_story_events`. That split is the whole answer.
6. **Aftercare for the Director's Chair:** Show Run → Show → roster row, driven by the roster so non-responders still appear.
7. **`/journal` participates** through a separate Director route over the same `character_journals` table (`source='director_journal'`, migration `076`), mirrored into Story So Far with the Director preserved as author. The Player's own journal handler is unchanged.
8. **Deliberately unimplemented:** listed at the end.

---

## 2. Ra closing sequence and the leave/continue split

`POST .../dialogue/leave` now records `ra_intro_completed` + **`tutorial_gate_opened`**, opens the projection, and returns `closing_beats`:

> Ra crouches at the foot of the gate and works his fingers beneath an iron plate you had taken for part of the door's reinforcement.
> It lifts. Set into the stone behind it is a recessed lock, worn smooth with use — nothing you would have found by looking.
> He turns a key in it. Somewhere inside the wall a heavy bolt withdraws, and the sound of it carries.
> The gate swings inward under its own weight. Beyond the threshold, the road goes on.

`POST .../tutorial/continue` is a **separate Player verb**: records `tutorial_completed`, generates Story So Far, grants recognition, returns the Program payload. `GET .../tutorial/completion` is the reopen path and writes nothing.

The projection still opens on *leave*, not on Continue — a Player who refreshes between the beats and pressing Continue must come back to the courtyard side of an open gate, not a stale Scene.

**Idempotency** rests on three independent database mechanisms, no read-then-write: the milestone UNIQUE, `uq_character_story_events_dedupe`, and the recognition UNIQUE. Deliberately not one transaction — each is independently idempotent, so a partial failure self-heals on the next press.

---

## 3. Deterministic reflection

`backend/internal/storysofar/` is a **leaf package importing nothing from this repo**, so `characters` can import it for the workbook page without a cycle. The canonical archetype rules are injected (`storyRules()` in `merchant`), never copied.

`Generate` is pure: no database, no clock, no randomness, **no map-iteration-order dependence**. That purity is a correctness requirement — the dedupe index can only absorb a retried Continue if the retry re-derives identical tuples. `generate_test.go` asserts byte-identical output across 200 runs; `facesheet_test.go` does the same for lens selection against shuffled maps.

Fourteen template cases (§5.3), each returning `(DraftEvent, bool)` where `false` **omits** the clause. That is how a pre-K75 Character with no attempt rows produces a complete, honest, thinner story rather than one full of blanks.

Live output for the proof Character (Observer, Empathy 8):

> K75 Player A Hero entered the Locked Courtyard as The Observer.
> K75 Player A Hero's strongest lens was Empathy, a pattern already visible in their history: "You learned early to notice who was left outside the group, and Empathy came with it."
> At Kessa's stall, K75 Player A Hero approached through insight…
> At the gate, K75 Player A Hero intended to: "I study the hinges and test whether the &lt;b&gt;door&lt;/b&gt; can be lifted instead of forced."
> Ra interrupted before the attempt resolved and introduced the Crown Bet. …stayed to ask more than they had to.
> Ra worked the concealed lock, the bolt withdrew, and the gate swung open.

The door intention is stored **verbatim**, markup and all, and escaped at every render surface.

---

## 4. Privacy and authority

- `visibility_state` defaults to `private`; the generator never writes anything else.
- Non-owners are filtered **in SQL** inside `ListForCharacter`, not by each caller trimming a payload — that per-call-site pattern produced K74's authority hole.
- The workbook page needed its own owner check because **`CanEditCard` grants any location authority-holder**, which would have shown a Director every private entry.
- The visibility PATCH is scoped to `owner_user_id` in the UPDATE's WHERE clause — **stricter than `CanEditCard`**, so a Director cannot publish a Player's reflection.
- Trailer Face isolation is structural: `playerprofile` reads only `player_profile_*` tables and has no path to `character_story_events`.
- Skip counts cannot be forged: one writer, cookie-derived identity, no DELETE route, and "consecutive" is **computed** from rows rather than stored.
- Review read is `CanViewBackstage`; **CSV export is `CanManageShowRun`** — reading the table is backstage visibility, exporting takes Player-written reflection off the platform.

---

## 5. Aftercare

Three locked qualitative prompts, every field optional, no numeric ratings anywhere. Drafts live in their own table so a reader of `aftercare_submissions` can never mistake unfinished thinking for something shared. Skips are append-only with **no counter column** — submitting resets the consecutive count by construction, and there is nothing to decrement.

The form states **"Your Director can read this"** above the first field. Aftercare is backstage-visible by design and copy implying otherwise would be the privacy bug.

Director's Chair is read-only *by construction*: no POST/PATCH/DELETE route exists for those paths. Its loader is fail-soft (`.catch(() => {})`) because that page's shared bootstrap calls `showForbidden()` on any throw, which would blank the Director Console.

**CSV** is the repo's first file download. Conventions set here: `.csv` path suffix, authority resolved and all rows buffered before any byte is written (so failures still return the JSON envelope), long format (one row per answer), and **spreadsheet-formula neutralisation** — a response beginning `=`, `+`, `-`, `@`, TAB or CR is prefixed with a quote, because these are participant-authored cells opened in Excel on a Director's machine.

---

## 6. Two defects the live proof caught

Both were found by the golden-journey script against real data, and neither would have been caught by the dbtests as originally written:

1. **Aftercare capability check matched nothing.** It joined `show_scene_placements` to `venues` on `venue_id`, but placements normally inherit their venue from the Scene and leave `venue_id` NULL. Fixed to `COALESCE(ssp.venue_id, sc.default_venue_id)`, the same rule `resolvePlacementVenueSlug` uses.
2. **The recognition beat never rendered for anyone.** `CompleteTutorial` granted correctly, then rebuilt its payload with `LoadTutorialCompletion` — a pure read that always reports `newly_granted=false`. The grant result is now carried across, and the dbtest assertion was strengthened from "either flag or grant" to a hard `newly_granted == true`.

A third, presentation-level defect was found in the evidence screenshots: a Player standing outside the gate was still being offered **"Speak with Kessa / Speak with Ra / Try the Door"**, because the shared Scene had not moved. Those buttons are now hidden while a local projection is active.

---

## 7. Test and proof output

- **Alpha gate: PASS** on every automated step, with exactly the tracked 9 `dice tray` failures and no drift in either direction. `dice.test.js` was not touched.
- **Backend:** all packages green, including new `storysofar` (30 pure tests), `aftercare`, `recognition`, and `merchant/kernel75_completion_dbtest_test.go`.
- **Frontend:** 29 new tests in `tests/stage-runtime/kernel75-completion.test.js`.
- **Golden journey:** `scripts/smoke/kernel75-golden-journey-browser.js` — **34 checks, all passing, on the live deployment.**

### The K74 blocker is fixed

K74's browser proof never completed a full run: it aborted whenever Catharsis held a Session for another Show. Three changes:

1. The refusal remains the **default** — the script still never silently ends a Session it did not open.
2. `K75_END_FOREIGN_SESSION=1` opts in explicitly and writes the ended Show's id, title and short code to the evidence directory so it can be restarted.
3. **The assertion that was actually missing:** after starting our own Session, assert the venue resolves to *our* Show before any Player read. Without it every downstream failure presented as an uninformative `no_active_session`.

The 23-day-old rehearsal Session on "Test Opening Socio" was cleared with the operator's explicit go-ahead (recorded in `evidence/kernel-75/ended-foreign-session.txt`).

**Evidence:** `Construction/OperatorLogs/evidence/kernel-75/` — 7 screenshots, `aftercare-export.csv`, `ended-foreign-session.txt`.

---

## 8. Video-readiness findings

1. **Art is the one open item.** `frontend/assets/ra.png` (3 KB) and `tutorial-handoff.png` (11 KB) are still generated placeholders. All *prose* was replaced this kernel. Swapping the art is a file drop requiring no code change — migration 066 points the backdrop at `/assets/tutorial-handoff.png` and the renderer reads it from the composition row. **This blocks PASS criterion 25.**
2. **The "Catharsis Welcome — This venue has four trays" modal covers the screen on a fresh account** and will appear in any recording of the first-time journey. Pre-existing first-appearance behaviour, not a K75 regression, but worth dismissing before recording (or resetting via the account hub's first-appearance reset).
3. Catharsis has been left with **no active Session**, so `/showtime` is clear for the walk.
4. Throwaway accounts from the proof runs (`k75_*`) are left in place per K62/63/65/74 precedent — no elevated privileges, but they are visible test records if the demonstration touches a roster list.

---

## 9. Known limitations (deliberate)

- **The Director journal route is deliberately separate from the Player's.** `POST /api/characters/{id}/director-journal` (migration `076`) writes to the same `character_journals` table with `source='director_journal'`, gated by `CanManageShowRun` at the Character's Location, and mirrors into Story So Far preserving the Director as author. The Player's own `/api/character-journals` handler keeps its stricter "your active Character only" rule untouched — widening it would have been the wrong extension. There is **no** route by which a Director can edit or archive a Player's own entry; `UpdateCharacterJournal` and `ArchiveCharacterJournal` remain author-scoped. **No Director-facing UI ships this kernel** — the route and its provenance guarantees are proven by test, but a Stage Management control for it is post-MVP.
- The dialogue editor edits **text and ordering only** — it cannot create or delete topics, edit prerequisites, or change the destination Scene, because those decide whether a Player can reach the end of a conversation.
- No numeric economy, points, levels, or badge catalog. One recognition key, CHECK-constrained; a second is a migration on purpose.
- `participantinteractions` was **not** extracted from `merchant`. K74 flagged it as worthwhile at a fourth interaction *type*; this kernel added a verb. Touching every call site on a deploy-live kernel was the wrong trade. Still open.
- The dice-test debt was **not** touched — orthogonal risk on a live deploy, and closing it requires changing `alpha-gate.sh` and `roadmap.md` together.

---

## 10. Files changed

57 files, +9807/−48. Highlights:

- **Migrations:** `069`–`075`.
- **New packages:** `backend/internal/storysofar/`, `backend/internal/aftercare/`, `backend/internal/recognition/`.
- **New backend files:** `merchant/kernel75_{completion,http,attempts,participation,aftercare_http,aftercare_review_http,dialogue_authoring_http}.go`, `dialogue/authoring.go`, `world/kernel75_tutorial_state.go`.
- **Modified:** `merchant/{tutorial_flow,interactions}.go`, `tutorial/progress.go`, `dialogue/{types,packets,state}.go`, `world/snapshot.go`, `characters/workbook_pages.go`, `access/kernel16_venue_bootstrap.go`, `cmd/victory/main.go`.
- **Frontend:** `stage-runtime/{participant-interactions,runtime}.js`, `venues/{greenroom,directors-chair,show-runs}/*.html`.
- **Tests/proof:** `tests/stage-runtime/kernel75-completion.test.js`, `scripts/smoke/kernel75-golden-journey-browser.js`.

---

## 11. Operator checklist (§13.3)

Run against live at `https://victory.amurray.family`. Catharsis is free; you will need a fresh account.

1. Ra finishes unlocking and opening the gate, one beat at a time.
2. Continue — not "Step Through" — opens the completion Program.
3. The reflection matches your Character's real data.
4. No unsupported claims appear.
5. Story So Far is private (check the Greenroom "Story So Far" tab).
6. Aftercare partial save works.
7. Closing and reopening preserves the draft.
8. The Skip warning reflects prior skips.
9. The Director's Chair receives the submission read-only.
10. The My People link works.
11. You can close the completion panel and explore Catharsis.
12. The waiting message explains scheduled human continuation.
13. A shared Scene advance clears the local projection.
14. The complete fresh-account journey works.
15. The recording environment is clean — **see §8 first**.

---

## 12. Recommended post-MVP priorities (not started)

1. A Stage Management control for the Director journal route (the route ships; the UI does not).
2. Real art for Ra and the handoff Scene.
3. Extracting `participantinteractions` from `merchant` at the next interaction type.
4. Closing the dice-test debt as its own commit.
5. A first-appearance suppression flag for recording sessions.

---

## 13. Final commit and deployment

- **Implementation:** `2586482` (kernel body) and `73d1334`'s successor (Director journal + this reportback).
  The reportback commit is the branch tip; run `git log --oneline -2` for the exact pair.

- **Deployed:** that commit is what is running live.
- **Ledger:** `076_kernel75_director_journal.sql` is the latest applied migration.
- **Catharsis:** left with no active Session, so `/showtime` is clear for the walk.
