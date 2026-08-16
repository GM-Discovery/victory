Kernel Report Back — Kernel 88: Socio Guided Play Surface

## 1. Status

**PARTIAL.** The canonical backend state (Fate, Stance, blank state flags, the Interrupt/Help pending-action stack), the Owner/Director tiered security boundary, Current Turn wiring, and the Director/Player frontend surfaces are all built, backend-tested (18 new Go tests, all passing against a real Postgres database, plus the pre-existing Kernel 85 tests in the same package still passing), and re-verified live in a real browser with a Director and a Player connected simultaneously to the same Show (25/25 assertions passing, screenshots below). That part is genuinely solid.

It is PARTIAL, not PASS, for four honest reasons:

1. **The player-initiated canonical roll path (`roll/dice_own_mechanic`) was never exercised — not by a Go test, not in the browser proof.** This is the single largest gap. The Interrupt/Help stack was proven end-to-end by resolving interrupts with a manually-supplied `roll_total` via the direct HTTP endpoint, not by actually having Alice roll her own skill through the WS path (`actions.StorePlayerMechanicRoll`, `canActPlayerMechanicRoll`). The code compiles, follows the same server-side-identity-resolution pattern as Kernel 87's IC chat, and mirrors `StoreDiceRoll`'s storage/audience/stage-effect delivery closely enough that I believe it's correct — but "I believe it's correct" is not the evidence standard this project holds itself to, and I did not meet it here.
2. **Director's health-pool bars are not a click-to-reveal-numeric-field widget.** The spec (§6.1, §33) asks for a clean bar graph that opens an editable number field on click and collapses back on save. What actually ships is Kernel 85's original always-visible number inputs, unchanged — functionally complete (Director can see and edit exact values) but not the visual-polish treatment specified.
3. **Story So Far integration is half-wired.** `generateFateLines`/`generateHelpLines` and the new `Inputs.FateAwards`/`Inputs.HelpResolutions` fields exist and are additive per the kernel's own "leaf package, deterministic, closed vocabulary" constraints — but I did not find and wire the caller that assembles `Inputs` from `character_socio_fate_ledger`/`socio_pending_actions` into the existing Continue/Generate pipeline. Until that caller exists, no Fate-award or Help-resolution line will actually appear in a real Character's Story So Far, despite the template code being ready for it.
4. **Two smaller UI gaps**: no Catharsis Character-Creation-time UI copy suggesting 1–2 FP, and no Debrief/Aftercare-adjacent UI copy suggesting 2–4 FP (the backend accepts `reason: "creation_guidance"`/`"debrief_guidance"` on the award endpoint; nothing in the frontend surfaces the suggestion yet). The Chapter 2→Fate handoff function (`HandoffCreationFateToSocio`) is wired into the real completion path but has no dedicated test and was not exercised by the browser proof (the fixture characters were created directly, not walked through Chapter 2).

Everything genuinely proven is listed with evidence below; everything not proven is listed as a known gap, not folded into a claimed PASS.

## 2. What Was Built

**Backend** (`backend/internal/socio/`, extending the Kernel 85 package):
- `fate.go` — `character_socio_fate`(+`_ledger`) tables (migration `100`), `GetFate`/`SpendFate`(owner-or-Director, decrement-only)/`AwardFate`(Director-only, any signed delta, clamps to 7/12)/`SetCreationMode`(the one clamp-to-7-on-completion point, logged)/`ListFateLedger`
- `stance.go` — `socio_stances`(seeded Insight/Command/Convince/Sympathize/Follow)/`character_socio_stance` (migration `101`), owner-or-Director `SetStance`
- `statuses.go` extended — blank Director-authored state labels via a nullable `custom_label` column on the existing `character_socio_status_effects` table (migration `102`) rather than a second table; `AddFlag`/`ClearFlag`
- `interrupts.go` — `socio_pending_actions` (migration `103`), `OpenPrimaryAction` (Player-declared, gated on Current Turn), `OpenInterrupt` (Director-adjudicated), `ResolveInterrupt` (overage on success, zero on failure), `CancelPendingAction` (cascades to open children), `ConsumeOverageFor` (idempotent, `overage_consumed_at`-gated)
- `projection.go` — `ProjectSocioState`, the one function every non-Director read path uses; two tiers only (`TierOwner`, `TierDirector`) — no Audience tier, since Audience never queries Socio state at all (see §7)
- `coordination.go` — thin adapter over Kernel 83's existing `venuecoordination.Registry` for Current Turn, keyed `showID:cohortID`
- `narration.go` — `QualitativeLabel`, a pure ratio-threshold table (steady/strained/critical/broken), no LLM
- `http.go` extended with 11 new handlers; all wired in `main.go`
- `actions/player_roll.go` + `authority.go` extension — `canActPlayerMechanicRoll` (role `cast` only), `StorePlayerMechanicRoll` (no `character_id` field in the request struct at all, mirroring `ic_chat.go`'s `resolveSpeakerCharacter`; skill ownership re-verified server-side; dice expression from the skill's own `StepExpression`, never invented)
- `network/ws.go` — new `roll/dice_own_mechanic` case, mirroring `roll/dice`'s action-broadcast + stage-effect delivery exactly, so a Player's own roll plays out on stage the same way a Director's does
- `characters/character_creation_fate_handoff.go` — one-time Chapter-2-completion→canonical-Fate seed (duplicates a few lines of SQL rather than importing `socio`, to avoid a real import cycle: `socio → cohorts → network → actions → characters`)
- `storysofar/templates.go`/`types.go` — new `EventFatePointsAwarded`/`EventHelpResolved` consts, `generateFateLines`/`generateHelpLines`, migration `104` widening the two CHECK constraints

