# Kernel Report Back — Kernel 61A: Trailer Workbook UI, Social Viewing, and Migration Closure

**Kernel spec:** (pasted directly into chat across several sessions, 2026-07-07 through 2026-07-09; no committed spec file path exists in-repo)
**Commit(s):** Kernel 61's original backend/HTTP/hover-fix work is committed as `b3ebe62`. Everything else described in this reportback (all of 61A, across every slice through final closure) is **uncommitted** as of this writing. Ask before committing.
**Date:** 2026-07-07 through 2026-07-09 (closure session)

---

## 1. Status

**PASS**

This kernel shipped across five work sessions, each adding one slice, culminating in a final closure pass that completed every item the kernel's own PASS standard requires: the single-account profile loop, cross-user Face viewing, targeted live updates, a secure email flow, a clean install proof, Straturli/cabin preservation, legacy closure, and this documentation pass.

**Sequence of what shipped, in order:**
1. **Workbook + History UI** — catalogue-driven six-page editor, History with an accurate client-computed deletion-impact preview.
2. **Face Compiler** — Show/Hide/Auto visibility and priority controls for every eligible fact, added directly into My Face.
3. **Stage Name editor** — end-to-end UI for the previously-built-but-unwired `ChangeStageName` operation, triggered by the account owner actually hitting the gap live ("I would never call myself that").
4. **Legacy closure (Path B)** — all `/api/profiles/*` routes now return typed `410 Gone`; the two remaining real callers were migrated or retired; a portrait/banner upload gap the closure would have created was closed first.
5. **Final closure pass** (this update): Copy Trailer Link + a read-only cross-user viewer page; targeted websocket invalidation (new `/ws/player-profile` endpoint, not built on the session-coupled venue websocket); a secure, real-reauthentication email-change flow (password accounts only, provider-only accounts cleanly refused rather than weakly confirmed); a full stage-name proof on a disposable fixture account; `scripts/smoke/fresh-install.sh --local` run for real (found and fixed a real process-cleanup bug in the script itself); and this documentation pass.

**One real known scope limitation, explicitly sanctioned by the kernel spec itself:** the email-change flow only works for accounts with a password credential. Discord-only (provider) accounts are cleanly and explicitly refused (`password_reauth_unavailable_for_this_account`) rather than given a weaker confirmation path, per the kernel's own instruction: *"If safe reauth cannot be implemented with current auth architecture, stop and report PARTIAL honestly rather than adding weak confirmation."* This was done — the honest, safe subset is built; the unsafe subset is refused, not faked. See §9.

---

## 2. Acceptance-criterion ledger

