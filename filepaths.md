# Victory — Repository Map

Current as of **Kernel 89** (2026-08-17). Paths are clickable. Agents should start their referencing here.

The previous version of this file was a flat link dump from an editor picker, last accurate around Kernel 4 — it stopped at migration `004` and listed six backend packages out of the thirty-four that exist. This version is organized by what each area *is*, so it can be read as well as clicked.

For vocabulary see [Dictionary.txt](Construction/Dictionary.txt). For current-state canon see [current-state.md](Construction/current-state.md). For durable implementation traps see [operator-notes.md](Construction/OperatorLogs/operator-notes.md).

---

## Backend — [backend/](backend)

Go. Entry point [main.go](backend/cmd/victory/main.go) registers every HTTP route and the WebSocket endpoints; [victory-bootstrap](backend/cmd/victory-bootstrap) is the operator CLI for granting producer authority.

### Identity, authority, access

| Path | What it is |
|---|---|
| [access/](backend/internal/access) | Venue visibility, operator checks, session-cookie→user resolution. [kernel16_venue_bootstrap.go](backend/internal/access/kernel16_venue_bootstrap.go) is the legacy every-boot venue seed; since Kernel 77A **a migration alone is the canonical way to add a venue** (migrations auto-apply before every `Ensure*` bootstrap — see 083/085 for the pattern). |
| [identity/](backend/internal/identity) | Signup/login/logout, Discord OAuth, invites, permission requests, productions. |
| [participation/](backend/internal/participation) | Kernel 71's canonical resolver reconciling `location_memberships`, roster rows, and legacy `memberships`. |
| [tickets/](backend/internal/tickets) | Two-punch Show tickets — the only ordinary path to a Player roster row. |
| [ratelimit/](backend/internal/ratelimit) | Credential-endpoint rate limiting. |

### Show structure

| Path | What it is |
|---|---|
| [showruns/](backend/internal/showruns) | Show Runs, roster, authority helpers (`CanManageShowRun`, `CanViewBackstage`, `CanCrewPerformNonDestructiveEdit`). |
| [shows/](backend/internal/shows) | Shows, short codes, [stage.go](backend/internal/shows/stage.go) — the **single write path** for the shared current Scene. |
| [scenes/](backend/internal/scenes) | Scene Library, Show Scene Placements, and [composition.go](backend/internal/scenes/composition.go) — visual Scene composition + element↔interaction bindings. |
| [cues/](backend/internal/cues) | Director/Crew GO cues. [types.go](backend/internal/cues/types.go) records why per-participant visibility was deferred — read before adding a second object model. |
| [showtime/](backend/internal/showtime) | `/showtime` orchestration: resolve code → derive venue → start/resume Session. |
| [showings/](backend/internal/showings) | Showing lifecycle and review. |

### Live runtime

| Path | What it is |
|---|---|
| [world/](backend/internal/world) | [snapshot.go](backend/internal/world/snapshot.go) — `LoadVenueSnapshot`, the one composer of what any viewer sees, including the role filter and theater context. |
| [network/](backend/internal/network) | WebSocket [hub.go](backend/internal/network/hub.go) and [ws.go](backend/internal/network/ws.go). **The hub does no role filtering** — role scoping must be applied when choosing recipients. |
| [actions/](backend/internal/actions) | The durable `actions` event log: chat, OOC, dice, game events, index cards, tokens. |
| [sessions/](backend/internal/sessions), [venues/](backend/internal/venues) | Session lifecycle; venue map/grid config. |
| [dice/](backend/internal/dice) | Canonical server-side dice. |

### Characters and players

| Path | What it is |
|---|---|
| [characters/](backend/internal/characters) | Character cards, workbook, skills, activation. |
| [playerprofile/](backend/internal/playerprofile), [playerrelationships/](backend/internal/playerrelationships), [profiles/](backend/internal/profiles) | Player Workbook, My People, profile surfaces. |
| [thirdplace/](backend/internal/thirdplace) | Headshot Commons. |
| [messages/](backend/internal/messages) | Durable mailbox; [backstage_notes.go](backend/internal/messages/backstage_notes.go) is Kernel 74's Directors+ note surface. |

### Gameplay packets — Kernels 73/74