**Frontend**:
- `kernel85-cohort-tools.js` extended (not forked) — Fate award/Stance override/blank-flag controls on the existing Game Status card, plus two new panels: Current Turn and the Help/Interrupt stack (indentation-based nesting, LIFO by construction)
- `kernel88-socio-player-hud.js` (new) — Face (name/portrait via the existing venue-sheet endpoint), Fate + spend control, the Stance wheel (conic-gradient, active slice marked by fill+border, never color alone), qualitative pool strip, blank-flag chips, mechanic-roll buttons sourced directly from `CharacterSheetProjection.Skills` (the existing compiler output) — no new mechanic-surfacing endpoint was built
- `runtime.js` — `window.VictoryStageKernel88Bridge` (userID, session ID, generic `sendAction`, and `getMyCharacterID`) added alongside the existing Kernel 85 bridge

**Deliberately not built** (see full reasoning in the conversation this reportback closes out): an LLM-backed narrator/rules-assistant (product decision: deterministic only), a Socio-specific Audience visibility tier (Audience already has the Kernel 81/86 Cave/stage view; this kernel does not touch it), and a Face-*switcher* UI (already exists in Greenroom — `character-select` dropdown + "Make Active" button → `POST /api/character-cards/{id}/activate` → `ActivateCharacterCard`, which since Kernel 74 also write-throughs into `show_run_roster_members.character_card_id` via `syncRosterCharacterSelection`, so it already keeps both identity concepts in sync with no new code needed).

## 3. Evidence (MANDATORY)

### Backend tests (real Postgres, `TEST_DATABASE_URL` set to `victory_test`)

```
$ export TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable"
$ cd backend && go test ./internal/socio/... -v
```

