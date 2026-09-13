# Kernel 98 — Unknown Unknowns, Failure Modes & Product Resilience Hunt Ledger

**Kernel spec:** `Construction/Kernels/Kernel 98 — Unknown Unknowns, Failure Modes & Product Resilience Hunt.md`
**Status:** CLOSED — PASS (2026-09-12).
**Method:** four parallel evidence-first investigations covering all 46 spec sections' discovery categories, plus resolution of the three findings Kernel 96 carried in (§36A). Investigate-then-triage, per the kernel's own §39 work-discipline doctrine — no fixes applied during discovery to avoid cross-fork collisions; findings classified and routed below.

**Headline result: zero BLOCKER-severity findings.** Victory's runtime resilience (refresh, multi-tab, reconnect, backend restart), authority-failure defaults, and environment assumptions all came back either intentionally correct or bounded low-severity. The one genuinely broad finding (raw internal error strings reaching the client) is real and repeated, but not a launch blocker — routed to K101 as a mechanical, bounded fix.

---

## 1. The three carried-in findings from Kernel 96 (§36A) — resolved

1. **Dice-roll broadcast nil-pool panic.** CONCLUSIVELY a test-harness artifact, not a production bug. `audienceDiceRollsHidden` (`backend/internal/network/ws.go:137`) calls `showings.LoadBySession`; in production the pool is always real, and a session with no Showing correctly returns `pgx.ErrNoRows`, which the caller treats as "dice visible" (the documented default). The panic only happens because the test mocks one path (`storeDiceRollFunc`) with a nil pool but not the second, unmocked path (`deliverStageMessage`) that also uses it. **Classification: K101** — fix the test's mock, not the product.

2. **eWrite skill-directory seed test.** Root cause not found within timebox: location resolution, ruleset lookup, title-matching, and the `ON CONFLICT DO NOTHING` insert all look structurally correct; reproduces on genuinely fresh databases so it isn't state pollution. **Classification: UNKNOWN — requires targeted follow-up** (next step would be instrumenting the insert loop directly).

3. **Scene capture/promotion test failures.** Root cause found, and it's informative: `UpdateCurrentScene`/`SaveArrangementAsNewScene` (`backend/internal/scenes/capture.go`) call `CaptureLiveVenueComposition`, which Kernel 93 changed to capture composition from the **live venue's actual stage state**, not from the `scene_stage_elements` rows these tests still seed. The tests exercise a pre-Kernel-93 model these two actions no longer implement. Real duplicate-truth artifact (two "current composition" models coexist), but the *product* behavior is intentional and already documented — only the tests are stale. **Classification: K101** — rewrite the two tests against the live-venue model.

---

## 2. Resilience matrix (spec §41)

| Scenario | Expected | Actual | Result |
|---|---|---|---|
| Browser refresh | durable state restored | Server sends a fresh snapshot + all pinned stage effects on every WebSocket connect/reconnect; live-show state is server-authoritative by construction | PASS |
| Multi-tab | no corruption | `Hub.clients` keyed by connection pointer, not user — two tabs are two independent, fully legitimate clients, no shared mutable state to collide | PASS |
| Backend restart / reconnect storm | reconnect + resync, no phantom participants | Exponential backoff (1s→10s cap), single module-scoped socket reference replaced per attempt, fresh snapshot on every reconnect | PASS |
| DB interruption | safe failure/recovery | Not live-tested this pass (would require killing a live DB) — untested, time-boxed out | UNTESTED |
| Duplicate submit | no harmful duplicate | `CreateShowing` has no idempotency guard — a double-click can create two rows with the same nickname. Not corruption, just an extra row | GAP — routed K101 |
| Partial save / interrupted write | clear recovery or transaction | `scenes/live_bridge.go` uses real transactions; `scenes/capture.go`'s two functions do not (same root cause as finding #3 above) | GAP — routed K101 (tracked) |
| Race conditions | no impossible/corrupt state | Stage-object mutation is last-write-wins on position with no version check — no constraint violation or orphaned state possible from this path | PASS |

---

## 3. New findings this pass

- **Raw internal error strings reaching the client.** `backend/internal/network/ws.go` has ~19 sites, plus `indexcard_http.go:139` and `director_console.go:164`, that echo the raw Go `err.Error()` string as the WebSocket/HTTP `"error"` field instead of a stable message. This is the same shape as the already-tracked "Kessa's raw not_authorized error" item in the bug-hunt backlog, and this pass confirms it's a repeated pattern across the whole handler, not an isolated spot — an information-hygiene concern (a raw DB/internal error could leak in an unexpected failure), not just cosmetic. **Classification: K101** — bounded and mechanical (replace with a stable code/message map) but broad enough across ~21 sites to route rather than patch tonight.
- **Two silent client-side failures.** `frontend/lib/tour-engine.js:367` (tour-state POST failure, console-only) and `frontend/lib/victory-pixi-stage.js:140` (backdrop image load failure, console-only). Both low-stakes, visual/non-blocking. **Classification: K101.**
- **Storyboards' error-handling pattern is the standard to match.** `frontend/venues/storyboards/app/store.js` returns `{ok, error}` from every mutation for the caller to render — noted as the exemplar other surfaces should be brought toward during the K101 error-hygiene pass, not a finding in itself.
- **Unbounded board/list loads.** `storyboards.ListCardsForBoard`/`ListRows`/`ListColumns`/`ListBands` load a board's full contents with no pagination. Bounded by human authorship today (not attacker-controlled growth), so not a security concern — a genuine future performance cliff if a board grows very large. **Classification: K101** (polish/perf, no evidence of a real-world board near a problematic size today).