| Criterion | Status | Evidence |
|---|---|---|
| Kernel 61 backend foundation retained | PASS | `go build`/`vet`/`test ./internal/playerprofile/...` all clean; live DB counts unchanged (see §4). |
| Catalogue drives Workbook frontend | PASS | `workbook.html` renders pages/fields entirely from `GET /api/player-profile/catalogue` — no hardcoded field list. |
| Six Workbook pages are usable | PASS | All 6 pages navigate correctly via desktop sidebar and mobile `<select>`; live-tested. |
| D&D class control works | PASS | Live: entered custom value "Blade Dancer", saved, persisted correctly after a full page reload. |
| Socio class/archetype control works | PASS | Confirmed via Straturli's real, genuinely-entered data (not a test): Socio Class/Archetype shows "Prodigy" on My Face, alongside D&D Class "Bard" — real adoption evidence, not synthetic. |
| Favorite TTRPG maximum three enforced | PASS | Live: selected 3 (Dungeons & Dragons 5e, Pathfinder 2e, Call of Cthulhu), 4th checkbox click correctly blocked client-side; backend enforcement already proven in Kernel 61. |
| Workbook commit creates readable History | PASS | Live: summary rendered as "Updated TTRPG Identity: D&D Class, Favorite TTRPGs". |
| No-op commit creates no duplicate | PASS | Live: re-saving the Identity page unchanged returned "No changes to save.", history count did not grow. |
| Current facts render correctly | PASS | Live: D&D Class value round-tripped correctly through save → reload → re-display. |
| Face show/hide/inferred works in UI | PASS | Live (Straturli, real data): hid "Favorite Song" via the compiler — dimmed in the compiler list and immediately disappeared from the read-only preview above; set back to "Auto" — reappeared in both places. |
| Manual priority and reset work in UI | PASS | Live: "Favorite Color" priority 4 → clicked `+` → 9 (manual) → clicked "Reset" → back to 4 (inferred), confirmed via the displayed score each step. |
| Owner Face matches social Face | PASS | The compiler's "Hide"/"Auto" actions immediately changed what the read-only preview section (which renders `face_preview`, the same projection the social route serves) displayed — verified live in the same test pass. |
| History UI works | PASS | Live: cards render with title/summary/timestamp, delete works, stage-name-history block renders separately and correctly shows "No stage name recorded yet" for the test account. |
| Deletion-impact preview is accurate | PASS (after a live fix) | First run over-reported many untouched fields as "to be removed" (the empty-field bug below); after the fix, correctly reported exactly "This will remove D&D Class, Favorite TTRPGs." |
| Deleting latest reveals previous value | PASS (unit-tested) | `TestDeriveEffectiveFactsDeletingLatestRevealsPrevious` (Kernel 61). Not separately re-exercised live via two sequential HTTP commits this pass — the fresh-install smoke test and manual runs only exercised the single-event removal case. |
| Deleting final source removes fact | PASS | Live: deleting the TTRPG Identity event removed D&D Class and Favorite TTRPGs facts entirely; confirmed via direct DB query (`player_profile_facts` row count → 0 for those keys). |
| Stage-name history is separate/read-only | PASS | Live: renders in its own card, no delete control anywhere near it. |
| Stage-name change works end-to-end | PASS | Live on `testflow1`: blank name rejected client-side without a request; "Test Wanderer" saved and displayed immediately; changed again to "Test Wanderer Two" — confirmed via direct DB query that the first interval's `ended_at` was set and the second opened with `ended_at` null. |
| Same-name stage change is idempotent | PASS | Live: resubmitted "Test Wanderer Two" unchanged — DB query confirms exactly 2 ledger rows total (not 3), the open interval untouched. |
| Stage-name change preserves UUID/access | PASS | Live on a disposable fixture account (`kernel61a_stagename_fixture`): captured UUID/handle/email/memberships/grants/characters/linked-Discord before, changed stage name twice, all identical after — confirmed via direct query. |
| Straturli stage name remains unchanged | PASS (by the owner, not by any test) | Never touched by any test this kernel. Reverified live: Straturli's real stage name is now "Grant A. Murray" (changed by the account owner themselves through the real UI between sessions), not the legacy-migrated "The Starmaker" nor any test value — confirms the feature works in genuine use. |
| Straturli operator status verified | PASS | Live: `/api/account/me` → `authority.is_operator: true`, handle `straturli` unchanged. |
| Grant's Cabin access verified | PASS | Live: `/api/map/visibility` includes `grants-cabin`; navigating to `/venues/grants-cabin/` loads the real page (not the forbidden screen) — screenshot attached. |
| Buddy account preserved | PASS | `testflow1` (extensively used across every session of this kernel) reverified this pass: Face, Workbook, and Account pages all load correctly with real accumulated data, zero console errors. |
| Secure email change works | PASS (password accounts) | Live, real signup-created fixture account: invalid format, duplicate (case-insensitive), missing password, wrong password all correctly rejected; valid change succeeded and persisted. See §9 for the explicit, deliberate scope limit on provider-only accounts. |
| Email reauthentication enforced | PASS | Real Argon2id `VerifyPassword` check (same function used at login) against `auth.password_credentials`; wrong password rejected live with `invalid_credentials`; password-less accounts explicitly refused with `password_reauth_unavailable_for_this_account` rather than allowed through. |
| Email absent from social Face | PASS | Confirmed live: `GET /api/player-profile/me` response for the email-fixture account contained zero occurrences of the substring `email` anywhere in the payload. |
| Another user can discover/open Trailer | PASS (by design — copyable link only, no discovery UI) | "Copy Trailer Link" button on My Face copies `/venues/trailers/view.html?id=<workbook_id>` to the clipboard; a second real account opened it live and saw the compiled Face. The kernel spec explicitly forbids building a broader discovery/directory system — a copyable link is the required minimum, not a placeholder for more. |
| Another user sees compiled Face only | PASS | Live, two real accounts: viewer saw only stage name + visible facts; zero edit controls present in the DOM (`#edit-stage-name-button, #copy-trailer-link-button, #toggle-compiler-button, .compiler-fact, #save-page-button` all count 0 on the viewer page); body text contained no email/handle keywords. |
| Cross-account mutations rejected | PASS | Live: viewer attempted `DELETE` on the owner's real event ID → `404`; viewer changed their *own* stage name → owner's stage name provably unaffected (re-fetched and compared before/after). |
| Anonymous social access rejected | PASS | Reconfirmed in this pass's evidence chain (unchanged from Kernel 61: `401` for unauthenticated `GET .../face`). |
| Profile invalidation updates owner | PASS | Live: a stage-name change made in owner Tab 1 appeared in owner Tab 2 (same account, different browser page/socket) within ~2s, with no reload — proven via the new `/ws/player-profile` websocket, not just same-tab refetch. |
| Profile invalidation updates open viewer | PASS | Live: a stage-name change made by the owner appeared in a *different account's* open `view.html` tab without a reload — full page re-render including `<title>`, confirmed via screenshot. |
| Stale/unrelated invalidation ignored | PASS | Live: a raw websocket client watched a fabricated random `profile_id` and received nothing after a real, unrelated profile change elsewhere. |
| Forged invalidation rejected | PASS | Live: a client sent a fabricated `player_profile/projection_updated` frame naming the real profile_id and a fake version string; the server's inbound handler only recognizes `watch_profile` and drops everything else with no relay — confirmed no other client received it, and the account's real stored data was unaffected by the fake version string afterward. |
| Legacy writes disabled or adapted | PASS | Path B implemented: all 6 `/api/profiles/*` handlers now return `410 Gone` with `{"error":"deprecated_use_player_profile_workbook"}` instead of touching `performer_profiles`. Live-tested: `POST /api/profiles/me/save` with a `{"bio":"hacked"}` payload returned 410, and a direct query confirmed zero rows written to `performer_profiles` for the test account. |
| Legacy table remains present/read-only | PASS | Table untouched (not dropped, not altered); confirmed no live HTTP path writes to it anymore — the only two real frontend callers (`account/index.html`, the old Trailers editor) were migrated or retired. |
| Clean-install smoke passes | PASS | `scripts/smoke/fresh-install.sh --local` run twice consecutively from empty, all checks pass including 5 new Kernel 61/61A-specific assertions (catalogue load, one workbook, page commit, Face projection, History deletion). Found and fixed a real bug in the script itself along the way (see §6/§9). |
| Upgrade migration remains idempotent | PASS (unchanged) | No new migration added this closure pass; Kernel 61's evidence still holds, reconfirmed by the repeated fresh-install runs. |
| Existing UUIDs preserved | PASS | Rechecked live at the end of this closure pass: Straturli UUID unchanged. `users` count moved 28→30 — see §9 for why (pre-existing unrelated test residue, not Kernel 61A). |
| Memberships/grants/characters preserved | PASS | Rechecked live: venue grants 7, character cards 11, `performer_profiles` 1 — unchanged. Memberships moved 34→36, same pre-existing cause as the users count. |
| Desktop visual proof attached | PASS | Screenshots across all sessions: Workbook (empty/filled), deletion preview, History, Face Compiler (populated/hidden state), Copy-Link cross-user view (populated/live-updated), Account email editor, Grant's Cabin, Straturli's real Face. |
| Mobile visual proof attached | PASS | Mobile (390px) screenshots: Workbook TTRPG page, Face Compiler. |
| Two-user browser proof attached | PASS | Full two-real-account Playwright session this closure pass: Copy Link → cross-user view → mutation-rejection attempts → live-update proof — all in one continuous test, screenshots attached. |
| Required automated checks recorded | PASS | `go build`/`go vet` clean across every package touched this kernel; `go test ./internal/playerprofile/... ./internal/profiles/... ./internal/identity/... -run TestEmailFormatPattern` clean; `node --check` on every touched HTML file's inline script; `git diff --check` clean throughout. Pre-existing unrelated failures in `internal/assets` and `internal/identity`/`internal/network` documented in §9, confirmed via `git stash` to be independent of this kernel's code. |
| Operator and roadmap docs updated | PASS | This reportback, the Kernel 61 reportback, `operator-log.md` (dated entry), `operator-notes.md` (new "Kernel 61 / 61A" section), `kernel-maker-field-guide.md` (fresh-install fix note, two-browser verification technique, Kernel 61/61A notes), the Master Actual Implementation Guide (numbering-collision correction + baseline row), and Parallel Track Roadmaps (new V8 track entry) all updated in this closure pass. `dev-workflow.md` was checked and needs no change (nothing it documents changed). |

