# Kernel Report Back — Kernel 71: Two-Punch Show Tickets, Character Participation, and Showtime

## Status

**PASS.** Implemented and verified in six phases (A: canonical resolver; B: ticket schema/transactions; C: Audition Hall + Stage Management UI; D: Character selection; E: short codes + `/showtime`; F: docs/security/alpha-gate). Deployed live 2026-07-16.

**Commit note**: Phase A's work (`backend/internal/participation/`, the `lookupVenueRole` fix in `main.go`, the dead-duplicate removal in `network/ws.go`, and the Kernel 70A reportback) was committed by the operator mid-session as part of commit `bef0a69` (alongside unrelated storage cleanup). Everything from Phase B onward — the ticket system, Character selection, short codes, `/showtime`, all UI, and this documentation — remains **uncommitted**, consistent with this project's practice of leaving a kernel's work for review before committing (Kernels 62, 65–69).

## Pre-implementation audit (spec §2)

Conducted via three parallel codebase audits before any code was written:

1. **Every source of role/access truth**: `location_memberships` (Kernel 66+, location-scoped, canonical for Show-Run/Show/Scene/Cue authority), `memberships` (older, location/production/venue-scoped, used only by `main.go`'s `lookupVenueRole`), `access_grants` (boolean venue/production reach, no role), `show_run_roster_members` (Show-Run-scoped, its own role vocabulary). Confirmed the exact bug: a Producer/Director whose only grant is `location_memberships` resolved as `viewerRole="none"` at `/api/world/*`, because `lookupVenueRole` read `memberships` alone.
2. **Which source becomes canonical**: see "Canonical participation rules" below.
3. **Legacy tables remaining compat storage**: `memberships`/`access_grants`, read only as a last-resort upgrade.
4. **Every ordinary Director path that could add a Player without consent**: Stage Management's roster "Add" control and Third Place's Headshot picker, both calling the same unilateral `showruns.AddRosterMember` with `role="player"`. A second, less obvious path was found and closed during implementation, not left as a known hole: `UpdateRosterMemberRole`'s PATCH could promote an *existing* non-player row to Player with no ticket at all.
5. **Client-supplied IDs trusted where they shouldn't be**: none found surviving in the shipped design — every ticket/Character-selection mutation resolves the subject from the authenticated session or a locked server-side row, never a client-supplied user/role/show-run ID (see Security section).

## Canonical participation rules

New package `backend/internal/participation/` (`ResolveParticipationContext`). Precedence, highest wins, nothing lower may downgrade a role already found:

1. Operator (`access.IsOperatorUser`) — full authority.
2. Active `location_memberships` row at the resolved location.
3. Active `show_run_roster_members` row at the resolved Show Run.
4. Any active `location_memberships` row at all (audience fallback).
5. **Only reached if 1–4 all yield "none"**: legacy `memberships`/`access_grants`, read as an additive upgrade only.

`main.go`'s `lookupVenueRole` now delegates to `participation.LegacyLookupVenueRole`, keeping its three `/api/world/*` call sites unchanged. Bug closure proven by two required tests in `participation/resolver_bugclosure_test.go`: a producer with only `location_memberships` resolves as `producer`, not `none`; a user with only a valid ticket-derived roster row (no `location_memberships`/`memberships`/`access_grants` at all) gets full venue entry.

## Legacy table compatibility strategy

No wholesale deletion, per locked scope. `memberships`/`access_grants` remain as last-resort reads inside the resolver. The ticket's second-punch transaction additionally writes a compatibility `access_grants` row (`ON CONFLICT DO NOTHING`, scoped by the Show Run's `production_id` since a Show Run has no single venue) for any not-yet-migrated reader — never read back by the resolver itself as a truth source. Three packages (`characters`, `assets/read.go`, `showings/review.go`) already independently UNION `location_memberships`+`memberships`; left as-is this kernel (already correct, just duplicated — deduplicating them is flagged for a future kernel, not required for this one's correctness). `assets/upload.go`'s `resolveProducerScope` is the one reader that unions neither table; flagged, not changed (no demonstrated bug against it).

## Ticket schema and transaction behavior

Migration `046_kernel71_show_tickets.sql`: `show_run_tickets(id, show_run_id, user_id, requested_role CHECK='player', initiated_by_side, player_punched_at/_by_user_id, director_punched_at/_by_user_id, status CHECK IN (pending_player, pending_director, valid, declined, withdrawn), message, roster_membership_id, timestamps)`, with a partial unique index on `(show_run_id, user_id, requested_role) WHERE status IN (pending_player, pending_director)` preventing conflicting active tickets.

New package `backend/internal/tickets/`: `RequestFromPlayer`/`InviteFromDirector` (first punch, either direction, grants nothing), `SecondPunch` (the atomic transaction — row-locks the ticket, verifies the punching side is the correct party, marks it valid, upserts exactly one Player roster row via the existing `show_run_roster_members` partial-unique-index `ON CONFLICT`, links the ticket, writes the compat `access_grants` row, commits), `Decline`/`Withdraw` (pre-validation exits), `ListMineAsPlayer`/`ListIncomingForDirector`.

Proven under genuine concurrency: 8 goroutines racing `SecondPunch` on the same ticket produce exactly one valid ticket, exactly one active roster row, zero errors on the losers (`TestSecondPunchIsIdempotentUnderConcurrency`).

## Direct roster-add path inventory (spec §6.2)

| Path | Disposition |
|---|---|
| Stage Management roster "Add" (role=player) | **Changed** — role=player removed from this control's options; replaced by a new ticket-based "Invite a Player" control |
| Stage Management roster "Add" (other roles) | Unchanged — direct-add remains ordinary for director/producer/crew/audience/guest/observer |
| Third Place Headshot picker | **Changed** — converted from a direct roster insert to `tickets.InviteFromDirector` (a Director's first punch) |
| `AddRosterMember` (Go function, any caller) | **Changed** — rejects `role="player"` for non-Operator actors outright, covering both a fresh insert and an upsert-promotion of an existing row |
| `UpdateRosterMemberRole` (Go function) | **Changed** — rejects promoting an existing row *to* `role="player"` for non-Operator actors |
| Operator override | Retained, explicit, both functions still permit it |
| Test fixtures | Unchanged in spirit — several pre-existing tests that called `AddRosterMember(..., "player", ...)` as fixture setup were rewritten to insert the roster row directly via SQL instead, since that's not the behavior under test in those files |

## Audition Hall workflow

New panels, not wired onto the existing (pre-Kernel-71, unrelated) venue-access stubs: "Request to Join a Show" (short-code lookup, explains Show-Run scope before submitting), "My Invitations" (Accept/Decline), "My Pending Requests" (Withdraw), "Accepted Show Runs" (links to a new Character-selection page, `character.html`). A distinction note separates "Venue access" from "Show ticket" language per spec §5.3.

## Stage Management wording changes

`roster.html` split into "Invite a Player" (ticket-based, explains "Nothing changes for them until they do") and "Add Crew / Production Role" (unchanged direct-add, player option removed). New "Incoming Tickets" panel with Approve/Decline, using the spec's required wording ("Approving lets this Player choose a Character and enter this Show Run's Shows immediately. This does not give them Director or Crew access." / "Closes this pending request without granting access.").

## Character selection/switch behavior

New column `show_run_roster_members.character_card_id` (migration `047`). New self-service endpoint `POST /api/show-runs/{id}/roster/me/character` (always acts on the caller's own row — no target-user parameter, closing spoofing risk by construction). Verified: ownership enforced (`not_authorized` for someone else's Character), free switching with no approval step, archived-Character fallback (an `is_deleted` selection is treated as unselected, both in the endpoint and in `world.LoadVenueSnapshot`'s `TheaterContext`). Deliberately independent of `active_user_characters` (site-wide) and `current_session_personas` (Session-scoped) — see current-state.md's Identity Hierarchy section for the full reasoning.

## Show short-code behavior

New column `shows.short_code` (migration `048`, not a repurposing of the existing per-show-run `slug`). Auto-generated at creation (4-char, confusable-excluding alphabet `ABCDEFGHJKMNPQRSTUVWXYZ23456789`, collision-retried, location-scoped uniqueness joined through `show_runs.location_id`). Editable via `PATCH /api/shows/{id}/short-code` (Director/Producer/Operator, `409 short_code_taken` on conflict, re-setting the same code is not a conflict). Resolved via `GET /api/shows/by-code?code=` — deliberately a query parameter, not a path segment (see Known Limitations/bugs-caught-during-this-kernel below) — returning only show/show-run/location identifiers.

## `/showtime` orchestration

New package `backend/internal/showtime/`. `Start`: resolves the Show by short code (authority-checked against the actor's own manageable location) → derives the venue (current-Scene placement's venue if set; else scans all staged placements — exactly one distinct venue is used automatically and set as the current Scene as a side effect, zero or multiple distinct venues triggers `needs_venue_choice` instead of guessing) → calls the existing, **unmodified** `shows.StartShowSession` → computes ready/needs-character player counts → returns the spec §9.3 message format. `Status` reports the same shape without mutating anything. `End` resolves the active Session and closes it via `showings.CloseBySession` plus a session-status update — deliberately touches nothing else, proven by a before/after snapshot test (`TestEndPreservesCurrentSceneRosterAndCharacterSelection`) that current Scene, roster membership, and Character selection are byte-identical after `/showtime end`.

## Mic-off and Discord-suggestion proof

`StartShowSession` never calls any mic-related function — confirmed by reading its full implementation, not merely asserting the response field. `MicOn` is hardcoded `false` in `showtime.StartResult`; the response message appends the literal required suggestion text. Verified live in `fresh-install.sh`: the `/showtime` start response is asserted to contain `/mic hot` and to never contain `"mic_on":true`.

## Show/Scene/Session distinction in implementation

`/showtime` never treats a Session as owning a Scene: venue derivation reads the Show's *persistent* `current_show_scene_placement_id` first (unchanged by Session start/end), and only falls back to scanning staged placements when none is set yet. `/showtime end` provably does not touch the Show's current-Scene pointer. This matches the Kernel 70/70A model unchanged — Kernel 71 adds no new coupling between Sessions and Scenes.

## Discord-native vs. in-app `/showtime` (confirmed with the user)

Before implementing Phase E, the user was asked whether `/showtime` needs to be typeable directly into a real Discord channel (requiring generalizing `identity/discord_mic.go`'s hardcoded single-command dispatch) or whether the in-app command UI + a Stage Management button is sufficient. **Confirmed: in-app only**, matching `/session`'s existing precedent. Implemented as a `Legacy: true` entry in `commands/registry.go` bound to `POST /api/showtime/control`.

## Security proof (spec §11)

- Anonymous ticket/Character-selection mutations → `401` (`requireAuthenticatedUser` on every handler).
- Unauthorized authenticated mutations → `403`/rejection: a Player cannot punch for another Player (`TestPlayerCannotPunchForAnotherPlayer`); a Director cannot approve outside their Show Run/location authority (`TestDirectorCannotApproveOutsideTheirAuthority`), verified against a genuinely non-Operator Director account in the live smoke test.
- Client-supplied role is constrained to Player at the schema level (`requested_role` CHECK) and ignored/re-derived everywhere else.
- Client-supplied user ID cannot substitute another subject — every "me" endpoint (`SelectCharacter`, ticket punches) resolves the subject from the session, never a body field.
- Client-supplied Character ID is verified to belong to the authenticated Player before selection (`not_authorized` otherwise).
- One punch creates no roster row (verified live); the second punch's `ON CONFLICT` makes a duplicate roster row or a double-processed concurrent punch structurally impossible (verified under real concurrency, not just by code inspection).
- Declined/withdrawn tickets grant nothing (verified).
- Show code lookup returns only identifiers, never roster/variables (verified both by code and live response inspection).
- Third Place cannot directly create a ticket punch or roster membership — its backend package has zero write path to either table; its frontend now calls the same gated ticket-invite endpoint Stage Management uses.
- Legacy viewer-role rows cannot override canonical authority — structurally guaranteed by the resolver's precedence order (step 5 only runs if steps 1–4 all yield "none").

## Automated test results

`go build`/`go vet`: clean. Full `go test -count=1 ./...`: **green**, zero regressions, including new `participation` (2 tests), `tickets` (7 tests, including the concurrency race), `shows/short_code_test.go` (3 tests), `showtime` (4 tests), plus new tests added to `showruns` (guard/promotion/character-selection, 3 tests) and `world` (theater-context archived-fallback, 1 test). Two real regressions were caught and fixed during this pass, not shipped: (1) a Go 1.22 `ServeMux` route collision between `GET /api/shows/by-code/{code}` and the existing `GET /api/shows/{show_id}/program`, caught by `fresh-install.sh` panicking the backend at startup — fixed by switching to a query parameter; (2) three pre-existing tests (in `cues`, `shows`, `showruns`) that called the now-gated `AddRosterMember(..., "player", ...)` directly, invisible in a cached `go test ./...` run and only surfaced by `-count=1` — all rewritten to seed their fixtures via direct SQL insert instead.

## Alpha-gate output

`scripts/test/alpha-gate.sh` run end-to-end: **Overall PASS** (TEST_DATABASE_URL set, test DB setup, go build/vet, go test, Node suites [tracked dice.test.js exception only], fresh-install smoke, git diff --check all PASS; manual/browser step correctly reported PENDING, not run by the script).

## Browser/manual proof

`scripts/smoke/fresh-install.sh --local` extended with real HTTP-driven proof against a live compiled backend instance, covering both required directions end-to-end:

- **Player-request direction**: a dedicated account requests to join → one punch creates no roster row (verified) → Producer approves (second punch) → ticket valid → roster row is `role_label: "Player"`, never `"Cast"` → Character selected.
- **Director-invitation direction**: Producer invites a fresh account → one punch grants nothing → invitee accepts → ticket valid.
- Direct role=player roster-add rejected for a genuinely non-Operator Director account (a disposable account was provisioned specifically for this, since the script's own bootstrap account is this environment's Operator and would otherwise exercise the override path instead of the ordinary-Director rejection).
- A declined ticket grants nothing and stays auditable.
- Full `/showtime` cycle: a Scene staged at Catharsis → `/showtime <code>` derives the venue automatically, starts the Session, leaves mic off, suggests `/mic hot` → `/showtime status` → `/showtime end` (technical Session only).
- Show-code lookup resolves the Show Run without exposing backstage data.

No browser/screenshot evidence — no browser automation tooling exists in this environment; this is the same substitution used since Kernel 65.

## Live deployment status

**Deployed live 2026-07-16.** DB backed up (`backups/victory_pre_kernel71_migrations_20260716_232811.dump`), migrations `046`-`048` applied to the production database (idempotent `IF NOT EXISTS`/`ADD COLUMN IF NOT EXISTS` re-run of the full `database/migrations/` directory, `000`-`045` no-op'd as already applied), `show_run_tickets`/`shows.short_code`/`show_run_roster_members.character_card_id` all confirmed present afterward. `victory-backend` rebuilt (`docker compose up -d --build backend`, the running image had predated this kernel's commit) and restarted; `/health` responds OK post-restart.

## Known limitations

- Audience tickets are not implemented (`requested_role` CHECK-locked to `'player'`, an explicit non-goal this kernel).
- No casting/attendance system beyond the one-time ticket event.
- Legacy `memberships`/`access_grants` tables were not removed (explicit scope decision) and three packages still independently UNION them instead of calling a shared resolver helper.
- `/showtime` is an in-app command, not a real Discord-native slash command (confirmed scope decision).
- No browser/screenshot evidence for any of this kernel's new UI (Audition Hall panels, Stage Management panels, Character-selection page) — HTTP-level proof substituted, per the standing gap this project has flagged since Kernel 65.

## Files changed

Committed by the operator mid-session (`bef0a69`, alongside unrelated storage cleanup): `backend/internal/participation/` (new), `backend/cmd/victory/main.go` (`lookupVenueRole` fix), `backend/internal/network/ws.go` (dead-duplicate removal), `Construction/OperatorLogs/kernel-70a-reportback.md`.

Still uncommitted (this pass):
- New: `backend/internal/tickets/` (types.go, tickets.go, http.go, tickets_test.go), `backend/internal/showtime/` (showtime.go, http.go, showtime_test.go), `backend/internal/shows/short_code.go`, `backend/internal/shows/short_code_test.go`, `database/migrations/046_kernel71_show_tickets.sql`, `047_kernel71_roster_character_selection.sql`, `048_kernel71_show_short_codes.sql`, `frontend/venues/audition-hall/character.html`, `Construction/OperatorLogs/kernel-71-reportback.md` (this file).
- Modified: `backend/cmd/victory/main.go` (Phase B–E routes), `backend/internal/commands/registry.go`, `backend/internal/cues/cues_test.go`, `backend/internal/showruns/{http,projection,roster,types,showruns_test}.go`, `backend/internal/shows/{http,shows,shows_test,types}.go`, `backend/internal/world/{snapshot,snapshot_test}.go`, `frontend/venues/{audition-hall/index,show-runs/roster,show-runs/show,third-place/index}.html`, `scripts/smoke/fresh-install.sh`, `Construction/{current-state.md,Dictionary.txt,roadmap.md,OperatorLogs/operator-log.md,OperatorLogs/operator-notes.md}`.

## Next recommended Kernel 72 scope

Visual Scene composition/capture remains the standing next recommendation. Independently viable smaller passes: repair the nine pre-existing `dice.test.js` failures; add browser screenshot evidence for Kernels 70/70A/71's stage and participation controls; deduplicate the three packages that independently UNION `location_memberships`+`memberships`. A live-deployment pass for Kernel 71 should explicitly re-verify migrations `046`–`048` reach production, per the lesson recorded after Kernel 70A's migration gap.
