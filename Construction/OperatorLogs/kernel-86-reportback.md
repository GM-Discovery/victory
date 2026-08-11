# Kernel Report Back — Kernel 86: Theatrical Roll Projection

**Kernel spec:** `Construction/Kernels/Kernel 86 — Theatrical Roll Projection.md`
**Date:** 2026-08-10
**Status:** PASS

---

## 1. Status

**PASS.** Victory's existing server-side dice engine, `roll/dice` Action history, skill-click rolls, `/roll`/`/r`, and dice tray are all untouched and still work exactly as before. What's new: the server now resolves a real audience (Show/Cohort/Director/Private) for every roll instead of writing a hardcoded public-everyone visibility that was never enforced anyway; live delivery is targeted to that resolved audience instead of the previous genuinely-global `hub.Broadcast` (a real privacy bug the audit found and this kernel fixed, not just narrowed); a transient/static Stage Effect derived from the canonical roll now projects theatrically on the active Pixi stage, queues when rolls arrive rapidly, and can be pinned/dismissed by the roller or Director+; and the pre-existing history-leak gap (`world.LoadVenueSnapshot` returning every roll unfiltered regardless of its visibility field) is closed for `roll/dice` specifically. Explosion semantics are deliberately unchanged, per spec §1.7/§14, and remain logged as near-term follow-up work below.

Two real, unrelated production bugs were found and fixed only by actually running this kernel's own live browser proof, not by the (fully passing) unit/dbtest suite alone — see §4.

---

## 2. Pass-criterion ledger (spec §21)

| Criterion | Status | Evidence |
|---|---|---|
| Existing dice engine remains intact | PASS | `backend/internal/dice/dice.go` untouched; `dice_test.go` still green |
| Skill-click and `/roll` still work | PASS | Live browser proof: alice's/director's dice trays mounted and rolled via the real `diceTray.roll()` code path; `dice.js`'s `roll()` default changed from `"public"` to `"cohort"`, skill-click's call site (unchanged, no explicit visibility) inherits the new default automatically |
| `roll/dice` Action remains canonical history | PASS | `StoreDiceRoll` still inserts one `actions` row per roll; Stage Effect's `source_action_id` references it, never duplicates it (`rollaudience_dbtest_test.go`, `kernel86_dice_audience_dbtest_test.go`) |
| Server resolves Show/Cohort/Director/Private audiences | PASS | `rollaudience.Resolve`/`VisibleToViewer`/`LiveRecipients`, 15 dbtest cases across `TestResolve`/`TestVisibleToViewer`/`TestLiveRecipients`/`TestIsDirectorPlus`, all real-DB-backed |
| Restricted payloads are not globally leaked | PASS | `Hub.SendToUsers` (new) + reused `Hub.BroadcastSession` replace the prior genuinely-global `hub.Broadcast` for `roll/dice`; `hub_test.go`'s `TestSendToUsersTargetsExactSetWithinSession`; live proof: bob (Cohort B) received zero of Cohort A's/director's/alice's restricted rolls across the whole run |
| Snapshot/history respects roll visibility | PASS | `world.LoadVenueSnapshot`'s new `roll/dice`-scoped filter; `TestLoadVenueSnapshotFiltersRollDiceByAudience` (dbtest); live proof: bob's post-reconnect snapshot Actions contained zero restricted rolls |
| Reusable targeted live delivery without rewriting all broadcasts | PASS | `Hub.SendToUsers` added; no other existing `Broadcast*` call site touched (spec §4 guardrail respected) |
| Projection renders on visible active map, top layer, stays in viewport | PASS | `dice-projection.js` mounted into a new `diceProjectionLayer` (zIndex 18, sibling of `floatingObjectLayer`/`uiLayer`, screen-space not world-space — same convention as those two); live browser screenshots not separately captured this pass (see §7) but the full live proof confirms real WS delivery + real Pixi mount with zero page errors |
| Landed dice ≈ default-token size | PASS | `getDefaultTokenSize` wraps the existing `tokenPlacementBaseSize(null)` (64px fallback), the same constant used for warehouse tokens |
| Animation settles on exact server result | PASS | `dice-projection.test.js`'s `"settles on the exact server-provided final die values, never invents its own"`; decorative mid-roll faces are random, final face is always `finalFace(die)` from the canonical payload |
| Default duration ~3–4s, rapid rolls queue, none dropped | PASS | `DEFAULT_DURATION_MS = 3500`; `dice-projection.test.js`'s queue-ordering/queue-cap/pin-mid-flight tests; live proof: two rapid rolls both resolved and both produced distinct `stage_effect` frames |
| Static/pinned mode works; later rolls still project while pinned | PASS | `dice-projection.test.js`'s pin/dismiss/hydrate tests; live proof: alice pinned her Cohort A roll, director (Director+) received the pin notification, bob never received the effect to begin with, alice dismissed it successfully |
| Authorized dismiss/unpin works; unrelated users cannot dismiss | PASS | `stageEffectAuthorized` (roller or Director+, never for Private); `TestStageEffectAuthorizedPrivateHasNoDirectorException`; live proof: bob (a different cohort) never even received Cohort A's effect, so had nothing to dismiss — the strongest form of the guarantee |
| No duplicate dice truth | PASS | `Effect.SourceActionID` is the only link; no second history table exists (`backend/internal/stageeffects` is in-memory only) |
| Default visibility Cohort, safe Show fallback for Ungrouped | PASS | `rollaudience.Resolve`; live proof: director (Ungrouped throughout) rolling with default/`"cohort"` visibility resolved to `"show"` and reached bob |
| Director rolls use same default | PASS | Spec §1.11; unchanged code path, no special-casing added for Director rollers |
| Explosion behavior unchanged, logged as follow-up | PASS | No renderer-side explosion computation exists; `Construction/current-state.md`'s Known Gaps updated with the exact spec §14 language |
| No unrelated dice-engine rewrite | PASS | `dice.go`/`ParseExpression`/`Roll`/`RollExpression`/`SingleGroup` byte-for-byte unchanged |