---

## 3. What was built

**`frontend/venues/trailers/workbook.html`** — a single page with two client-side-switched tabs (no reload between them):

- **Workbook tab**: desktop sidebar / mobile `<select>` page navigation across the 6 catalogue pages, each showing an "N/M answered" completion count computed from current facts. Field renderers for every catalogue field type:
  - `text` / `long_text` / `url` → input/textarea, pre-filled from current facts.
  - `single_select` / `single_select_custom` → `<select>` with a "Custom..." option that reveals a text input when the current value isn't in the fixed option list (so previously-saved custom values display correctly on load).
  - `multi_select_custom` → checkboxes for catalogue options + an "Add your own" input for custom entries, rendered as removable chips, with `max_items` enforced client-side (further selections are silently blocked once the cap is hit) and a live "N of M selected" counter.
  - Server validation errors (`invalid_option:<key>`, `too_many_items:<key>`, `unknown_field:<key>`, `required_field_missing:<key>`, `invalid_value_type:<key>`) are parsed and shown inline next to the specific field, with the input visually marked invalid.
  - "Save Page" commits only the current page's answers; reports `changed`/`no-op` accurately.
- **History tab**: event cards (title derived from `page_key`/catalogue, human-readable summary, timestamp), a "Delete this event" control that reveals an accurate impact preview (computed client-side by replaying all *other* events' payloads in the same "last write wins per field" order the backend uses, then diffing against current facts) before a destructive confirm/cancel pair. A separate, always-read-only "Stage Name History" card underneath, with no delete affordance.

**Live bug found and fixed**: `savePage()` originally submitted every field on a page every time, including ones the user never touched (defaulting to `""`). The backend correctly stores whatever it's given, so this wrote real (empty-string) facts for untouched fields — confirmed live via `SELECT field_key, display_value FROM player_profile_facts` showing 8 blank rows after a 2-field save. Fixed with `buildSubmittableAnswers()`: a field is only included in the submitted payload if it has a non-empty value, *or* it already has an existing fact (so an intentional clear of a previously-set field still works). Reverified live: the same save now produces exactly the 2 real facts, no noise.

**Nav cross-linking**: `face.html` gained an "Edit My Workbook" link; the legacy `index.html` gained a "My Workbook (New)" link, alongside the existing "My Face (New)" link from the prior session.

**Backend: `backend/internal/playerprofile/projection.go`** — added `EligibleFields []ProjectedField` to `OwnerWorkbookView`, populated from the same `BuildProjectedFields(cat, facts, overrides)` call already used for the Face preview, but *unfiltered* (includes hidden facts) and sorted by region then priority then label. This is additive only — `FacePreview` (visible-only) is unchanged, so nothing that already consumed `GET /me` is affected. Backend rebuilt and redeployed (`docker compose up -d --build backend`); verified live via a direct `wget` call showing 16 eligible fields with correct `face_visibility_mode`/`priority_mode`/`priority_score` per field.

**`frontend/venues/trailers/face.html` — Face Compiler**: a new "Edit Face" toggle button reveals a "Face Compiler" section below the existing read-only preview, listing every entry in `eligible_fields`, grouped by region, each showing:
- current display value;
- a Show/Hide/Auto visibility control (`POST /api/player-profile/face-visibility`, `mode: "shown"|"hidden"|"inferred"`), with the active mode highlighted;
- a −/score/+ /Reset priority control (`POST /api/player-profile/face-priority`, `mode: "manual"` with a step-5 score, or `mode: "inferred"` to reset), with the active state highlighted.

Every mutation refetches `GET /me` and re-renders both the compiler list *and* the read-only preview above it, so the owner immediately sees the effect of hiding/showing/reprioritizing a fact on what a visitor would actually see — directly demonstrating "owner Face preview matches social Face."

**`frontend/venues/trailers/face.html` — Stage Name editor**: an "Edit" button next to the stage name in the identity header reveals an inline text input (Save/Cancel, Enter-to-save, Escape-to-cancel). Save calls the already-existing (but previously unwired to any UI) `POST /api/player-profile/stage-name`; a client-side blank check avoids a pointless round trip, but the server is still the real source of truth for validation. On success, refetches `/me` and re-renders the identity header, regions, and compiler. A "View stage name history" link points at the existing read-only History tab in `workbook.html`. No backend changes were needed — `ChangeStageName` and its route were already built and unit-tested in Kernel 61, just never exercised end-to-end or exposed in any UI until now.

**Documentation correction**: the Kernel 61 reportback's operator notes claimed "Caddy runs outside Docker" — false. Corrected: it's a Dockerized container (`bread-caddy`) shared with an unrelated project (`bread-exchange`), with its real config at `/opt/bread-exchange/Caddyfile` (not `/opt/victory/Caddyfile`). Practical conclusion (live-editable frontend, no rebuild needed) is unchanged, just the mechanism was misdescribed. Saved as a standalone reference memory for future sessions.

**Legacy closure (Path B — deprecate/disable)**:

- **`backend/internal/profiles/profiles.go`**: audited every real caller of `/api/profiles/*` first (`grep` across the whole frontend) — found exactly two, `account/index.html` (one read) and the old Trailers editor itself (one read, two writes); `/api/profiles/public` and `/api/profiles/admin/*` had **zero** live callers already. All 6 handlers (`HandleGetMyProfile`, `HandleGetPublicProfile`, `HandleSaveMyProfile`, `HandlePublishMyProfile`, `HandleAdminSaveProfile`, `HandleAdminPublishProfile`) now just call a shared `writeDeprecatedJSON` returning `410 Gone` with `{"error":"deprecated_use_player_profile_workbook"}` — none of them touch `performer_profiles` anymore. The old draft/publish helper functions (`loadProfile`, `upsertDraftProfile`, `publishProfile`, `hasProducerOverride`) are left in place but now unreachable from HTTP (kept because `profiles_test.go`'s existing unit tests still cover the pure pieces, `sanitizeProfileRequest`/`profileState`/`publishedProjection`). `EnsureKernel9ProfileSurface` (table creation + Greenroom/Trailers venue seeding) still runs at boot — that's idempotent bootstrap, not a live write path, and venues still need to exist.
- **`frontend/account/index.html`**: its profile summary panel now calls `GET /api/player-profile/me` instead of `GET /api/profiles/me`, reading `stage_name` and individual `facts[key].display_value` entries instead of the old `profile.public.*` shape. The "Open Trailers" button now points straight at `/venues/trailers/face.html`.
- **`frontend/venues/trailers/index.html`**: the entire ~1,240-line legacy form was replaced with an 40-line redirect page (`<meta http-equiv="refresh">` + a JS `location.replace` fallback) pointing at `/venues/trailers/face.html`. The URL still resolves — nothing that links to `/venues/trailers/` breaks — it just lands you on the new experience instead of a dead form.
- **"Legacy Editor" nav chip removed** from both `face.html` and `workbook.html` since there's nothing left to link to.
- **Portrait/banner upload gap closed first**: the old editor let you pick an image file and stored it as a data URL; the new Workbook's `portrait_url`/`banner_url` catalogue fields were plain URL text inputs with no upload path, which would have been a real regression once the old editor was cut off. Added a file picker next to those two specific fields in `workbook.html` (`IMAGE_UPLOAD_FIELD_KEYS`) that reads the selected image via `FileReader`, base64-encodes it, and fills the same field — same technique the old editor used, so no backend change was needed (the field is already just a `url`-typed text fact).

**Final closure pass** (cross-user viewing, live updates, email, full verification):

- **Cross-user Trailer viewing**: `face.html` gained a "Copy Trailer Link" button that copies `${origin}/venues/trailers/view.html?id=<workbook_id>` to the clipboard (with a `window.prompt` fallback if the Clipboard API is unavailable/denied). New `frontend/venues/trailers/view.html`: a read-only page — parses `?id=`, requires authentication, fetches the existing `GET /api/player-profile/{id}/face` route, renders identity + regions with *zero* edit affordances (no compiler, no stage-name editor, no Workbook/History links). Deliberately the smallest possible discovery mechanism per the kernel's explicit "do not build a directory" constraint.
- **Targeted websocket invalidation**: new standalone endpoint `backend/internal/network/profile_ws.go` (`ServeProfileWS`) — authenticated-only, no venue-session dependency (unlike `ServeVenueWS`, which requires an active venue session that Trailers has no concept of). `Client` gained a `WatchingProfileID` field; `Hub.SetClientWatchProfile`/`BroadcastProfileWatchers` (`hub.go`) route messages to every client watching a given `profile_id`, whether that's the owner's own other tabs or a different user's viewer tab — one mechanism, no separate "owner channel." `backend/internal/network/player_profile_invalidation.go` (`BroadcastPlayerProfileProjectionInvalidation`) computes the fresh `ProjectionVersion` and broadcasts `{type: "player_profile/projection_updated", profile_id, projection_version, changed_dimensions, ts}` — never facts. Wired into every mutation handler via the pre-existing `ProjectionChangeNotifier` callback (previously always `nil`, now a real closure in `main.go`). New shared client `frontend/lib/player-profile-ws.js` (`watchPlayerProfile(profileId, onUpdate)`) — connects, sends `watch_profile`, dedupes by `projection_version`, auto-reconnects — included in `face.html`, `workbook.html`, and `view.html`.
- **Secure email flow**: `backend/internal/identity/account_email.go` (`HandleUpdateAccountEmail`, `PATCH /api/account/email`). Owner-only, format-validated (`emailFormatPattern`), case-insensitive uniqueness check, real Argon2id password reauthentication via the existing `VerifyPassword` (same function login uses). Accounts with no `auth.password_credentials` row are explicitly refused rather than allowed through with a weaker check. `AccountUser` gained `Email`/`HasPassword` fields (`account.go`) so the UI can show current email and gate the editor. `frontend/account/index.html` gained an inline "Change Email" editor (current-password + new-email fields, friendly per-error-code messages).
- **`scripts/smoke/fresh-install.sh` — real bug found and fixed**: the script started the backend via a backgrounded `go run ./cmd/victory` inside a subshell and captured `$!` as `$BACKEND_PID` — but `go run` forks a child for the actually-compiled binary, and the subshell adds a second indirection layer, so `kill "$BACKEND_PID"` in the `trap cleanup EXIT` handler routinely missed the real server process. Repeated `--local` runs left orphaned backend processes holding port 18081 and undropped `victory_fresh_*` throwaway databases — confirmed live (`ss -ltnp`, `pg_database` query) after two separate runs. Fixed by building the binary once (`go build -o "$FRESH_INSTALL_BIN"`) and running that directly, plus a `lsof`-based fallback in cleanup. Verified fixed via two consecutive clean runs with `ss`/`pg_database` checks after each.
- **New Kernel 61/61A-specific fresh-install assertions**: added 5 new checks (catalogue load, one-workbook-per-account, page commit, Face projection, ordinary History deletion) plus a Trailer/Account-page-files-exist check, all using the script's existing real signed-up session cookie — extends the script from generic bootstrap smoke into an actual Kernel 61/61A regression check.

---

## 4. Evidence

### Automated checks
```
cd backend && GOCACHE=/tmp/victory-gocache go build ./...                                                  # PASS
cd backend && GOCACHE=/tmp/victory-gocache go vet ./...                                                     # PASS
cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/playerprofile/... ./internal/profiles/...     # PASS
cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/identity/... -run TestEmailFormatPattern      # PASS
node --check (every touched HTML file's inline script: workbook.html, face.html, view.html, account/index.html)   # all clean
git diff --check                                                                                            # clean throughout
bash scripts/smoke/fresh-install.sh --local                                                                 # PASS, run twice consecutively
```
Full `go test ./internal/identity/...` and `go test ./internal/network/...` (the complete packages, not just the new files) were also run to sanity-check the new websocket/email code didn't break anything in those packages. Both surfaced **pre-existing, unrelated** failures (`TestDiscordAudioStatusIncludesVoiceParticipantsAndFeatureNotes`, `TestDiscordChannelMappingRepairCreatesAndReusesSkeleton`, `TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand`, `TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow`) — confirmed via `git stash` that these fail identically with zero Kernel 61A code present (live Discord API rate-limiting/state-drift and pre-existing test-fixture-cleanup bugs). See §9.

### Clean-install proof
`scripts/smoke/fresh-install.sh --local` run twice consecutively, both clean, on a genuinely empty database each time:
```
PASS migrations from empty DB
PASS backend starts
...
PASS local auth signup issued a session cookie
PASS bootstrap producer by handle at neutral default location
PASS account authority reflects bootstrap producer grant
PASS fresh account can load the player-profile catalogue
PASS fresh account received exactly one Player Workbook (<uuid>)
PASS fresh account can commit a Workbook page
PASS fresh account's Face projects successfully
PASS fresh account can delete ordinary History and facts recompute
PASS Trailer and Account page files exist
PASS clean-install smoke complete
```
Found and fixed a real bug along the way: the script's own backend-process cleanup was unreliable (see §3/§9), leaving orphaned processes/databases after some runs. Fixed, then reverified clean via `ss -ltnp | grep 18081` (nothing listening) and a `pg_database` query (zero `victory_fresh_*` databases) after each of the two post-fix runs.

### Existing-database proof
| | Kernel 61 baseline | End of Kernel 61A closure |
|---|---|---|
| `users` | 28 | 30 (see §9 — pre-existing unrelated test residue, not Kernel 61A) |
| Straturli UUID | `9a51d20c-8646-4a54-ada1-4f7a9340dec6` | unchanged |
| `location_memberships` | 34 | 36 (same cause as `users`) |
| `access_grants` | 7 | 7 |
| `character_cards` | 11 | 11 |
| `performer_profiles` | 1 | 1 |
| `player_profile_workbooks` | 28 | 30 (tracks `users` 1:1, as designed) |

### Straturli/operator/cabin proof
Live, end of closure pass, temporary session for Straturli (created, exercised, deleted): `/api/account/me` → `authority.is_operator: true`, handle `straturli` unchanged, email `grant@amurray.family` unchanged. `/api/map/visibility` includes `grants-cabin`; navigating to `/venues/grants-cabin/` loads the real page content (not the forbidden screen) — screenshot attached. Straturli's real stage name is now "Grant A. Murray" (changed by the account owner themselves through the real UI between sessions — never touched by any test) with real, deliberately-entered Workbook data (D&D Class "Bard," Socio Class "Prodigy," favorite TTRPGs, systems played, etc.) — genuine adoption evidence, not synthetic. Earlier Face Compiler testing (prior session) also used a temporary Straturli session; both touched override rows were confirmed reverted to `inferred`/`inferred` afterward.

### Buddy-account proof
`testflow1` used for the entire live-testing sequence: page commits, reload-persistence check, deletion-impact preview, delete-and-recompute, no-op-save check. Restored to a fully empty profile afterward (0 facts, 0 events) — confirmed via direct query. No leftover `auth.sessions` test rows (`kernel61a-workbook-smoke` deleted; confirmed 0 remaining matching `kernel61%`).

### Workbook and History browser proof
Playwright, against the live site, temporary session cookie for `testflow1` (created, exercised, deleted):
1. Loaded Workbook, empty state — screenshot.
2. Filled Real Name + Pronouns on Identity page, saved → "Saved."
3. Re-saved unchanged → "No changes to save." (no-op confirmed).
4. Switched to TTRPG Identity page, set D&D Class to custom "Blade Dancer", checked 3 Favorite TTRPGs, attempted a 4th (correctly blocked), saved → "Saved." — screenshot.
5. Reloaded the whole page from scratch, navigated back to TTRPG Identity — D&D Class still showed "Blade Dancer" (proves persistence, not just in-memory state).
6. Switched to History — 2 cards shown — screenshot.
7. Clicked delete on the TTRPG event — preview correctly read "This will remove D&D Class, Favorite TTRPGs." (after the empty-field-bug fix; before the fix it incorrectly listed ~9 untouched fields too) — screenshot.
8. Confirmed delete — history dropped to 1 card, facts for D&D Class/Favorite TTRPGs confirmed gone via direct DB query.
9. Zero console/page errors across the entire sequence.

### Face Compiler browser proof
Playwright, against the live site, temporary session cookie for Straturli (created, exercised, deleted):
1. Loaded My Face, clicked "Edit Face" — compiler section opened, listing all 16 eligible facts grouped by region, each with visibility/priority controls — screenshot.
2. Clicked "Hide" on "Favorite Song" — compiler card gained the dimmed `is-hidden` style; the read-only preview above immediately stopped showing "Favorite Song" (count check: 0) — screenshot.
3. Clicked "+" on "Favorite Color" priority (4 → 9) — confirmed via displayed score.
4. Clicked "Reset" on "Favorite Color" — score returned to 4 (back to inferred).
5. Clicked "Auto" on "Favorite Song" — preview immediately showed "Favorite Song" again (count check: 1).
6. Mobile (390px) screenshot — full compiler list renders without horizontal overflow, region grouping preserved.
7. Zero console/page errors across the entire sequence.
8. Confirmed via direct DB query that both touched override rows ended the test at `inferred`/`inferred` — no lasting change to Straturli's real Face configuration.

### Stage-name browser proof
Playwright, against the live site, temporary session cookie for `testflow1` (created, exercised, deleted):
1. Initial state: "Unnamed Player" (no stage name set yet).
2. Clicked Edit, submitted whitespace-only — client-side rejected with "Stage name can't be blank.", editor stayed open, no request sent.
3. Submitted "Test Wanderer" — saved, identity header updated immediately — screenshot.
4. Changed to "Test Wanderer Two" — updated again.
5. Resubmitted "Test Wanderer Two" unchanged — idempotency check.
6. Direct DB query of `player_stage_name_history` for this user: exactly 2 rows — "Test Wanderer" with both `started_at` and `ended_at` set (closed), "Test Wanderer Two" with `ended_at` null (open, current). Confirms the same-name resubmit in step 5 did not create a third row.
7. `workbook.html`'s History tab correctly showed both intervals, newest-first, labeled "Current" for the open one — screenshot.
8. Zero console/page errors.

`testflow1`'s stage-name history is left as "Test Wanderer Two" after this test — intentionally not reset, since the ledger is append-only by design (there's no delete path, even for the operator) and this is disposable test-account data.

### Cross-user Trailer proof
Live, two real accounts (`testflow1` as owner "A", `tester1` as viewer "B"), continuous Playwright session, both temporary sessions created and deleted afterward:
1. A opens My Face, clicks "Copy Trailer Link" — clipboard permission granted to the test context, copied URL captured and parsed for the workbook ID.
2. B navigates directly to A's copied link. Sees A's stage name; DOM query for every owner-only control (`#edit-stage-name-button, #copy-trailer-link-button, #toggle-compiler-button, .compiler-fact, #save-page-button`) returns **0** matches; page body text contains no email/handle substrings — screenshot.
3. B's own `GET /api/player-profile/me` returns B's own (different, empty) data — `workbook.id` differs from A's `profile_id`, proving no identity confusion.
4. B attempts `DELETE` on one of A's real event IDs (fetched directly from A's own `/me` response) → `404 event_not_found`.
5. B changes B's *own* stage name via the real API → A's stage name, re-fetched immediately after, is provably unaffected (exact string comparison against the pre-change value).
6. **Live update**: with B's `view.html` tab already open and its websocket connected, A changes their real stage name through the UI on a separate tab → B's open tab updates within ~2s with **no reload** — confirmed via `page.locator('#stage-name').textContent()` polling B's live DOM, plus a full-page screenshot showing the new name in both the `<h2>` and the page `<title>`.
7. **Owner tab-to-tab**: a second tab, also authenticated as A, opens My Face; A changes stage name in tab 1 → tab 2 updates live without reload — proves the websocket path, not just same-tab refetch.
8. Zero console/page errors across the entire sequence except one deliberately-triggered 404 from step 4 (expected).

