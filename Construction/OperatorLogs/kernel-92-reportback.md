# Kernel 92 Report Back — Showtime Composition & Showing Scheduler

**Kernel spec:** `Kernel_92_Showtime_Composition_and_Showing_Scheduler.md` (provided directly, not committed to `Construction/Kernels/`)
**Status:** PASS for the confirmed scope (see §2); the true-Showtime and End-Showtime live-start/live-end paths are exercised by Go tests over a real database but not by a scripted browser click (see §6)
**Date:** 2026-08-19
**Deployed:** live, `victory.amurray.family` — migration 109 applied at boot with the usual pre-apply backup

---

## 1. What this kernel is

Before this kernel, a Director started a live performance by typing a 4-character Show short code into a manual panel (`frontend/venues/show-runs/show.html`) or the `/showtime <code>` command — no way to schedule a future performance with a human-friendly name, no chronological browser of past/upcoming performances, and the chat bridge ("mic") was a separate manual step nothing else ever triggered automatically. Kernel 92 composes this into one flow: **Showtime button → pick or schedule a Showing → SHOWTIME**, backed by one server-side domain operation shared by the GUI and the pre-existing `/showtime` command.

The central design finding, made during the required repository audit before writing any code: **no new "Showing" table was needed.** The existing `shows` table (Kernel 67) already models exactly what the kernel doc calls a "Showing" — a single scheduled/occurring performance instance, already carrying `ScheduledStartAt/EndAt`, a 4-char `ShortCode`, and a persistent current-Scene pointer that survives Session end — and a Show Run can already own many `Show` rows, which is precisely the "many scheduled performances" shape the doc asks for. The one real gap was a Director-facing nickname distinct from Title; everything else was extending the existing `backend/internal/showtime` package (`Start`/`Status`/`End`, Kernel 71) rather than building a parallel system.

---

## 2. Scope as implemented

All of the kernel doc's pass criteria (§60) except the two noted below:
- Showtime button, New Showing form (required nickname ≤50 chars, date/time, canonical UTC timestamp), Today/Upcoming/Historical age-banded browsing, short code visible but not required for selection.
- Preflight with a real blockers-vs-warnings split (incomplete Characters, an empty ticket count, and a not-ready chat bridge are warnings; a busy venue or no derivable venue are blockers) — Character completeness never blocks Showtime.
- One domain operation (`showtime.Start`/`showtime.End`) composes venue/Scene resolution, Session start, and chat-bridge connect/disconnect, used identically by the GUI and by `/showtime <code>`/`/showtime end`.
- Idempotent Start (`AlreadyLive`) and End (`AlreadyEnded`) — no duplicate Session is ever created by a repeated press.
- End Showtime preserves Show/Scene/roster/Character state (already guaranteed by the pre-existing `showtime.End`, extended rather than touched) and returns an Aftercare-eligible count for the GUI's reminder — Aftercare is never auto-sent.
- "House Mic" → "Chat Bridge" relabeled everywhere in user-facing text; `/mic` command name, the Discord slash command, and Go identifiers left unchanged for compatibility.

**Two deliberate placement/verification deviations, both explained in §5:**
- The Showtime button lives on the Director's Chair page's header toolbar, not the in-stage Kernel 89 live-stage toolbar the kernel doc's literal wording suggested — the Kernel 89 toolbar structurally cannot appear before a Session already exists.
- The browser proof (real production, real disposable account, 7/7 checks) verifies scheduling, browsing, and Preflight's real blocker/warning output, but deliberately does not press "GO TO SHOWTIME" for real, since that would have written live Session state onto a real production venue. The Go idempotency/composition tests exercise that path against a real test database instead.

---

## 3. What was built