| Path | What it is |
|---|---|
| [merchant/](backend/internal/merchant) | Equip Mode, Kessa, `participant_interactions`. Kernel 89 added merchant *authoring* over the same canonical `merchant_packets` tables ([kernel89_packet_authoring.go](backend/internal/merchant/kernel89_packet_authoring.go)) and Director-triggered Aftercare *delivery* ([kernel89_aftercare_send.go](backend/internal/merchant/kernel89_aftercare_send.go) — delivery only; it writes no `aftercare_*` row). `ResolveEligibleContext` ([interactions.go](backend/internal/merchant/interactions.go)) is the **one Player-eligibility gate** every participant action goes through. [tutorial_flow.go](backend/internal/merchant/tutorial_flow.go) holds Kernel 74's door/Ra flow — the package name is narrower than its contents, explained in that file's header. |
| [tutorial/](backend/internal/tutorial) | Five CHECK-constrained tutorial milestones, keyed (user, **character**, show). |
| [dialogue/](backend/internal/dialogue) | Bounded guided-dialogue packets; prerequisite graph validated acyclic on load. |
| [projection/](backend/internal/projection) | Participant-local stage projection — substitutes *which Scene a viewer resolves*, never writes the shared current Scene. |
| [stageobjects/](backend/internal/stageobjects) | **Kernel 90: Victory's one answer to "who can currently perceive or interact with this thing on stage?"** Canonical object identity (`Ref{kind,id}`), the `stage_object_states` store, and the viewer `Projector`. `ApplyMutation` is the **single** state-write path — manual Director controls and Cue actions both call it, which is what makes their parity structural rather than a coincidence. Deliberately a **leaf package**: it takes its Director gate and its WS notifier by injection ([main.go](backend/cmd/victory/main.go)) because `world` imports it while `shows → network → world`, so importing `showruns` or `network` here would close a real cycle. Not fog-of-war, not an ACL system, not a rule engine. |

### Director preparation and theatrical announcement — Kernel 89

| Path | What it is |
|---|---|
| [directorprep/](backend/internal/directorprep) | Kernel 89's bounded preparation store (`director_preparations`, migration 105). **Director+ end to end — there is no Player read path in the package at all**, which is how "hidden prep must not leak" is satisfied structurally. `ValidatePayload` is the per-kind typed validator that drops unknown keys, and it is the reason the JSONB payload is not an executable surface. |
| [announcements/](backend/internal/announcements) | The theatrical announcement palette as a **pure leaf package** — no DB, no authority, no delivery. `Compose` is the one validator both a live announcement and a saved preset pass through. Nothing here reads a die, and `announcements_test.go` machine-checks that no two styles are distinguished by colour alone. Delivery lives in `network/ws.go`'s `announce/push` case, as a Kernel 86 Stage Effect. |
| [stageeffects/](backend/internal/stageeffects), [rollaudience/](backend/internal/rollaudience) | Kernel 86's ephemeral effect registry and server-side audience resolution, reused unchanged by Kernel 89's announcements. |

### eWrite — Kernel 78

| Path | What it is |
|---|---|
| [ewrite/](backend/internal/ewrite) | The writing/publication/reading system: typed collection tree, publications with append-forward revisions and 409 save-conflict protection, stable section anchors + aliases, named editor grants, typed object links, Postgres FTS search, per-publication export. [markdown.go](backend/internal/ewrite/markdown.go)/[markdown_policy.go](backend/internal/ewrite/markdown_policy.go) are the repo's **only** Markdown render+sanitize path (goldmark + bluemonday, server-side only). Design notes in [Construction/eWrite/](Construction/eWrite). |

### Schema and infrastructure

| Path | What it is |
|---|---|
| [migrations/](backend/migrations) | **105 migrations**, `000`–`105` (numbering has one gap), embedded via [embed.go](backend/migrations/embed.go) and auto-applied at boot with a checksum ledger and pre-apply backup. Never edit a shipped migration; always confirm the next number against both this directory and the live `schema_migrations` table. |
| [migrate/](backend/internal/migrate) | The embedded runner. |
| [dbtest/](backend/internal/dbtest) | Kernel 64's **safety gate**: `ValidateTestDatabaseURL` refuses any database not obviously a test database. Every DB-touching test must call it first. |
| [assets/](backend/internal/assets), [commands/](backend/internal/commands), [config/](backend/internal/config), [db/](backend/internal/db) | Warehouse assets, command palette, config, pool. |