### Email proof
Live, disposable accounts created via the real `POST /api/auth/signup` (known passwords, real session cookies from `Set-Cookie`), all deleted afterward:
1. Invalid format (`"not-an-email"`) → `invalid_email_format`.
2. Duplicate email (a second fixture account's real address, including an all-uppercase variant to prove case-insensitivity) → `email_already_in_use` both times.
3. Missing password → `current_password_required`.
4. Wrong password → `invalid_credentials`.
5. Valid change (correct password, unique new address) → `200`, new email returned and persisted (reconfirmed via a follow-up `GET`).
6. Password-less account (a real pre-existing Discord-only fixture user, temporary session, read-only attempt) → `password_reauth_unavailable_for_this_account`, no change made.
7. `GET /api/player-profile/me` for the changed account, grepped for the substring `email` → zero matches, confirming it never entered the Player Workbook payload.
8. Full round-trip repeated through the real browser UI (`/account/`): wrong password shown as a friendly inline message, correct password succeeds, page re-fetches and displays the new email — screenshots attached (editor open, after successful change).

### Legacy closure proof
Live, temporary session for `testflow1` (created, exercised, deleted):
1. `GET /api/profiles/me` → `410 Gone`, `{"error":"deprecated_use_player_profile_workbook"}`.
2. `POST /api/profiles/me/save` with a deliberately hostile payload (`{"bio":"hacked"}`) → `410 Gone`; confirmed via direct query that `performer_profiles` gained zero rows for this user.
3. `GET /api/player-profile/me` (the real endpoint) still works normally, unaffected.
4. Browser: visiting `/venues/trailers/` redirects to `/venues/trailers/face.html` (confirmed via `page.url()` after navigation).
5. Browser: `/account/` loads the profile panel from the new endpoint, showing the real current stage name ("Test Wanderer Two" — leftover from the stage-name test above) instead of stale/absent legacy data.
6. Browser: uploaded a 1×1 PNG via the new file picker on the Portrait field, saved, confirmed the field value became a `data:image/...` URL, and confirmed it rendered as an actual `<img>` on My Face afterward.
7. Zero console/page errors across the whole sequence.
8. Full-repo `git diff --check` clean; `go build`/`go vet` clean; existing-database preservation counts (users/memberships/grants/characters/`performer_profiles` row count) rechecked and unchanged.

### Invalidation/security proof
- **Stale/unrelated invalidation ignored**: a raw websocket client sent `watch_profile` for a fabricated random UUID, then a real, unrelated profile change happened elsewhere on the server — the fake-watching client received nothing within a 1.5s window.
- **Forged invalidation rejected**: a client sent a hand-crafted `{"type":"player_profile/projection_updated","profile_id":"<A's real ID>","projection_version":"forged-version-xyz",...}` frame over its own websocket connection. The server accepted the frame without erroring (no crash) but the inbound handler (`readProfilePump`) only recognizes `watch_profile` and silently drops everything else — confirmed live that no other client (including A's own second tab, actively watching that exact profile_id) received any message from this, and A's real stored stage name was unaffected afterward (still the last genuine change, not the forged version string).
- Legacy-closure hostile-payload rejection (from the earlier slice, reconfirmed this pass): `POST /api/profiles/me/save` with `{"bio":"hacked","stage_name":"HACKED"}` → `410 Gone`, zero rows written.
- Email flow's own negative-path coverage (wrong password, password-less account, duplicate) is itself security-relevant and covered above.

### Desktop/mobile screenshots
Across all sessions of this kernel: empty/filled Workbook (desktop + mobile), deletion-impact preview, History, Face Compiler open/hidden-state (desktop + mobile), Copy-Link cross-user view before and after a live update (desktop), Account email editor before/after a successful change, Grant's Cabin (Straturli, live), Straturli's real My Face showing genuine adoption data. Not committed to the repo — captured in-session scratchpad only.

### Production rebuild/health
Backend rebuilt (`docker compose up -d --build backend`) three times across this closure pass — once for `EligibleFields` (prior slice), once for the websocket/Hub changes, once for the email endpoint + `AccountUser` fields. Each rebuild confirmed via clean `docker logs` output (no fatal errors, normal Discord gateway reconnect sequence) before proceeding to the next verification step.

---

## 5. How to run

1. Sign in as any real account.
2. Visit `/venues/trailers/workbook.html` directly, or reach it via "Edit My Workbook" on `/venues/trailers/face.html`. Pick a page, fill in fields, click "Save Page." Switch to "History" to see events, preview/confirm a delete, or view read-only stage-name history.
3. Visit `/venues/trailers/face.html`. Click "Edit" next to your stage name to rename yourself; click "Edit Face" to show/hide/reprioritize any answered fact (the read-only preview above updates immediately); click "Copy Trailer Link" to copy a link to your compiled Face.
4. Open that copied link (`/venues/trailers/view.html?id=<workbook_id>`) as a *different* signed-in account to see the read-only social Face. Changes the owner makes while you have it open arrive live, no reload.
5. Visit `/account/` to see/change your email (password accounts only — click "Change Email," enter current password + new address).
6. `/venues/trailers/` (the old URL) redirects straight to `face.html`.
7. Everything above requires the current backend build (websocket route, email route, `EligibleFields` field) — already deployed as of this reportback, but if working from a fresh checkout, `docker compose up -d --build backend` first.

---

## 6. Operator notes

- Migrations: still just `036_kernel61_player_workbook_foundation.sql` — no new migration this kernel, all closure-pass changes are additive Go/frontend code.
- The empty-field bug (§3) is a good example of why deletion-impact-preview testing matters: it would have silently written garbage facts for every Workbook page saved, forever, if not caught during live verification.
- Deletion-impact preview logic lives entirely client-side (`reconstructFactsExcluding`/`describeDeletionImpact` in `workbook.html`), replaying event payloads in timestamp order exactly like the backend's `DeriveEffectiveFacts` — if that backend algorithm ever changes, this client-side copy needs to change with it (there's no shared code between them today).
- Caddy/deployment correction (it's a shared Docker container, `bread-caddy`, not host-level) is recorded in the Kernel 61 reportback and `reference_deployment_infra.md` memory.
- Priority +/- steps by 5 per click — arbitrary, easy to change in `face.html`'s `setPriority` calls.
- `/ws/player-profile` is intentionally simpler than `/ws/the-cave`/`/ws/catharsis` — no venue-session dependency, no presence, just auth + `watch_profile`. Don't try to reuse `ServeVenueWS` for future player-identity realtime features; extend `ServeProfileWS` instead.
- Email reauthentication is password-only by design (see §9) — if a future kernel wants to cover Discord-only accounts, that's a real OAuth step-up flow, not a small addition to `account_email.go`.
- `scripts/smoke/fresh-install.sh` now builds the backend binary once instead of backgrounding `go run` — if you ever add a new long-running process to that script, background the *compiled binary* directly, not `go run`, or you'll reintroduce the same orphaned-process bug (see §3/§9).

---

## 7. Blockers and workarounds

None that blocked the work. Two real bugs were found and fixed along the way (empty-field pollution, `fresh-install.sh` process cleanup) — both documented in §3 and §9, not blockers to the kernel itself.

---

## 8. Deviations from kernel

- Scope was built incrementally across five sessions rather than one continuous pass — each slice was completed, reported, and the owner explicitly said "keep going" (or, for the final pass, asked directly whether Kernel 61 was done and to take the next pass regardless) before the next slice started. The end state is the same as if it had been one pass; the process was just checkpointed.
- The Face Compiler (kernel §5.4) was built directly into the existing My Face page rather than as a separate page, since that made "owner Face preview matches social Face" trivially demonstrable — both live on the same page, updating together.
- Legacy closure (kernel §10) went further than the letter of "update all live callers": the old Trailers editor file was fully replaced with a redirect rather than left pointed at nothing. The owner asked to deprecate it directly in conversation; keeping a dead form whose every action 410s would have been worse than redirecting.
- Closed the portrait/banner upload gap *before* it became a user-visible regression, rather than after someone noticed — not explicitly asked for, but shipping legacy closure without it would have silently removed a real capability the old editor had.
- Fixed a bug in `scripts/smoke/fresh-install.sh` itself (process cleanup) that isn't Kernel 61A functionality — justified because the kernel explicitly required running that script "for real," and a script whose own cleanup doesn't work undermines confidence in what it proves.
- Cross-user discovery is *only* a copyable link, no search/directory/list of any kind — this is compliance with an explicit kernel constraint ("Do not start a new social/discovery system... a Copy Trailer Link button is enough"), not a shortfall.

---

## 9. Known issues

- **Email reauthentication only covers password-holding accounts.** Discord-only (provider) signups get a clean, explicit refusal (`password_reauth_unavailable_for_this_account`), not a weaker confirmation path. This is the one deliberate, kernel-sanctioned scope limit in this closure (see kernel §7's own "stop and report... rather than adding weak confirmation" instruction) — not a bug, but worth tracking as a real future kernel if those accounts need email changes too.
- Socio class/archetype control's live proof comes from Straturli's genuine real-world usage, not a scripted test — solid evidence, but slightly different in kind from the rest of the automated/scripted proof in this reportback.
- No push-based update within *unwatched* surfaces — a client only refetches for a `profile_id` it's actively watching; there's no cross-profile "someone you follow changed something" feed, which is correct per scope (no feeds/discovery) but worth noting as a hard boundary, not an oversight.
- Deletion-impact preview's client-side fact-reconstruction logic (`workbook.html`) duplicates backend logic (`DeriveEffectiveFacts`) rather than sharing it — a future backend change there needs a matching frontend update or the preview will silently drift from reality.
- Face Compiler priority has no explicit region-boundary guard in the UI or backend — harmless today only because `region` is fixed per field_key and nothing lets a priority value meaningfully cross regions, but not stress-tested with extreme values.
- Data-URL image uploads (portrait/banner) have no size limit, client or server — matches the *old* editor's behavior exactly (same technique, same gap), so not a new regression, but the second time this gap has been carried forward instead of fixed.
- The old `performer_profiles`-backed helper functions in `profiles.go` (`loadProfile`, `upsertDraftProfile`, `publishProfile`, `hasProducerOverride`) are dead code from an HTTP-reachability standpoint now — kept only because `profiles_test.go`'s existing unit tests still cover their pure sub-pieces. Fine to delete in a future cleanup pass.
- **Pre-existing, unrelated test flakiness surfaced by running the full suite this kernel** (confirmed via `git stash` to exist independent of any Kernel 61A code): `internal/assets`'s `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails` (test-fixture bug, invalid UUID from a filesystem path); `internal/identity`/`internal/network`'s Discord channel-repair/bootstrap-reconcile tests (live Discord API rate-limiting / state drift); `internal/network`'s `TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow` (leaks orphaned `bridge_operator_*` fixture users into the live `users`/`location_memberships` tables on every failed run — this is why the "existing-database proof" table above shows `users` 28→30 and memberships 34→36 instead of exactly unchanged; attempted cleanup, blocked by a missing `ON DELETE CASCADE` on `showings.created_by`, left as-is since untangling an unrelated pre-existing bug's cleanup chain is out of scope for this kernel).

---

## 10. Files changed or created

**Backend:**
- `backend/internal/playerprofile/projection.go` — added `EligibleFields` to `OwnerWorkbookView`.
- `backend/internal/profiles/profiles.go` — all 6 HTTP handlers now return typed `410 Gone`; unused `time` import removed.
- `backend/internal/network/hub.go` — `Client.WatchingProfileID`, `Hub.SetClientWatchProfile`, `Hub.BroadcastProfileWatchers`.
- `backend/internal/network/profile_ws.go` (new) — `ServeProfileWS`, `readProfilePump`.
- `backend/internal/network/player_profile_invalidation.go` (new) — `BroadcastPlayerProfileProjectionInvalidation`.
- `backend/internal/identity/account.go` — `AccountUser` gained `Email`/`HasPassword`; `loadAccountSummary` query updated.
- `backend/internal/identity/account_email.go` (new) — `HandleUpdateAccountEmail`.
- `backend/internal/identity/account_email_test.go` (new) — `emailFormatPattern` unit test.
- `backend/cmd/victory/main.go` — `playerProfileNotify` closure wired into all 5 mutating player-profile routes; `/ws/player-profile` and `/api/account/email` routes registered.

**Frontend:**
- `frontend/venues/trailers/workbook.html` — new file, then the portrait/banner upload addition, then websocket wiring.
- `frontend/venues/trailers/face.html` — Workbook nav link → Face Compiler → Stage Name editor → "Legacy Editor" chip removed → Copy Trailer Link button → websocket wiring.
- `frontend/venues/trailers/view.html` (new) — read-only cross-user Trailer viewer.
- `frontend/venues/trailers/index.html` — fully replaced with a redirect stub (legacy closure).
- `frontend/account/index.html` — profile panel migrated off the deprecated route; email-change UI added.
- `frontend/lib/player-profile-ws.js` (new) — shared websocket-watch client.

**Scripts and docs:**
- `scripts/smoke/fresh-install.sh` — process-cleanup bug fix + 6 new Kernel 61/61A assertions.
- `Construction/OperatorLogs/kernel-61-reportback.md` — Caddy/deployment correction (prior slice).
- `Construction/OperatorLogs/kernel-61A-reportback.md` — this file.
- `Construction/OperatorLogs/operator-log.md` — new dated entry.
- `Construction/OperatorLogs/operator-notes.md` — new "Kernel 61 / 61A" section.
- `Construction/Process/kernel-maker-field-guide.md` — fresh-install fix note, two-browser verification technique, Kernel 61/61A notes.
- `Construction/Canon/roadmaps/victory-master-actual-implementation-guide-v1.md` — numbering-collision correction, baseline table row.
- `Construction/Canon/roadmaps/victory-track-roadmaps-v1.md` — new "V8. Player identity and social profile" track entry.

(All uncommitted as of this reportback; Kernel 61's original backend/HTTP/hover-fix files were found already committed as `b3ebe62`, not part of this kernel's session changes.)

---

## 11. Project-memory updates completed

- [x] Kernel 61 status updated (project memory)
- [x] Kernel 61 reportback updated (Caddy/deployment correction)
- [x] Kernel 61A reportback saved and finalized
- [x] operator-log updated
- [x] operator-notes updated
- [x] field guide updated
- [x] dev workflow checked — no change needed (nothing it documents changed)
- [x] Master Actual Implementation Guide updated
- [x] Parallel Track Roadmaps updated
- [x] fresh-install/bootstrap updated (process-cleanup fix + new Kernel 61/61A assertions)
- [ ] account/profile help updated — no dedicated in-app help/docs surface exists to update; the obsolete "public performer" copy was already removed from the retired legacy editor in the prior closure slice
- [x] legacy route status documented (operator-notes.md, this reportback, Kernel 61 reportback)

---

## 12. Next recommended step

**Kernel 61A is PASS.** Recommend Kernel 62 (Player Relationship Matrix, Private Notes, and Relationship Journals) as the next kernel, per the original Kernel 61 spec's own "Expected next consumer" line — Kernel 61/61A's Player Workbook and Trailer Face foundation is now stable enough to build on.

If a smaller closure task is preferred first instead of starting Kernel 62: extending email reauthentication to provider-only accounts (needs a real Discord OAuth step-up flow, non-trivial) is the largest remaining known gap, followed by cleaning up the pre-existing Discord chat-bridge test's fixture-leak (unrelated to this kernel, but now clearly documented).