**Database** — one migration:
- `backend/migrations/109_kernel92_showing_nickname.sql` — `shows.nickname` (nullable `TEXT`, `CHECK (char_length(nickname) <= 50)` added via a `pg_constraint`-guarded `DO $$` block since `ADD CONSTRAINT` has no `IF NOT EXISTS`, matching migration 084's precedent), backfilled from `Title` for every pre-Kernel-92 row.

**Backend:**
- `backend/internal/shows/showing_create.go` (new) — `CreateShowing`, the Kernel 92 "New Showing" validation wrapper (required, ≤50-char nickname; derives Title/Slug) over the existing `CreateShow`, which gained additive `Nickname`/`ScheduledStartAt`/`ScheduledEndAt`/`Status` fields on `CreateShowInput` (nil-safe, every pre-existing caller unaffected).
- `backend/internal/shows/showing_list.go` (new) — `ListShowingsForViewer`, one query across every Show Run the caller manages (`showruns.ListShowRunsVisibleToUser`, unmodified) bucketed in Go into Today/Upcoming/Historical age bands (`<7d, 7-30d, 31-90d, 91-180d, 181-365d, 366d-36mo, older`, exactly per spec), alphabetical within each band, with per-row liveness derived from a single batched `sessions` query (never a stored flag — no second liveness model).
- `backend/internal/showtime/showtime.go` (extended, not rewritten) — `Start`/`Status`/`End` now accept an optional `showID` alongside `shortCode` (`resolveShow` prefers the ID), `Start` calls the real chat-bridge connect and reports `AlreadyLive` on a repeat call without re-running `StartShowSession`, `End` reports `AlreadyEnded` and a real `AftercareEligibleCount`, and a new `Preflight` operation reuses `deriveVenueSlug` in a new `dryRun` mode so checking readiness never persists a current-Scene placement the way an actual Start does.
- `backend/internal/identity/chat_bridge.go` (new) — `TryTurnOnChatBridgeForVenue`/`TurnOffChatBridge`/`ChatBridgeReady`, exported wrappers around the pre-existing unexported `discordMicTurnOn`/`discordMicTurnOff` so `showtime` can drive the same Discord-thread mechanism `/mic` already uses, without introducing an HTTP-layer dependency between the two packages. Bridge failure is always a warning to `showtime.Start`, never a blocker — matches spec §14 vs §15.
- `backend/internal/shows/http.go` — `HandleShowingsCollection` (`GET`/`POST`), mounted at `POST/GET /api/showtime/showings` — **not** `/api/showings`, which was already registered by the unrelated Kernel 22 `showings` package (the live audience-visibility review routes); a real naming collision caught by `panic()` at boot on first deploy attempt, not by the audit.
- `backend/internal/showtime/http.go` — `HandleShowtimeControl` gained a `preflight` action and a `show_id` request field alongside the existing `short_code`; response payload renamed `mic_on`→`chat_bridge_on` (grepped the frontend first — nothing read the old key) and gained `already_live`/`already_ended`/`aftercare_eligible_count`.
- "House Mic" → "Chat Bridge" relabeled in every backend message string (`discord_mic.go`) and one existing test's assertion updated to match.

**Frontend:**
- `frontend/lib/stage-runtime/kernel92-showtime-panel.js` (new) — the Showtime popup: New Showing form, Today/Upcoming/Historical collapsible sections, Preflight rendering with blockers disabling SHOWTIME and warnings not, an End Showtime action, and an Aftercare "Send Aftercare? / Not Yet" reminder that calls the existing Kernel 89 aftercare-send endpoint directly (see §5 for why it doesn't reuse Kernel 89's own panel JS).
- `frontend/venues/directors-chair/index.html` — a **Showtime** button added to the header toolbar (see §5 for why here, not the in-stage toolbar) plus the new script include.
- `frontend/venues/show-runs/show.html` — one line of copy above the existing manual short-code/Start-Showtime/End-Showtime panel pointing at the new guided flow; the panel itself is left fully functional as a debug/fallback surface, unchanged.
- "House Mic" → "Chat Bridge" relabeled in the header status chip across `the-cave`, `first-theater`, `catharsis`, `middle-school-stage`, and `frontend/lib/discord-mic.js`.

---

## 4. Evidence

### Backend tests

```bash
cd /opt/victory/backend
go build ./...
go vet ./...
TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" go test ./...
```

Full suite green, `gofmt -l` clean on every touched file. New: 4 tests in `backend/internal/shows/showing_create_test.go` (nickname required, ≤50-char cap accepted/rejected, scheduled status/timestamp set, manage-authority enforced), 4 tests in `showing_list_test.go` (Today/Upcoming/Historical bucketing across all boundaries, alphabetical-within-band, Show-Run-scoped visibility, live-session-derived `IsLive`), and 8 new tests in `backend/internal/showtime/showtime_test.go` (start-by-show-ID equals start-by-short-code, idempotent double-Start creates no second `sessions` row, idempotent double-End reports `AlreadyEnded` rather than erroring, Preflight reports a busy-venue blocker, Preflight reports incomplete Characters as a warning not a blocker, Preflight's dry-run venue derivation never mutates `CurrentShowScenePlacementID`, End's `AftercareEligibleCount` is correct, and the pre-existing state-preservation test extended to cover the new fields). The `showing_list_test.go` tests needed two real fixes mid-write, not just green-on-first-try (see §5).

### Deployment

```bash
docker compose build backend && docker compose up -d backend
```

Migration 109 applied cleanly at boot with a pre-apply backup (`victory_pre_migrate_20260819_051647_1pending.dump`); Discord gateway reconnected normally; `/health` verified live afterward.

### Browser proof

Real Playwright run against `https://victory.amurray.family`, a disposable fixture-row Director account (per `kernel-maker-field-guide.md`'s Two-Browser technique — production password signup is closed), and a disposable Production/Show Run under the real `amurray-family` location. Script: `scripts/smoke/kernel92-showtime-scheduler-browser.js`; fixture setup: `scripts/smoke/kernel92-showtime-scheduler-fixture.sh`.

**7/7 checks passed:** Showtime button reachable and visible after hovering the hover-to-reveal header; popup opens with the New Showing form; nickname input is capped at 50 characters (`maxlength` attribute verified); a scheduled Showing appears immediately in Upcoming (real `POST /api/showtime/showings` write, real `GET` re-read — this incidentally also surfaced one of Grant's real pre-existing production Showings in the Historical band, confirming the list genuinely reads live data rather than only its own fixture); selecting the Showing renders a real Preflight (`Ready for Showtime`, venue/Scene/cast/ticket/Chat-Bridge fields, a real `no_venue_derivable` blocker and a real `no_tickets_issued` warning for this unstaged fixture Show); **GO TO SHOWTIME is correctly disabled** while that blocker is present. Screenshots in `/tmp/k92-screens/` (not committed — ephemeral proof artifacts).

Deliberately did not press GO TO SHOWTIME for real — the fixture Show has no staged Scene on purpose, so the blocker above is the safe, correct outcome to prove rather than a limitation of the proof. All fixture rows (user, session, location membership, production, show run, shows, sessions) were deleted afterward and verified at zero residue via a direct count query.

---

## 5. Findings and deviations, and why

**Finding 1 — the Kernel 89 in-stage toolbar cannot host a "begin the show" button; it structurally requires a show already being live.** `kernel89-director-tools.js`'s toolbar only renders once `window.VictoryStageKernel88Bridge.getShowID()` returns a non-empty value, and that bridge's `getShowID` is `() => currentSnapshot?.session?.show_id || ""` (`frontend/lib/stage-runtime/runtime.js`) — it needs an already-live session snapshot. Putting the Showtime button there, as the kernel doc's "top-right tool stack" wording suggested at a glance, would mean the button only appears after the thing it's meant to start has already started some other way — self-defeating. The button instead lives on the Director's Chair page (`frontend/venues/directors-chair/index.html`), the one venue-independent Director surface, reachable regardless of live-session state. This was found during the repository audit, not after building the wrong thing.

**Finding 2 (deviation) — the Aftercare reminder calls the send endpoint directly rather than reusing Kernel 89's own panel.** Kernel 89's `openAftercarePanel`/`renderAftercarePanel` are already exported on `window.VictoryKernel89DirectorTools` for exactly this kind of reuse, but that whole script is only loaded on live-stage venue pages (`the-cave`, `first-theater`, `catharsis`) — not on Director's Chair, where the Showtime popup lives. The reminder therefore calls `POST /api/shows/{id}/aftercare/send` directly with a minimal Send/Not-Yet UI of its own. This still satisfies "reuse the existing Kernel 89 operation, don't rebuild the domain logic" — only the presentation is duplicated, by necessity of the two features living on different pages, and only a two-button confirm, not the cohort-targeting UI.

**Finding 3 — `POST/GET /api/showings` was already taken.** The kernel doc's own §11 warned "if the current code belongs to Show rather than Showing, reconcile carefully rather than duplicating concepts," but the collision that actually mattered was at the URL layer: the pre-existing Kernel 22 `showings` package already owns `GET /api/showings` (live audience-visibility review, an unrelated concept — see `backend/internal/showings`). `main.go`'s `net/http` mux panics on route registration conflicts, which is exactly what happened on the first `docker compose up -d backend` of this kernel. Fixed by mounting Kernel 92's endpoints at `/api/showtime/showings` instead, with a comment at both registration sites explaining the two "showing" concepts are unrelated on purpose.

**Finding 4 — a shared test database accumulates real state across every kernel that ever ran against it, and a naive test can silently read someone else's fixtures instead of its own.** `showing_list_test.go`'s first version asserted an exact alphabetically-sorted 3-item list for one Show Run; it failed with dozens of unrelated leftover Shows mixed in ("K87v2 Show", "K88 Fixture Show...", "K90 Visibility Show..." — real fixture rows from prior kernels' own test runs, never cleaned up, because `location_memberships` at `producer` role grants `CanManage` over *every* Show Run at that shared location `amurray-family`, not just the ones a given test itself created). Fixed by filtering result assertions to the test's own `ShowRunID` and excluding the pre-Kernel-92 base fixture Show's own empty-nickname row, rather than assuming the list would only ever contain what the test created. A second, related bug in the same test: two Showings scheduled a few hours apart from "now" landed in different buckets (Today vs Upcoming) depending on what time of day the test happened to run, since a same-calendar-day offset is inherently wall-clock-dependent — fixed by using >48h offsets that are always Upcoming regardless of when the suite runs.

**Finding 5 (operational, not code) — `docker exec <container> psql ... <<'SQL' ...` silently does nothing without `-i`.** Cleaning up the browser-proof fixture's first attempt via a `docker exec victory-postgres psql ... <<'SQL'` heredoc produced no output and no error, and the rows were still present afterward — `docker exec` does not forward stdin to the container process unless `-i` is passed, so the heredoc never reached `psql` at all, and psql with no input on a TTY-less non-interactive invocation just... exits, silently. `docker exec -i` fixed it immediately. Recorded in `operator-notes.md` as a reusable gotcha.

---

## 6. Known gaps, honestly stated

- **The live-start/live-end path (pressing SHOWTIME for real, with a staged Scene, against a real venue) is proven by Go tests over a real database, not by a scripted browser click.** The browser proof stops at Preflight's blocker being correctly enforced, deliberately, to avoid writing live Session state onto a real production venue with a disposable throwaway fixture. A follow-up scripted proof using a genuinely disposable venue/Scene fixture (rather than any of the three real production venues) would close this gap the same way `kernel90fixture`/`k89fixture` do for other kernels' live-write proofs.
- **The `/showtime <code>` and `/mic` command paths were not re-exercised end-to-end after these changes** — confirmed correct by code inspection (both are unmodified request shapes, additive fields only) and by the pre-existing `showtime_test.go` tests still passing unmodified against the new signatures, but not by re-running the actual command bar in a browser.
- **No Producer/Crew/Operator-specific Showtime affordance was built or considered** — the kernel doc scopes this to Director/Producer-authorized users via the existing `showruns.CanManageShowRun`, unchanged, and that's what shipped.
- **The three duplicate Director-authority-check implementations found during the audit** (`showruns.CanManageShowRun`, `network.canAccessDirectorConsole`, `identity.discordMicCanControl`) were **not** consolidated — new Kernel 92 code exclusively calls `CanManageShowRun`, matching `showtime.go`'s pre-existing convention, but the other two call sites are untouched. Flagged as optional future cleanup, not part of this kernel's scope.

---

## 7. How to run from a clean state

1. Ensure `.env` has `POSTGRES_PASSWORD` set.
2. `docker compose up -d postgres` (already running in production).
3. `docker compose build backend && docker compose up -d backend` — migration 109 applies automatically at boot with a pre-apply `pg_dump` backup to `/opt/victory/backups/`.
4. Frontend needs no build step — `/opt/victory/frontend` is bind-mounted directly into the shared Caddy container.
5. Sign in as a Director/Producer/Operator, visit `/venues/directors-chair/`, hover the header, click **Showtime**.

---

## 8. Files changed or created

**Backend:** `backend/migrations/109_kernel92_showing_nickname.sql`; `backend/internal/shows/{types.go, shows.go, http.go, showing_create.go, showing_create_test.go, showing_list.go, showing_list_test.go}`; `backend/internal/showtime/{showtime.go, http.go, showtime_test.go}`; `backend/internal/identity/{chat_bridge.go, discord_mic.go, discord_mic_test.go}`; `backend/cmd/victory/main.go`.

**Frontend:** `frontend/lib/stage-runtime/kernel92-showtime-panel.js` (new); `frontend/venues/directors-chair/index.html`; `frontend/venues/show-runs/show.html`; `frontend/venues/{the-cave,first-theater,catharsis,middle-school-stage}/index.html` (label only); `frontend/lib/discord-mic.js`.

**Scripts:** `scripts/smoke/kernel92-showtime-scheduler-browser.js` (new); `scripts/smoke/kernel92-showtime-scheduler-fixture.sh` (new).

**Docs:** this reportback; `operator-log.md`; `operator-notes.md`; `current-state.md` top section.

---

## 9. Required project-memory updates completed

- [x] Reportback saved in repository (this file)
- [x] `operator-log.md` appended
- [x] `operator-notes.md` updated (URL-collision + shared-test-DB + `docker exec -i` lessons)
- [ ] `kernel-maker-field-guide.md` — not updated; no repo-layout/test-command/runtime-mode change this kernel introduced
- [ ] `dev-workflow.md` — not updated; no startup/port/service change beyond the one new migration file, which follows the existing convention exactly
- [x] Master/current-state guide updated (`current-state.md` top section)
- [x] Fresh-install/bootstrap migration list — automatic (`//go:embed *.sql`), no manual step
- [ ] Help/command documentation — not applicable; `/showtime`/`/session`/`/mic` command names, verbs, and endpoints are all unchanged, only response fields/copy changed

---

## 10. Next recommended step

A scripted Playwright proof of the true live-start/live-end path (pressing SHOWTIME for real, End Showtime, the Aftercare reminder's Send/Not-Yet branches) against a genuinely disposable venue/Scene fixture rather than any of the three real production venues — the one claim in this kernel proven by Go tests over a real database but not yet by a scripted browser click, matching how `k89fixture`/`k90fixture` close the equivalent gap for other kernels' live-write proofs.