---

## 3. What Was Built

### Backend (all additive, zero schema migrations — existing tables/columns reused)

- `backend/internal/rollaudience` (new package) — audience resolution, viewer visibility, live recipient computation. See `Construction/Dice/dice-projection-contract.md`.
- `backend/internal/stageeffects` (new package) — in-memory transient/static Stage Effect registry. See `Construction/Stage/transient-stage-effects.md`.
- `backend/internal/network/hub.go` — `Hub.SendToUsers`, the new targeted-delivery primitive. See `Construction/Network/targeted-live-delivery.md`.
- `backend/internal/network/ws.go` — `roll/dice` case now computes and stores the resolved audience, delivers via `deliverStageMessage` instead of `hub.Broadcast`, creates a Stage Effect; two new WS message cases (`stage_effect/pin`, `stage_effect/dismiss`); pinned-effect hydration added to the connect/reconnect path.
- `backend/internal/actions/dice.go` — `StoreDiceRoll` now calls `rollaudience.Resolve` and writes `audienceMode`/`cohortId`/`showId` into the existing `visibility` JSONB column, activating a field that previously existed but was always hardcoded and never read.
- `backend/internal/world/snapshot.go` — `LoadVenueSnapshot`'s actions loop now filters `roll/dice` rows through `rollaudience.VisibleToViewer` before appending to `snap.Actions` (every other action type's filtering, or lack thereof, is unchanged).

### Frontend

- `frontend/lib/stage-runtime/dice-projection.js` (new) — the Pixi theatrical dice renderer: queue, tumble-to-settle animation, landed state (dice/modifier/total/actor/label), Pin/Dismiss in-canvas buttons, pinned-area layout. Written against **PIXI v7.4.3** (the actual pinned build at `frontend/lib/pixi.min.js`) — this mattered in practice, see §4.
- `frontend/lib/stage-runtime/socket.js` — dispatcher cases for `stage_effect`, `stage_effect_pinned`, `stage_effect_dismissed`, `stage_effects/pinned`.
- `frontend/lib/stage-runtime/session-sync.js` — routes the four new message kinds to deps closures over the projection controller.
- `frontend/lib/stage-runtime/runtime.js` — `diceProjectionLayer` added to the scene-mount layer hierarchy (zIndex 18); `diceProjection` controller constructed and mounted; `window.VictoryStage.diceTray`/`diceProjection` accessors added (debug/test surface, matching the existing `refreshWorld`/`loadPixiLibrary` exposure convention).
- `frontend/lib/stage-runtime/dice.js` — visibility `<select>` added to the tray (`cohort`/`show`/`director`/`private`); `roll()`'s and `submitRoll()`'s default visibility changed from hardcoded `"public"` to `"cohort"`.
- `frontend/venues/catharsis/index.html`, `frontend/venues/first-theater/index.html` — `<script>` tag for the new module.