---

## Frontend — [frontend/](frontend)

Static, served by Caddy. No build step, no package.json.

### Shared stage engine — [lib/stage-runtime/](frontend/lib/stage-runtime)

Kernel 72 replaced First Theater's and Catharsis's separately-copied runtimes with this one engine, configured per venue by a small `venue.js`.

| File | What it is |
|---|---|
| [runtime.js](frontend/lib/stage-runtime/runtime.js) | The large orchestrator: selection, action bar, cue/interaction/hotspot controls, backdrop texture lifecycle, chat tabs. |
| [state.js](frontend/lib/stage-runtime/state.js) | `normalizeSnapshot` / `createProjectedState` — where a snapshot element becomes a stage object, and where composition kinds map to node kinds. |
| [geometry.js](frontend/lib/stage-runtime/geometry.js) | Coordinate resolution, including the normalized 0-1 composition-coordinate branch. **Gated on node kind** — a new kind must be added here or its stored position is silently ignored. |
| [scene-nodes.js](frontend/lib/stage-runtime/scene-nodes.js) | PIXI node factories: token, card, fire, interaction hotspot. `markHiddenForDirector` applies Kernel 90's backstage treatment (dimming + dashed outline) to objects hidden from ordinary viewers. |
| [session-sync.js](frontend/lib/stage-runtime/session-sync.js), [socket.js](frontend/lib/stage-runtime/socket.js), [socket-controller.js](frontend/lib/stage-runtime/socket-controller.js) | Snapshot application, WebSocket decode and dispatch. |
| [program-panel.js](frontend/lib/stage-runtime/program-panel.js), [participant-interactions.js](frontend/lib/stage-runtime/participant-interactions.js) | The venue-agnostic Program Panel and the controller driving Equip Mode, the freeform door prompt, and Ra's dialogue. |
| [kernel89-director-tools.js](frontend/lib/stage-runtime/kernel89-director-tools.js) | Kernel 89's grouped Director surface: one `Director Tools ▾` selector over seven tool families, plus a top-level `Send Aftercare`. Owns the toolbar; [kernel85-cohort-tools.js](frontend/lib/stage-runtime/kernel85-cohort-tools.js) stands down when it sees `window.VictoryDirectorToolbarOwner`. |
| [kernel90-visibility-tools.js](frontend/lib/stage-runtime/kernel90-visibility-tools.js) | Kernel 90's Director visibility surface: the four canonical operations and the scope panel. Writes only through `/api/shows/{id}/stage-object-states`, and applies **no optimistic local state** — a hidden object is omitted from payloads rather than flagged, so the truthful update is the snapshot that follows the server's stage invalidation. |
| [kernel89-announcements.js](frontend/lib/stage-runtime/kernel89-announcements.js) | Theatrical announcement rendering — DOM, not PIXI, deliberately (a screen-space caption wants typography and an accessible reading order; dice need the camera transform). Knows how to draw a Style; does not know which ones exist. |
| [action-router.js](frontend/lib/stage-runtime/action-router.js), [editors.js](frontend/lib/stage-runtime/editors.js), [map-grid.js](frontend/lib/stage-runtime/map-grid.js), [token-ui.js](frontend/lib/stage-runtime/token-ui.js), [stage-controls.js](frontend/lib/stage-runtime/stage-controls.js), [context.js](frontend/lib/stage-runtime/context.js), [logic.js](frontend/lib/stage-runtime/logic.js), [lifecycle.js](frontend/lib/stage-runtime/lifecycle.js), [dice.js](frontend/lib/stage-runtime/dice.js) | Context menus, editors, grid, token picker, controls, lifecycle, dice tray. |

### Venues — [venues/](frontend/venues)