18 new tests, all PASS: Fate cap clamp (award beyond 7 clamps to 7), creation-mode temporary ceiling of 12, creation-mode-off clamps excess and logs a `creation_clamp` ledger entry, owner self-spend (decrement-only, rejects over-spend, rejects non-positive amounts), spend rejected for a non-owner/non-Director, award rejected for a non-Director, the 5-stance registry, Stance set by owner-or-Director (rejects outsider, rejects invalid key), blank flags distinct from canonical statuses (mixed list, correct `is_blank` split, clears independently), label-length rejection, Owner-tier projection (qualitative pools, exact own Fate, no `exact_pools` field at all), Director-tier projection (exact pools), projection rejects a non-owner/non-Director outright, Current Turn assignment requires Cohort/Ungrouped membership, the current holder may pass the turn on without Director authority, `OpenPrimaryAction` requires the actor to actually hold Current Turn (Director may still open on a Player's behalf), interrupt resolution computes overage on success and zero on failure, interrupt-of-interrupt nests correctly and a parent cannot resolve while a child is still open.

Plus the 7 pre-existing Kernel 85 tests in the same package, still passing (no regressions from the `character_socio_status_effects`/`ActiveStatus` schema changes).

**Full existing suite**: `go test ./...` — all green except two pre-existing, unrelated failures in `internal/shows`/`internal/showtime` caused by a stale, abandoned Catharsis session ("K87v2 Show") left in the shared `victory_test` database by an earlier, separate debugging session — confirmed unrelated by re-running those two packages in isolation and reading the failure (`venue_busy`/`no_active_session_for_show`, both about that specific stale row, not about anything this kernel touched). Not fixed as part of this kernel's test suite (out of scope); it *was* closed via direct SQL (`UPDATE sessions SET status='closed', ended_at=NOW()`) specifically to unblock the browser proof below, which needed a real active session on that venue.

`go build ./...` and `go vet ./...`: clean.

### Browser proof (LOCAL DEV STACK — never production)

Real compiled backend (`go build ./cmd/victory`) run locally against the disposable `victory_test` Postgres database, fronted by the same same-origin static+proxy dev server Kernel 87 built (`scripts/smoke/kernel87-local-dev-proxy.js`, reused as-is). `PASSWORD_SIGNUP_ENABLED` was not needed this time — fixture accounts were created directly via a throwaway Go program (`backend/cmd/k88fixture`, deleted after the run — see §5) that reused the same internal packages (`showruns.CreateShowRun`, `shows.CreateShow`, `tickets`, `showruns.SelectCharacter`) the Go test fixtures already use, so the fixture shape matches what the real app produces rather than hand-rolled SQL guessing at columns.

```
=== Starting the Show Session on Catharsis (singleton-session venue) ===
PASS show session starts cleanly

=== Fate: normal cap and Director award ===
PASS director award clamps to normal cap 7
PASS player spends own fate
PASS over-spend is rejected

=== Stance: Player sets own, Director overrides ===
PASS player sets own stance

=== Health pools: Director sets exact, Player sees only qualitative ===
PASS director sets exact health pool
PASS player's own view is qualitative-only for health, got {"condition":"critical"}
PASS player projection carries no exact_pools field at all
PASS director's view carries exact health 3/10

=== Blank state flags: distinct from canonical statuses ===
PASS director adds a blank state flag

=== Director opens Game Status panel (Fate/Stance/Flags controls visible) ===
PASS Director Game Status panel opens
PASS Director panel shows Alice's Fate balance 5

=== Director opens Current Turn panel ===
PASS Director Current Turn panel opens

=== Player HUD renders: Fate/Stance wheel/qualitative condition visible ===
PASS Player HUD is visible
PASS Player HUD shows own Fate balance 5
PASS Player HUD stance wheel renders 5 stance slices

=== Current Turn -> primary action -> interrupt -> resolve -> overage ===
PASS director gives alice current turn
PASS alice opens her own primary action on her turn
PASS director opens an interrupt
PASS interrupt resolves with overage 4 (9 - target 5)

=== Failed-help produces zero overage ===
PASS failed help adds no overage

=== Director opens Help Stack panel and sees the nested actions ===
PASS Director Help Stack panel opens

ALL PASS (25/25)
```

Screenshots in `Construction/OperatorLogs/evidence/kernel-88/`:
- `01-director-stage.png` / `02-player-stage.png` — both accounts connected to the same live Show
- `03-director-game-status.png` — the extended Game Status card: exact Health 3/10, Fate 5 with an award control, Stance dropdown showing "Command", the "Waiting on Kessa" blank-flag chip with its own remove button, the canonical-status apply dropdown, and the new toolbar row (Cohorts/Game Status/Current Turn/Help Stack)
- `04-director-current-turn.png` — the Current Turn panel
- `05-player-hud.png` — the Player HUD: Fate 5, the 5-slice Stance wheel with "Command" unmistakably active (filled white dot + border, not color alone), all 8 pools in qualitative form ("Health: critical", the rest "unknown" since unset), the blank-flag chip, Spend Fate control
- `06-director-help-stack.png` — the Help/Interrupt stack panel

## 4. How to Run (Operator Steps)

```bash
# 1. Test database (idempotent)
export TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable"
scripts/test/setup-test-database.sh

# 2. Go tests
cd backend && go test ./internal/socio/... -v

# 3. Local backend + dev proxy (never the production containers)
go build -o /tmp/k88-backend ./cmd/victory
PASSWORD_SIGNUP_ENABLED=true PORT=8092 DATABASE_URL="$TEST_DATABASE_URL" \
  STORAGE_ROOT=/tmp/k88-storage EXPORTS_ROOT=/tmp/k88-exports BACKUP_DIR=/tmp/k88-backups \
  /tmp/k88-backend &
node ../scripts/smoke/kernel87-local-dev-proxy.js "$(pwd)/../frontend" http://127.0.0.1:8092 8090 &

# 4. A Show needs an active session before anyone can connect (singleton-session
#    venue) -- start one for whatever Show your fixture uses:
curl -X POST "http://127.0.0.1:8090/api/shows/{show_id}/sessions/start" \
  -H "Content-Type: application/json" -H "Cookie: victory_session={director_cookie}" \
  -d '{"venue_slug":"catharsis"}'
```

The fixture-building program (`backend/cmd/k88fixture`) was deleted after this run per its own header comment ("throwaway, deleted after the proof runs") — reconstructing it from this reportback's §2/§5 description plus `backend/internal/socio/fixtures_test.go`'s existing `showFixture`/`playerFixture` helpers should take a few minutes if a future pass needs the same setup again.

## 5. Operator Notes (CRITICAL)

- **Headless Chromium GPU-stalls into a fully unresponsive main thread on this host's Pixi/WebGL canvas** — not just failed screenshots, but `page.evaluate()` itself (a plain `fetch()` call) hung indefinitely. Root-caused via a minimal repro (`page.evaluate` racing an 8s timeout) and fixed by launching Chromium with `--disable-gpu --disable-webgl --disable-software-rasterizer`. This is worth remembering for any future Playwright work against Catharsis on this host — the existing Kernel 85/87 scripts' "wait for `#stage-load-tools`, click it" dance was written assuming WebGL fails to init cleanly in headless mode; on this host it was instead succeeding partway and then stalling, which is a strictly worse failure mode than the one those scripts were defending against.
- **A session-token cookie insert bug cost real debugging time**: my first fixture program hex-encoded the SHA-256 session-token hash and inserted it as a *string* into a `bytea` column — Postgres silently stored the ASCII bytes of the hex string rather than the raw hash, so every request with that cookie came back `not_authenticated` with no server-side error to point at the cause. Fixed by passing the raw `[]byte` hash directly to the query, matching `sessions.HashToken`'s actual comparison. Worth documenting since it's an easy mistake to repeat: **never hex/base64-encode a hash before handing it to pgx for a `bytea` column** — pass the raw bytes.
- **A stale, abandoned session blocked the fixture Show from becoming active** — Catharsis is a singleton-session venue (Kernel 70A), and a session tied to "K87v2 Show" (started 2026-08-12, apparently from the other debugging session sharing this test database) was still `status='rehearsal'`/`ended_at IS NULL`, so `POST /api/shows/{id}/sessions/start` correctly refused with `venue_busy_with_other_show` for my fixture's Show. This is also what caused Alice's browser tab to connect to the *wrong* Show entirely on the first proof attempt (whatever session is "current" for the venue is what a bare `/venues/catharsis/` visit joins — there's no per-Show URL parameter) — her `theater_context.selected_character_id` came back empty because she had no roster selection on the K87v2 Show. Closed the stale session directly via SQL (`UPDATE sessions SET status='closed', ended_at=NOW() WHERE id='781d0bda-...'`) rather than deleting anything, so the row is still there for reference. This is disposable test-database housekeeping, not a code change, but flagging it since it affects a database other in-flight work may still be using.
- **`currentIdentity.active_character` and `theater_context.selected_character_id` are two different, unrelated-until-Kernel-74 concepts**, and I initially pointed the new `VictoryStageKernel88Bridge.getMyCharacterID()` at the wrong one (the global Greenroom "active character," `active_user_characters`) instead of the Show-Run-roster-scoped one (`theater_context.selected_character_id`, `world/snapshot.go`'s `resolveTheaterContext`) that `backend/internal/socio`'s own authority checks (`LoadMyRosterMember`) actually key off. Fixed in `runtime.js`; documented inline so a future reader doesn't reintroduce the mismatch. Since Kernel 74, `ActivateCharacterCard` write-throughs into both, so in normal play this bug would rarely surface — it only showed up here because the fixture set the roster selection directly without ever calling the Greenroom "Make Active" endpoint.
- **`GET /api/characters/venue-sheet` depends on a third identity mechanism** (`current_session_personas`, populated by an explicit `persona/equip` action) that the fixture never triggered, so the Player HUD's Face/mechanic-roll section legitimately had nothing to show in this proof run (`no_active_character`). Found and fixed a real bug this exposed: the HUD originally fetched its own `/socio/view` projection and the venue-sheet in one `Promise.all`, so a venue-sheet failure was blocking the Fate/Stance/pools display too. Decoupled them — venue-sheet failure now degrades gracefully (Face/mechanic section just omitted) without affecting the rest of the HUD. The mechanic-roll section itself was never exercised end-to-end in this proof as a result (see §1 item 1).

## 6. Deviations from Kernel

- **No LLM narrator/rules-assistant.** Explicit product decision (confirmed before implementation began): "qualitative condition narration" and "rules assistance" are deterministic (`QualitativeLabel`'s ratio-threshold table; mechanic surfacing is literal compiler-data display via the existing rule-links widget) rather than generated text. Zero new LLM infrastructure exists in this codebase and this kernel does not add any.
- **No Audience-specific Socio visibility tier.** Confirmed with the product owner mid-implementation: Audience never sees Stance/pools/Director tools; "View as Audience" in the spec is satisfied by the Director toggling into the existing Kernel 81/86 Cave/stage view, not a new stripped-down mechanical panel. `ProjectSocioState` has exactly two tiers (Owner, Director) as a direct consequence.
- **No new mechanic-surfacing data model.** The spec's §10/§14 "surface the real compiler mechanic, don't invent one" requirement is met entirely by reusing `CharacterSheetProjection.Skills` and the existing `ewrite-rule-link.js` widget — no `SurfacedMechanic` struct, no tags table, nothing new to keep in sync with the real compiler output.
- **Current Turn has no Group Leader.** The spec's own Kernel-83 precedent includes both; this kernel's product direction was Current-Turn-only for Socio.
- **Progressive-disclosure health bars not built** (§6.1/§33) — see §1 item 2.
- **Story So Far caller wiring not done** — see §1 item 3.
- **Creation/Debrief FP-guidance UI copy not built** — see §1 item 4.
- **Face-switcher UI**: none needed. Confirmed (after initially and incorrectly reporting this as a gap) that Greenroom's existing character-picker + "Make Active" flow already covers this and already keeps the Show-Run roster selection in sync via `syncRosterCharacterSelection` (a Kernel 74 mechanism). No code change was made or needed.
- **Player-initiated roll path untested** — see §1 item 1. This is the most significant deviation from a "fully proven" bar and the primary reason this reportback is PARTIAL rather than PASS.

## 7. Self-Assessment Against Kernel §35/36/37

**PASS criteria met:** Player view is spoiler-safe (qualitative pools, no `exact_pools` field reaches a non-Director viewer at all — structural, not a UI convention, verified both in Go tests and by inspecting the real JSON response in the browser proof); Fate is prominent and usable (Player HUD, spend control, tested cap/clamp/authority); normal Fate cap = 7 (tested); Character Creation temporary over-cap works (tested to 12, clamps on completion-off, logged); stance wheel works (5 real stances, unmistakable active slice via fill+border not color, owner-or-Director authority, tested and screenshot-verified); Director health bars edit canonical values (still true, just not the bar-graph-collapse visual treatment — see deviations); Players do not see exact pool values (structural); state flags work (canonical + blank, tested); compiler mechanics surface contextually via the real, unmodified compiler output; no Character impersonation (player-roll request struct structurally has no `character_id` field, mirroring the established IC-chat pattern — though, per §1, this specific path was never exercised live); Helping works as a Director-adjudicated interrupt with overage correctly computed and consumed exactly once (tested, including the failure case and interrupt-of-interrupt); Current Turn remains the temporal frame, reused from Kernel 83 rather than replaced; the hide-the-ball boundary is enforced structurally, not by convention.

**PARTIAL criteria triggered, honestly:** "Player rolls still require Director submission" is **not** true (the path exists and is authorized correctly) but it is also not proven working end-to-end, which is close enough to this bucket's spirit that I'm not claiming the stronger PASS-level statement. Progressive-disclosure bars are "technically functional but not the specified UI" rather than absent. Everything else in the PARTIAL trigger list (Fate exists but can't be spent — false, tested; stance cosmetic-only — false, tested; compiler mechanics invented — false, reused as-is; interrupt nesting corrupts order — false, tested) does not apply.

**FAIL criteria:** none apply. The client is never authoritative for any hidden Socio state (every mutation and every non-Director read goes through server-side authority/tier resolution, tested both by Go dbtest and by inspecting real HTTP responses in the browser proof); a Player cannot forge Character identity in the roll or Interrupt paths (no such field exists in the request shapes); a Player cannot award their own Fate (the only increasing path, `AwardFate`, is Director-gated; `SpendFate` can only ever decrement); hidden numbers do not leak through the Player-facing API (structural, verified); Help bonus is never applied without a successful, target-complexity-beating roll (tested); the interrupt stack does not replace Current Turn (it reads Current Turn state, never writes a competing notion of "whose turn it is"); Director does not lose authority in any code path added by this kernel (no Audience-tier code exists to lose authority to); no expansion into full CRPG automation occurred; nothing production was touched, and Grant is not locked out of anything.

---

# Kernel 88A — Live Bugfix Pass (2026-08-14)

Opened by Grant reporting that he could not put himself into a cohort: the
Cohorts panel's dropdown "doesn't register at all when I click it, no matter
where in the box," and the Game Status box would not switch between his two
cohorts. Three distinct defects, all now fixed and deployed.

## B1 — Every dropdown and text field in the Director panels was dead (root cause)

`makeDraggable(panel, panel)` made the *whole panel* its own drag handle, and
its `mousedown` handler called `event.preventDefault()` for any target that was
not a `<button>`. `preventDefault()` on mousedown suppresses the browser's
native `<select>` popup outright and blocks click-to-focus on `<input>`. Buttons
were explicitly exempted, which is exactly why "+ New Cohort" and "Archive"
worked while nothing else did.

All five panels called it that way, so the casualty list was: Assign-to /
Move-to, the cohort picker in Game Status *and* Scene Configuration *and*
Current Turn, every HP field, the Fate delta, the stance and status selects,
the blank-flag text box, and the Help Stack's helper-ID / TC / roll-total
fields.

**This predates Kernel 88** — it shipped in Kernel 85 and was live in `HEAD`.
It survived Kernel 85's and Kernel 88's own browser proofs because Playwright's
`selectOption()` sets a value programmatically and never dispatches the
mousedown that the bug depends on. A proof that drives selects programmatically
cannot see this class of defect; only a real mouse can.

Fixed in `frontend/lib/stage-runtime/kernel85-cohort-tools.js`: drag now yields
to any interactive target (`INTERACTIVE_SELECTOR`), and is bound once per panel
instead of re-registering a fresh pair of `window` listeners on every re-render
(the old code leaked two listeners per render). Panels centred with
`translateX(-50%)` also no longer jump on first drag.

## B2 — Stale cohort selection

`state.selectedCohortId` was only ever assigned when empty and was never
validated against the live cohort list, so archiving the selected cohort left a
dangling id: the `<select>` fell back to displaying the first option while every
fetch still asked for the archived cohort. Grant had archived Cohort 18 minutes
before creating 19 and 20, so this was live on his install. Fixed with a single
`resolveCohortSelection()` used by all four panels.

## B3 — The entire Kernel 88 backend had never been deployed

`victory-backend` had been running since 2026-08-12 05:51 — before the Kernel 88
code existed. Confirmed against production: `/api/socio/stances` returned **404**
while Kernel 85's `/api/socio/statuses` returned 401, and `schema_migrations`
topped out at `099_kernel87`. Meanwhile Caddy bind-mounts `/opt/victory/frontend`
read-only into `/srv/web2`, so the Kernel 88 *frontend* was already live-serving
against a Kernel 87 backend. Every Kernel 88 surface (Fate, Stance, Current Turn,
Help Stack, Player HUD) was therefore non-functional in production.

Rebuilt and restarted; migrations 100–104 applied cleanly at boot with the usual
pre-apply backup (`victory_pre_migrate_20260814_065112_5pending.dump`).

## B4 — Player HUD never reflected Director-side changes

`kernel88-socio-player-hud.js` polled every 1.5s but early-returned unless the
Show or Character id changed, so a Fate award or an applied status never reached
the Player. Now re-reads the projection each tick and rewrites the DOM only when
the fetched data actually differs (fingerprint-gated, so polling cannot thrash
the HUD or steal focus), caches the Face/skill sheet across ticks since it only
changes with the played Character, and preserves a half-typed Spend Fate amount
across an unrequested redraw. Poll interval relaxed to 3s.

## Test-database housekeeping (not a code change)

`go test ./...` failed in `internal/shows` and `internal/showtime` with
`venue_busy`, caused by a leftover `rehearsal` session on the shared `catharsis`
venue slug in `victory_test`, left by an ad-hoc Kernel 88 fixture run on
2026-08-13 (the "K88 Fixture Show" string exists nowhere in the repo — the
throwaway fixture program was deleted, per §5 above). Closed it via SQL rather
than deleting the row. Note the contrast with the `k73-fixture-stage-*`
fixtures, which mint their own unique venue slugs and so cannot collide; a
future fixture that needs a session should do the same rather than occupying
`catharsis`. Full suite green afterwards.

## Verification

`scripts/smoke/kernel88a-director-panel-controls-browser.js` (+ its harness
`.html`) is a real-mouse regression proof: it dispatches actual `mousedown`
events and asserts they are not `defaultPrevented`, which is the precise
mechanism of B1. 7/7 pass against the fix. Run against the pre-fix `HEAD` copy
with `--expect-fail`, 4 of the 7 fail — including all three `defaultPrevented`
assertions — confirming the test is sensitive to the bug rather than merely
passing. Evidence: `evidence/kernel-88/07-director-panel-controls-fixed.png`.

Full Go suite: green, `go test -count=1 ./...` with `TEST_DATABASE_URL` set to
`victory_test`.

---

# Kernel 88A — Second Bugfix Pass (2026-08-15)

Grant's follow-up after the first pass: Game Status and the Player HUD
disagreed on Fate, rolling a skill did nothing visible, the Stance wheel's
colours implied a valence Stance does not have, and the HUD forced itself
into every Player's view.

## B5 — Game Status and the Player HUD disagreed on Fate

Both read the same row (`character_socio_fate`, one balance per Character), so
this was never a data conflict — it was staleness. The Director card fetched
Fate once, in a fire-and-forget IIFE at render time, and never again; the HUD
polls. A Player spending Fate moved the HUD and left the Director's card
showing whatever was true when the panel opened.

Fate and Stance are now re-read on a 3s poll while the panel is open
(`refreshCharacterLiveFields`), which is also the one code path that fills them
initially — so there is no second opinion about what those fields say. The poll
deliberately touches only non-editable text: it never overwrites a `<select>`
the Director currently has focus, and it never touches the HP inputs, which
the Director types into.

## B6 — Rolling a mechanic did nothing because there was nothing to click

The HUD sourced its skill list from `GET /api/characters/venue-sheet`, which
resolves the **equipped session persona** (`current_session_personas`, written
by the `persona/equip` action). That is a third identity mechanism, unrelated
to the Show-Run roster selection that both `ProjectSocioState` and the roll
authority path (`actions.StorePlayerMechanicRoll` →
`showruns.LoadMyRosterMember`) key off. `current_session_personas` was empty on
production, so venue-sheet 404'd, `skills` was `[]`, and the "Roll a mechanic"
section — rendered only when it had skills — was omitted entirely. No buttons,
no explanation. The database confirms the click never reached the server: no
`roll/dice` row since 2026-08-10, and no denial in the logs.

Fixed at the source rather than the symptom: new
`GET /api/shows/{show_id}/characters/{character_card_id}/socio/mechanics`
(`socio/mechanics.go`) resolves the skill list from the same roster Character
the roll will authorize against, so a mechanic the HUD offers is by
construction one the roll path accepts. The empty case now states why instead
of vanishing.

This exposed a second real defect on the way: every existing read path into
skill data gates on `CanEditCard` — an **edit** permission — so a Player who
owns their Character but lacks location-level drafting rights could not read
their own mechanics (`forbidden`). Rather than weaken a global authority check,
added `characters.ListCharacterSkillsForAuthorizedReader`, whose contract is
explicitly "the caller has already established authority", called only after
`socio.ResolveTier`. Same reasoning applied to `CharacterDisplayName` — the HUD
had been falling back to the literal string "Your Character" for exactly the
same reason.

## B7 — Stance wheel colours implied a valence

The wheel spread a ~300° hue rotation across the slices, so Insight landed on
red and read as a warning. Stance carries no valence. Now one slate-blue hue
(212°) stepped only in lightness; the active slice is still marked by fill AND
border, never colour alone.

## B8 — The HUD forced itself into the view

It now starts minimized to a small header pill showing Character name and Fate,
drags by that header, minimizes/restores, and persists position and open state
in `localStorage`. While minimized it makes no network requests at all. Its
drag handler carries the same interactive-element guard as the Director panels,
so the fix for B1 cannot be reintroduced here.

## Verification

- `scripts/smoke/kernel88a-player-hud-browser.js` — 11/11. Asserts collapsed-by-
  default, zero requests while collapsed, roll buttons rendering *while
  venue-sheet 404s* (the exact production state), the roll action being sent,
  one-hue wheel (computed from rendered colours, since Chromium serializes
  `hsl()` back as `rgb()`), header drag, the minimize button surviving the drag
  handle, and persistence across reload.
- `scripts/smoke/kernel88a-director-panel-controls-browser.js` — 7/7, unchanged.
- Three new Go tests pin the mechanics endpoint to the roster Character with no
  equipped persona, reject an unrelated viewer, and distinguish Owner from
  Director tier. Full suite green.
- Evidence: `evidence/kernel-88/08-player-hud-monochrome-wheel.png`.

## Known gap, found not fixed

The HUD reports "Roll sent." on a successful socket write, but the server's
`type: "error"` WS reply (e.g. a denied roll) is never surfaced — this module
has no listener on the socket, only `sendAction`. A rejected roll is therefore
still silent. Worth a Kernel 88B: route WS errors carrying a `request_id` back
to whichever module originated that request.

---

# Kernel 88B — Per-Request Error Routing (2026-08-15)

Closes the gap flagged at the end of the 88A pass: an action refused by the
server produced a `{type:"error", request_id}` frame that only ever reached
generic surfaces (stage status line, movement line, system chat notice, dice
tray). The module that *sent* the action never learned it had failed. The Socio
HUD's mechanic roll was the motivating case — "Roll sent.", then nothing.

## Design

A module registers the `request_id` it is about to send, paired with a handler.
If an error for that id arrives, the handler runs and **claims** the error, and
the generic surfaces stand down. Claiming matters for more than tidiness: the
generic path appends a system chat notice, so an unclaimed private failure gets
announced to the whole room.

New module `frontend/lib/stage-runtime/action-requests.js`
(`createActionRequestRegistry`) — deliberately a separate UMD module in the
style of `socket.js`/`session-sync.js` rather than a closure inside
`runtime.js`, because `runtime.js` is browser-only and untestable outside a
page. Extracting it made the registry unit-testable in plain Node.

Chain: `socket.js` normalization now carries `requestId` → `session-sync.js`
offers the error to `deps.handleActionError` first and returns early if claimed
→ `runtime.js` owns the registry and exposes `registerActionRequest` on
`VictoryStageKernel88Bridge`.

Deliberate properties:
- **Fails open.** An unknown id, a missing id, a handler returning `false`, or a
  handler that throws all leave the error unclaimed and fall through to the old
  behaviour. The cost of failing open is a duplicate message; the cost of
  failing closed is a silent failure, which is the bug being fixed.
- **Registrations expire** (15s default). The common outcome is success, which
  produces no reply at all, so without expiry every successful action would leak
  a handler for the life of the page.
- **Register before send.** A server that refuses faster than `sendAction`
  returns must still find a handler waiting; a send that fails withdraws its
  registration rather than leaving one waiting for a reply that cannot come.
- **Degrades loudly.** `runtime.js` falls back to a no-op registry with a
  console error if the module is missing, rather than throwing and taking the
  entire stage down for a non-essential feature.

## Server side: no change needed, and why

Audited all 45 error writes in `network/ws.go`. Exactly two handlers accept a
`request_id` at all — `roll/dice` and `roll/dice_own_mechanic` — and both echo
it on **both** of their error branches. So the invariant "every handler that
receives a request_id echoes it on every error path" already holds. The other
43 error writes belong to actions whose senders never supply an id; those
frames route to the generic surfaces exactly as before (pinned by a test).

Retrofitting `request_id` across every action type was considered and rejected:
it is broad regression surface with no consumer asking for it. It should be
driven by the second module that actually needs it.

## Verification

- `scripts/smoke/kernel88b-action-error-routing-test.js` — 14/14, plain Node
  (these modules are UMD factories). Covers delivery, unknown/missing id,
  explicit decline, throwing handler, once-only delivery, expiry, withdrawal,
  re-registration replacing rather than stacking, and both socket normalization
  cases.
- `scripts/smoke/kernel88b-wiring-browser.js` — loads socket.js,
  action-requests.js and session-sync.js into a real browser in the venue
  pages' load order and pushes a raw frame through the whole chain. This is the
  part the Node tests cannot reach, since `runtime.js` is browser-only.
- `scripts/smoke/kernel88a-player-hud-browser.js` — now 16/16, extended with
  the HUD's half of the contract: registers before sending, registered id
  matches the sent id, a refusal appears on the HUD naming the mechanic, the
  error is claimed, and a roll that cannot be sent withdraws its handler.
- Go suite green; no backend change in this pass.

## Known gap (closed same day -- see Kernel 88C below)

The `ws.go` error paths wrote via `c.Conn.WriteJSON` directly, so the server's
error branches could not be exercised by the existing `handleCavePayload` test
seam. Verifying the request_id echo invariant meant reading the six relevant
lines rather than testing them. Closed by the 88C pass below -- which found a
real concurrency defect sitting underneath it.

---

# Kernel 88C — WebSocket Write Consolidation (2026-08-16)

Closes 88B's known gap. Started as a testability refactor; the audit found a
real concurrency defect underneath it.

## The actual bug: two goroutines writing one connection

Every venue and storyboard socket runs `writePump` in its own goroutine,
draining `Client.Send` with `Conn.WriteMessage`. Meanwhile 51 handler call
sites wrote to the *same* connection directly with `c.Conn.WriteJSON`, from the
read goroutine.

`gorilla/websocket` supports exactly one concurrent writer. Any broadcast
landing while a handler was replying was a `concurrent write to websocket
connection` panic waiting on timing. It had presumably never fired often enough
to be noticed, but nothing prevented it -- it was luck, not design. Two sockets
were affected (`network/ws.go`, `storyboards/ws.go`), both with the same shape.

## The change

New `Client.SendJSON` queues onto the same `Send` channel the writePump drains,
so the connection has exactly one writer by construction. All 51 direct writes
converted; `network` and `storyboards` now pass `go test -race`.

Four writes deliberately remain on `conn.WriteJSON`: the handshake sequence
(snapshot, presence snapshot, pinned stage effects, and their failure reply).
Those run *before* `go writePump(client)` starts, so that goroutine is provably
the only writer, and each needs its error synchronously to abandon the
connection -- queuing would hand the failure to a goroutine that does not exist
yet. Marked with a comment saying so, since they otherwise look like misses.

Two consequences worth recording:

- **`CloseSend` + `closeSendOnce`.** The session-revocation path used to write
  a `session_revoked` notice and immediately `Conn.Close()`. Queuing the notice
  and then closing the socket would usually lose it, so that path now closes
  the *channel* instead: writePump drains the buffer, delivers the notice, then
  its deferred `Conn.Close` tears down the socket. That gave `Send` a second
  closer alongside `Hub.Remove`, and closing a channel twice panics -- hence the
  `sync.Once` guard. Pinned by `TestCloseSendIsIdempotent` and
  `TestQueuedMessageSurvivesCloseSend`.
- **Buffer raised 16 -> 64** on both sockets. Per-client replies used to bypass
  the buffer entirely, so 16 was sized for broadcast traffic alone. Now every
  write shares it, and overflow drops a *reply*, not just a broadcast frame.

## What this unlocked

`kernel88b_ws_error_reply_test.go` -- 9 tests, all previously impossible.
Every client in it is built with a **nil `Conn`**, which is the structural
proof: a handler still writing to the connection directly would panic rather
than fail an assertion.

The 88B invariant is now enforced rather than asserted by inspection: both
`roll/dice` and `roll/dice_own_mechanic` echo `request_id` on their denial
branch *and* their store-failure branch, and omit the key entirely when the
client sent none (an empty-string id would be a trap -- the client treats "" as
"nobody is waiting"). Also covers the server-authored-event rejection,
ping/pong, and SendJSON's drop-rather-than-block contract, since blocking there
would wedge the read loop.

## Verification

- 9 new Go tests in `internal/network`; full Go suite green.
- `go test -race ./internal/network/ ./internal/storyboards/` clean.
- Deployed. Live `/ws/catharsis` and `/ws/storyboards` reject unauthenticated
  upgrades cleanly (403/401) with no hang or panic; no errors in the logs after
  restart.
- All four frontend proofs re-run green (14/14, wiring OK, 16/16, 7/7).
- Stale comment in `kernel86_stage_effect_test.go` that documented the old
  untestability corrected rather than left to mislead.