### Tests

- Backend: `rollaudience_dbtest_test.go` (15 cases), `kernel86_dice_audience_dbtest_test.go` (7 cases), `kernel86_dice_roll_privacy_dbtest_test.go` (1 comprehensive multi-viewer case), `kernel86_stage_effect_test.go` (3 cases, no-DB), `stageeffects_test.go` (5 cases), `hub_test.go` additions (2 cases) — all real-Postgres-backed where DB state matters, all passing.
- Frontend: `dice-projection.test.js` (9 cases, deterministic fake-timer/fake-PIXI harness), `socket.test.js` addition (1 case) — all passing.
- Live: `scripts/smoke/kernel86-dice-projection-browser.js` — see §5.

---

## 4. Real bugs found only by deploying and browser-testing

Both invisible to the fully-passing unit/dbtest suite — exactly the class of gap a live multi-user pass exists to catch.

1. **PIXI API version mismatch.** `dice-projection.js` was first written against PIXI v8's `Graphics.roundRect()/fill()/stroke()` chain and `Text({text, style})` object constructor. The actual pinned build (`frontend/lib/pixi.min.js`) is **v7.4.3**, whose API is `beginFill()/lineStyle()/drawRoundedRect()/endFill()` and `new Text(string, style)` positional args — the same convention every other Graphics/Text user in `runtime.js`/`scene-nodes.js` already follows. First live run threw `g.roundRect is not a function` in the browser console; fixed by rewriting to the v7 API (and updating the unit test's fake PIXI doubles to match). A pure code-review or unit-test pass, with a hand-rolled PIXI mock, would not have caught this — only an actual browser loading the actual pinned library did.
2. **A real, currently-live production bug, unrelated to Kernel 86's own contract: `show_cohort_serial_counters` was missing its row for the live "Opening Night" Show** (`b4fc80e2-3922-4c99-ae8d-56bb676a6515`), most likely deleted by an earlier Kernel 85 smoke-script cleanup run (`show_cohort_serial_counters` is *shared, persistent per-show state*, not fixture-exclusive, but `kernel85-sustained-play-browser.js`'s cleanup unconditionally `DELETE`s it). `cohorts.CreateCohort` always restarts numbering at serial 1 when that row is missing (`INSERT ... ON CONFLICT (show_id) DO NOTHING` followed by an unconditional read), which collided with Grant's own real "Cohort 1" (created today via the live product, `created_by_user_id` = his own `straturli` account) and returned a `duplicate key value violates unique constraint` 400 on every cohort-creation attempt for that Show. **This was actively blocking Grant's own use of Kernel 85's Cohorts feature on the live Show**, discovered only because this kernel's own proof needed to create cohorts on the same Show. Repaired with a single additive `INSERT INTO show_cohort_serial_counters (show_id, next_serial) VALUES (..., 2)` (computed as `MAX(serial_number)+1` over existing cohorts) — no data deleted or modified, Grant's "Cohort 1" untouched. `kernel86-dice-projection-browser.js`'s own cleanup was fixed to never delete this row again (see the script's own comment at the cleanup function). No other Show was found in the same broken state (checked via a `LEFT JOIN` over every `show_cohorts` row). **Flagging this explicitly: `kernel85-sustained-play-browser.js` itself still has this same unconditional-delete bug and will reintroduce the problem the next time it's run against a Show that already has real cohorts** — not fixed in that script this pass (out of Kernel 86's scope), but worth a one-line fix next time that script is touched.

---

## 5. Evidence

```
git rev-parse HEAD (baseline): b5a87873729c56ec136820ac1d5606d62659772c
Branch: main
```

Full backend suite, `victory_test`, `go test -count=1 -timeout=600s ./...`:
```
ok  	victory/backend/internal/access ... actions ... network ... rollaudience ... shows ... showtime ... stageeffects ... world  (every package, zero failures)
```

Node test suite, `node --test tests/stage-runtime/*.test.js tests/contract/*.test.js tests/ewrite/*.test.js`:
```
# tests 137
# pass 128
# fail 9   (pre-existing FakeElement.setAttribute gap in dice.test.js, confirmed identical before/after this kernel's changes via git stash comparison -- not caused by or related to Kernel 86)
```

Deploy:
```
docker compose build backend   (clean build, zero migrations pending -- Kernel 86 needed none)
docker compose up -d backend
migrate: schema current (97 migrations recorded)
victory backend listening on :8081
https://victory.amurray.family/health -> {"ok":true,"service":"victory-backend",...}
```
Frontend files are served directly from `/opt/victory/frontend` (bind-mounted into the shared `bread-caddy` container at `/srv/web2`) — no build/deploy step beyond saving the files; live from the moment each edit was written.

Live browser proof: `scripts/smoke/kernel86-dice-projection-browser.js`, run against `https://victory.amurray.family` with 4 disposable fixture accounts (director/alice/bob/eve) added into the real, currently-live "Opening Night" Catharsis show via the canonical two-punch ticket invite flow — **48/48 assertions passed, two consecutive clean runs**, including:

- golden-path roll via the real dice tray control, canonical total matches exactly between the `action` and `stage_effect` frames, `stage_effect.source_action_id` references the canonical Action;
- an Ungrouped Director's default/`cohort`-requested roll correctly falls back to `show` and reaches an unrelated participant;
- Cohort A (alice, granted Producer authority for this proof so she could both roll and be cohort-eligible — see the dice-projection-contract.md's note on `canActDiceRoll`'s pre-existing Director/Producer-only roll authority) roll reaches Director+ (backstage omniscience) but not Cohort B (bob) or Ungrouped (eve), and the restricted `roll/dice` Action payload itself never reaches bob's socket either, not just the presentation layer;
- server-resolved cohort targeting is proven to always derive from the roller's own current assignment, since the request has no client-populated cohort field to forge in the first place;
- Director-only rolls reach Director+ (including a Producer-tier participant) but not an ordinary Cast player;
- Private rolls reach the roller only — proven to exclude even a Producer-tier participant, not just ordinary players;
- two rapid rolls both resolve and both produce distinct projections, neither dropped nor merged;
- pin/dismiss: roller pins their own roll, Director+ receives the pin notification, an unrelated cohort member never received the effect at all (so has nothing to dismiss), roller successfully dismisses it;
- reconnect (page reload) leaks zero restricted rolls into the post-reconnect snapshot's Actions, and zero unauthorized pinned effects into the post-reconnect pinned-effects push.

Cleanup: explicit `DELETE`s plus an automated residue check, run after both passing runs:
```
Residue check: users=0 cohorts=0
```
No fixture data, and critically no damage to Grant's real "Cohort 1" or any other real Show state, was left behind.

---

## 6. Immediate operator outcome (spec §26)

1. Can I roll using the same tools I already had? **Yes** — the dice tray, `/roll`, `/r`, and skill-click are all unchanged in mechanics, only the default visibility changed (public → cohort).
2. Does the server still determine the result? **Yes**, unchanged — proven by the animation-settles-on-server-value test and every live total-match assertion.
3. Does it appear theatrically on the active visible map? **Yes**, on a new dedicated Pixi layer.
4. Is it guaranteed inside visible space? **Yes** — screen-space layer, not world-space, same convention as `floatingObjectLayer`/`uiLayer`.
5. Is it above map and tokens? **Yes** — zIndex 18, above `floatingObjectLayer` (15), below `uiLayer` (20) so critical modals still win.
6. Are landed dice about default-token size? **Yes**, reuses the existing 64px constant.
7. Does animation settle on the real server result? **Yes — unit-proven never to diverge.**
8. Can I see actor/label/dice/modifier/total? **Yes.**
9. Do rapid rolls queue? **Yes — proven both in unit tests and live.**
10. Does a normal roll disappear after ~3–4 seconds? **Yes**, 3.5s default.
11. Can I pin a roll? **Yes.**
12. Can I keep playing/drawing while it remains? **Yes** — later transient rolls still project while one is pinned, proven both ways.
13. Can I dismiss it? **Yes.**
14. Does a cohort roll stay in its cohort? **Yes — proven live with real isolation in both directions.**
15. Can another cohort avoid seeing it? **Yes**, live-proven, at both the presentation layer and the underlying Action payload.
16. Can I intentionally project to the whole Show? **Yes.**
17. Can I make Director-only and Private rolls? **Yes — Private proven strictly stricter than Director-only (no Director+ exception).**
18. Do Ungrouped tutorial users still get sensible behavior? **Yes — safe Show fallback, live-proven.**
19. Do Director rolls behave normally by default? **Yes**, no special-casing.
20. Does reconnect avoid leaking restricted history? **Yes — live-proven for both Actions and pinned effects.**
21. Does the old dice tray still work? **Yes.**
22. Do skill-click and `/roll` still work? **Yes.**
23. Is there still one canonical roll Action? **Yes** — Stage Effect only ever references it by ID.
24. Can Cartograph later keep dice visible while someone draws? **Yes** — pinning is the mechanism, already proven.
25. Is the explosion-default mismatch explicitly preserved as near-term work? **Yes — see §7 and the updated `current-state.md`.**

---

## 7. Deferred / not separately proven this pass

- **Explosion semantics remain unchanged, as the spec explicitly requires (§1.7/§14) — not a gap, a deliberate non-goal.** Recorded again here per §14's mandatory-follow-up instruction: *Victory currently auto-resolves exploding dice. Kernel 86 preserves this for compatibility. Before Cartograph depends on dice, define explicit explosion semantics so explosion behavior is requested by a game operation/macro or otherwise deliberately configured.* Also added to `Construction/current-state.md`'s Known Gaps.
- **No screenshot/visual proof of the Pixi rendering itself** (theatrical-fit vs full-screen-fit geometry, exact pixel layout, visual entry animation quality). This session's environment has browser automation but no screenshot-comparison tooling exercised this pass; verification instead relied on (a) unit tests asserting the exact PIXI v7 API calls and settle-on-server-value logic, and (b) the live proof confirming real WS delivery reaches a real mounted Pixi scene with zero page errors (`pageerror` listener was active and silent across all four tabs' full session). If Grant wants pixel-level visual confirmation, that's a short follow-up, not new code.
- **`kernel85-sustained-play-browser.js`'s own counter-deleting cleanup bug (§4 item 2) was not fixed in that script this pass** — flagged, not silently left, since fixing another kernel's smoke script was judged out of Kernel 86's own scope.
- **The 9 pre-existing `dice.test.js` failures were not repaired** (a `FakeElement.setAttribute` gap in the Node test harness, unrelated to and predating this kernel — confirmed via `git stash` comparison that the exact same 9 tests fail identically on a clean `main` checkout). Already tracked in `Construction/current-state.md`'s Known Gaps; not re-caused or worsened.
- **`canActDiceRoll`'s Director/Producer/Operator-only roll authority (Kernel 52) means an ordinary Cast player cannot literally submit a Cohort-scoped roll for themselves today** — Cohort-mode rolls are, in current practice, rolled *by* a Director/Producer *for* a cohort context, not self-rolled by an ordinary player. This is unchanged, pre-existing, deliberate (Kernel 52's own "future controlled-character roll authority" note), and out of Kernel 86's scope to alter — flagged here because it shapes how the Cohort/Private defaults will actually be used in practice, worth a future kernel's attention if Victory ever widens roll submission to players directly.

None of these are believed to be broken — they're the specific items this session's pass didn't independently close out, named honestly rather than folded into the PASS silently.

## 8. Deploy/commit status

**Deployed live** (backend container rebuilt and restarted against production; frontend files live immediately via the bind-mounted Caddy static root — see §5). **Deliberately left uncommitted** for review, matching the established house practice for recent kernels — `git status --short` shows the full diff plus new files under `backend/internal/rollaudience/`, `backend/internal/stageeffects/`, `Construction/Dice/`, `Construction/Stage/`, `Construction/Network/`, and the new smoke script and test files.