[catharsis/](frontend/venues/catharsis) is the live Socio stage (thin shell + `venue.js` + onboarding). [show-runs/](frontend/venues/show-runs) holds Stage Management, including [show.html](frontend/venues/show-runs/show.html)'s Scene Setup composer and [scene-preview.html](frontend/venues/show-runs/scene-preview.html) (Preview as Player). [writers-room/](frontend/venues/writers-room) (eWrite dashboard + [edit.html](frontend/venues/writers-room/edit.html) editor) and [library/](frontend/venues/library) (browse + [read.html](frontend/venues/library/read.html) reader) are Kernel 78's authoring/reading pair, backed by the pure-logic modules in [lib/ewrite/](frontend/lib/ewrite). Others: [audition-hall/](frontend/venues/audition-hall), [first-theater/](frontend/venues/first-theater), [greenroom/](frontend/venues/greenroom), [third-place/](frontend/venues/third-place), [trailers/](frontend/venues/trailers), [warehouse/](frontend/venues/warehouse), [workshop/](frontend/venues/workshop), [the-cave/](frontend/venues/the-cave), [directors-chair/](frontend/venues/directors-chair), [producers-office/](frontend/venues/producers-office), [victory-theater/](frontend/venues/victory-theater), [grants-cabin/](frontend/venues/grants-cabin), [middle-school-stage/](frontend/venues/middle-school-stage), [construction/](frontend/venues/construction), [construction-site/](frontend/venues/construction-site), [stage-template/](frontend/venues/stage-template), [shared/](frontend/venues/shared).

### Other

[assets/](frontend/assets) — static images including venue art, `courtyard.png`, `Kessa.png`, `ra.png`, `tutorial-handoff.png`, and [rulesets/](frontend/assets/rulesets). [lib/](frontend/lib) also holds PIXI, the camera, grid, command palette, mic/audio, and account badge. Top-level pages: [index.html](frontend/index.html) (map), [login/](frontend/login), [signup/](frontend/signup), [account/](frontend/account), [mailbox/](frontend/mailbox), [legal/](frontend/legal).

---

## Tests — [tests/](tests)

Plain `node --test`; no jest/vitest anywhere.

- [stage-runtime/](tests/stage-runtime) — engine unit tests, one per module. **Note:** [dice.test.js](tests/stage-runtime/dice.test.js) carries nine known failures tracked as a named exception in the alpha gate and roadmap; changing that count in *either* direction fails the gate, so a fix must update the script and roadmap together.
- [contract/](tests/contract) — scene-node contract tests.
- [ewrite/](tests/ewrite) — eWrite editor/outline pure-logic tests (Kernel 78).

Backend tests live beside their packages; DB-touching ones are conventionally `*_dbtest_test.go` and require `TEST_DATABASE_URL`.

---

## Scripts — [scripts/](scripts)