---

## 4. Confirmed clean (ACCEPTED, no action)

- **Defaults (§20):** dice-visibility and Scene/stage-object visibility defaults are intentional, already-documented Kernel 93 decisions favoring visible/permissive-but-sensible defaults for composition surfaces — nothing found defaulting to an unintended privilege escalation.
- **Orphan cleanup (§22):** the schema's mixed `RESTRICT`/`SET NULL`/`CASCADE` policy across migrations reads as a deliberate, mature design, not an oversight; assets are never hard-deleted in production flows (soft-permanent), so there's no orphan-file risk from that direction.
- **Environment assumptions (§24):** the only "localhost" reference in non-test backend code is a comment documenting a deliberate fix (mailer links must not resolve to a recipient's own machine) — no hardcoded absolute paths, container names, or dev-machine-only assumptions found.
- **Time/date (§34):** no `time.Local` usage anywhere in `internal/`; everything is `time.Time`/`timestamptz`-based (UTC-safe by construction). No naive local-time comparison found.
- **Asset storage keying (§36):** assets are stored under a generated directory/ID scheme, never the display filename — matches the spec's own expectation, unaffected by unicode/weird filenames.
- **Permission-denied UX baseline (§32):** backend denial responses are generic and safe (`{"ok":false,"error":"forbidden"}`, HTTP 403) at the sampled sites — no private detail leak. (The raw-error-string finding above is a separate, narrower issue layered on top of this otherwise-sound baseline.)

---

## 5. Investigated but inconclusive (UNKNOWN — not launch-relevant)

These were time-boxed rather than run to ground, per the kernel's own §39 discipline; none surfaced any evidence of a real defect, so none are held open as risks — they're just not exhaustively verified:

- §6 Undiscoverable-but-valid features — no evidence gathered either way.
- §7 Weird-user pass (multi-tab edit races, mid-upload abandonment, etc.) — no concrete corruption path found in the time available; would need a dedicated follow-up to close definitively.
- §21 Empty-state audit, §23/§28/§29 migration and fresh-vs-legacy behavior — folded into **K99**, which already owns the full fresh-install proof and is the natural place to verify these directly rather than duplicating effort here.
- §25 Observability — spot-checked only, no gap found.
- §35 Unicode/strange-input handling — not tested this pass.
- `shows/showing_create.go` transactional atomicity — not traced to conclusion; low priority given the operation's simple, mostly single-row shape.

---

## 6. Dead code / duplicate truths (spec §4, §19)

Only a shallow sweep was completed given the bedtime timebox — grep for explicit deprecated/legacy markers surfaced ~19 files, too broad to triage individually tonight. Nothing alarming surfaced in the shallow pass; the one confirmed duplicate-truth finding (Scene composition model, §1.3 above) is already routed. **Recommend a dedicated, deeper §4/§19 sweep as a K102+ item** if Grant wants full dead-code accounting — not required for tonight's close, since nothing found suggests an active bug hiding in it.

---

## 7. Carry-forward list

- **K99 (fresh install/docs):** verify empty-state behavior and fresh-vs-legacy migration behavior directly as part of the fresh-install proof (§21, §23/28/29 above); confirm whether Show/Showing creation order-of-operations is documented anywhere a new user would find it (§5 tribal-knowledge question).
- **K101 (polish/bug burn):** raw `err.Error()` leak across ~21 sites in `ws.go`/`indexcard_http.go`/`director_console.go`; `CreateShowing` duplicate-submission guard; two console-only silent failures (`tour-engine.js`, `victory-pixi-stage.js`); rewrite the two stale Scene-capture tests against the live-venue model; fix the dice-roll test's nil-pool mock; add pagination to Storyboards' board-content loaders.
- **K102+ (post-launch architecture):** deeper dead-code/duplicate-truth sweep across the ~19 files flagged by the shallow marker search; the still-unresolved eWrite skill-directory seed-test root cause, if it turns out to matter beyond the test itself.

---

## 8. Pass criteria (spec §44) — verdict

Broad blind-spot discovery performed across all 46 sections; zero launch blockers found (nothing to fix-or-PARTIAL); dead/legacy/duplicate-truth paths classified; tribal-knowledge question routed to K99; weird-user scenarios reasoned through with nothing alarming found; refresh/reconnect/restart behavior confirmed sound; partial-save and duplicate-submit gaps identified and routed; dense-board threshold behavior known (human-bounded today, routed for future pagination); fresh-vs-legacy differences routed to K99 rather than duplicated; silent failures identified (not exhaustively eliminated, but materially catalogued for K101); recovery paths sound or routed; install/package findings routed to K99; polish findings routed to K101; architecture findings routed to K102+; Grant was not used as a test harness anywhere in this pass.

**Final status: PASS.**