| Path | What it is |
|---|---|
| [smoke/kernel89-run.sh](scripts/smoke/kernel89-run.sh) | Kernel 89's acceptance proof: builds the backend, boots it against `victory_test`, runs [k89fixture](backend/cmd/k89fixture) and drives [kernel89-director-prepared-play.js](scripts/smoke/kernel89-director-prepared-play.js) over real HTTP and two real WebSockets. **Runs on the real `catharsis` venue row** (the `/ws/*` routes exist only for named venues, so a throwaway venue 404s on upgrade); it closes any leftover session first and its own afterwards. Do not run it concurrently with `go test ./internal/shows/...`. |
| [smoke/kernel89-director-tools-browser.js](scripts/smoke/kernel89-director-tools-browser.js) | Kernel 89's real-mouse UI proof: grouped tools, nested context menus, announcement rendering. |
| [smoke/kernel89-first-theater-announcement.js](scripts/smoke/kernel89-first-theater-announcement.js) | The same announcement proof aimed at First Theater. **Currently reports a PRECONDITION blocker, by design**: `access.ResolveVisibleVenues` has no `first-theater` branch, so no ordinary user can reach that venue. Kept working so the claim becomes provable in one command if that changes. |
| [smoke/kernel90-run.sh](scripts/smoke/kernel90-run.sh) | Kernel 90's acceptance proof: builds the backend, boots it against `victory_test`, runs [k90fixture](backend/cmd/k90fixture) and drives [kernel90-stage-object-visibility.js](scripts/smoke/kernel90-stage-object-visibility.js) — **five real viewers** (Director, two Cohorts, one Ungrouped Player, one Audience) against one live Show, each reading its own `/api/world/catharsis` projection. `--ui` additionally runs the browser proof. Same `catharsis` singleton-session caveat as Kernel 89's runner. |
| [smoke/kernel90-visibility-browser.js](scripts/smoke/kernel90-visibility-browser.js) | Kernel 90's real-browser proof, through the Kernel 87 dev proxy so it is same-origin. Player contexts are opened and closed **one at a time** — a memory constraint on this host (~3.8 GB total), not a weakening: the concurrent-multi-viewer claim is carried by the acceptance proof above. |
| [test/alpha-gate.sh](scripts/test/alpha-gate.sh) | The release gate: 7 orchestrated steps, explicit PASS/FAIL each. Do not create a competing gate. |
| [test/setup-test-database.sh](scripts/test/setup-test-database.sh), [reset-test-database.sh](scripts/test/reset-test-database.sh), [require-isolated-database.sh](scripts/test/require-isolated-database.sh), [lib-migrate-and-bootstrap.sh](scripts/test/lib-migrate-and-bootstrap.sh) | Test-database lifecycle, all behind the isolation guard. |
| [smoke/fresh-install.sh](scripts/smoke/fresh-install.sh) | Boots a real compiled binary against an empty database and drives real HTTP. Catches route collisions and bootstrap-ordering gaps that `go build`/`go vet` never see. |
| [smoke/kernel74-tutorial-browser.js](scripts/smoke/kernel74-tutorial-browser.js) | Playwright golden path for the tutorial. Preflights for a foreign Session and aborts rather than reclaiming one. Also: [kernel65](scripts/smoke/kernel65-third-place-browser.js), [kernel63](scripts/smoke/kernel63-back-to-map-browser.js), [kernel62](scripts/smoke/kernel62-browser.js), [kernel59a](scripts/smoke/kernel59a-phase3-browser.js). |

---

## Construction — [Construction/](Construction)

| Path | What it is |
|---|---|
| [current-state.md](Construction/current-state.md) | **Current-state canon.** Wins over any older current-tense statement. |
| [roadmap.md](Construction/roadmap.md) | Forward-looking roadmap and standing open items. |
| [Dictionary.txt](Construction/Dictionary.txt) | Vocabulary. |
| [Kernels/](Construction/Kernels) | Kernel specifications, including the drafted [kernel-75](Construction/Kernels/kernel-75-tutorial-completion-aftercare-continuation-v0.1.md). |
| [OperatorLogs/](Construction/OperatorLogs) | [operator-log.md](Construction/OperatorLogs/operator-log.md) (chronological), [operator-notes.md](Construction/OperatorLogs/operator-notes.md) (durable traps), and per-kernel reportbacks — most recent [kernel-74-reportback.md](Construction/OperatorLogs/kernel-74-reportback.md). |
| [Operations/](Construction/Operations) | Operator runbooks, including [director-prepared-play.md](Construction/Operations/director-prepared-play.md) — Kernel 89's Director guide, with an explicit section on what is deliberately *not* automated. |
| [kernel-maker-field-guide.md](Construction/kernel-maker-field-guide.md), [reportbacktemplate.txt](Construction/reportbacktemplate.txt), [templates/](Construction/templates) | How to write a kernel and its reportback. |
| [workflow/](Construction/workflow), [deployment/](Construction/deployment), [roadmaps/](Construction/roadmaps) | Dev workflow, install contract, fresh-install guide, long-form track roadmaps. |
| [character-workbook.md](Construction/character-workbook.md), [Construction — Contracts.txt](<Construction/Construction — Contracts.txt>) | Workbook design and standing contracts. |

---

## Deployment and runtime

[docker-compose.yml](docker-compose.yml) defines `victory-postgres` and `victory-backend` on the external `edge_net`. TLS and static file serving are handled by the **shared `bread-caddy` container**, whose real config lives in `/opt/bread-exchange/` — not by this repo's [Caddyfile](Caddyfile). Migrations auto-apply at backend boot with a pre-apply dump into [backups/](backups). Runtime uploads live in [storage/](storage).

Deploy is `docker compose build backend && docker compose up -d backend`; confirm the migration lines in `docker logs victory-backend`.
