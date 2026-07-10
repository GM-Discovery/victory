# Operator Notes — VICTORY Foundation

## Current Canon Note
Current system truth now lives in:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [roadmap.md](/opt/victory/Construction/roadmap.md)

This file remains useful for longer-form modeling notes and historical reasoning, but some older status sections below are now historical/superseded snapshots rather than the live canonical state.

## Kernel 59A Phase 3 Projection Sync

- Character tray projection invalidation is server-authored only. Clients must not send `character/projection_updated`; the websocket handler returns `server_authored_event_only`.
- The invalidation payload is deliberately small: `character_id`, `projection_version`, `changed_dimensions`, optional `source_event_id`, and timestamp. It is a refetch signal, not character truth.
- `projection_version` is currently derived from durable `updated_at` values on `character_cards`, `character_face_overrides`, and `character_skills`. There is no revision-counter migration yet.
- Venue runtimes compare projection versions and refetch `/api/characters/venue-sheet` only for the currently displayed character and only when the incoming version is newer.
- First Theater and Catharsis character trays now have Face and Mechanics tabs. Face is default; Mechanics preserves skill click-to-roll.
- Backend Docker images contain a baked Go binary. Rebuild with `docker compose up -d --build backend`; a plain restart is not enough after projection code changes.

## Purpose of this file
This file exists to keep future builders from re-arguing settled concepts, repeating solved mistakes, or building against the wrong model.

Read this before making structural changes.

---

## Current Status
Historical note:
The section below began as an early foundation snapshot and should no longer be treated as the single source of truth for the live system. Use `Construction/current-state.md` for current route, venue, and kernel state.

Current confirmed capabilities:
- Postgres is running in Docker on the server
- schema and seed world are loaded
- venue `the-cave` exists
- reusable element `first-fire` exists in the library
- `first-fire` is placed into `the-cave`
- one live session exists for `the-cave`
- persistent user creation/join works by `handle`
- HTTP snapshot endpoint works
- WebSocket observer connection works
- `perform/speak` works over WebSocket
- `react/emote` works over WebSocket
- `react/emote` is persisted into `actions`
- `react/emote` is broadcast to connected observers
- later-joining observers see prior recorded actions in snapshot history
- The Cave WebSocket now resolves server-side identity and emits presence joins/leaves
- Speech and reaction actions carry server-enriched actor display names, handles, and roles
- The Cave client renders a live "Who is here?" roster
- The Cave client renders speech and reactions with attribution badges
- Multiple tabs for the same user are deduped in the visible presence roster
- Presence messages now use explicit `presence/snapshot`, `presence/join`, and `presence/leave` event types
- Discord OAuth is now wired as the primary login path alongside local handle/password login
- The Victory session cookie model remains authoritative after Discord login
- Discord login does not grant producer/director authority by itself
- Kernel 33 adds a separate operator bootstrap CLI for granting producer authority after identity already exists
- Kernel 33 captures Kernel 32 plus the live-site follow-on work into the canon docs
- Kernel 7 evidence is covered by `go test ./internal/network -run TestKernel7PresenceAndAttributionEvidence -v`
- Kernel 8 adds a shared identity surface and `persona: null` in action/presence payloads
- The Cave does not allow anonymous presence
- Presence labels must never render blank; fall back through display name, handle, shortened user id, then `Unknown Participant`
- `elements.context_class` is now persisted in Postgres so props, scenery, cards, and future classes do not depend only on UI inference
- `prop` means mobile stage object; `scenery` means fixed set piece / anchor; `first-fire` is treated as fixed scenery for now
- Kernel 21 venue chat is a separate `chat/message` lane and must not be conflated with `perform/speak`
- The Cave chat panel is venue-scoped, session-backed, and collapses after inactivity
- Kernel 23 adds story-first character cards, draft grants, session personas, `persona/equip`, `persona/unequip`, and `presence/update`
- Kernel 24 moves character drafting/editing into The Greenroom and leaves The Cave with only live character selection plus Put On / Take Off
- Character card sheet links are metadata on `character_cards`; they are not playable sheets, macros, dice, stats, or rules execution yet
- The Cave is now explicitly the full-feature proving-ground venue. Tools may appear there first, then later move into overlays, drawers, and context menus before template extraction.
- Middle School Stage is the first clean stage-shell experiment. It should stay producer-only, drawer-first, and center-stage-primary.
- Kernel 32 now extracts a hidden Stage Template venue from that shell and reuses the portable overlay over First Theater's Pixi stage.
- Pixi remains outside The Cave and is not automatically part of the stage shell.
- Showing Review means review of actions/chat/reactions/logs. Video recording is future-only and not a near-term priority.

Not yet complete:
- full two-user kernel proof: actor speaks, audience perceives, audience reacts, actor perceives reaction

Status notes for newer kernels:
- Kernel 6 action authority is centralized in `backend/internal/actions/authority.go`
- Kernel 7 presence and attribution are centralized in `backend/internal/network/presence.go`
- Presence is in-memory only; it is not canonical history
- The Cave remains the only venue wired for live action/presence protocol right now
- The Greenroom and Trailers are signed-in performer profile venues
- `persona` is reserved for future production characters and is `null` for Kernels 8 and 9
- The profile surface now includes expressive fields like favorite fun, favorite color, favorite artist, favorite food, favorite song, favorite place, favorite movie or show, hidden talent, and ideal day
- `OPERATOR_HANDLE` and `OPERATOR_USER_ID` provide an infrastructure-only permission seam and do not create a production membership
- Discord OAuth state must remain single-use and short-lived
- Discord access tokens and client secrets must not be logged

---

## Authority Model (settled)
Do not blur these layers.

### Operator
This is the infrastructure authority.
The operator:
- controls the server
- controls Docker and deployment
- controls updates
- controls source code
- provisions hosted instances

The operator is **outside** the ordinary in-app permission model.

### Producer
Producer is the highest normal in-app authority within a Location.
Producer does **not** get:
- shell access
- Docker access
- source code
- `/opt` access
- host-level authority

Producer may:
- manage production space inside the app
- assign directors
- share producer authority with other producers if desired

Producer is a **role**, not host ownership.

---

## Core World Model (current accepted interpretation)
These terms matter.

### Location
Hosted software instance / world-space.
Contains:
- lots
- libraries
- users
- memberships

### Lot
Collection of venues inside a location.

### Venue
Defines experience rules and view structure.
A venue is not an asset library.
Examples:
- `the-cave`
- future classroom / school venue
- future vault / hidden space
- future large vertical venue

### Library
Owns reusable elements.
Elements belong here, not to the venue.

### Element
Reusable object / asset.
Example:
- `first-fire`

Element subtype / context class is now stored in the database with `elements.context_class` so later kernels can distinguish:
- `prop` for mobile objects
- `scenery` for fixed set pieces
- `card` for index cards
- other future classes as needed

### Placement
Instruction that places an element into a venue/session context.
Important:
- the fire is an element
- the placement is what belongs to venue defaults

### Session
Live instance of a venue.
This is where actions happen.

### Action
Append-forward event in a session.
Do not destructively rewrite past recorded actions.
Change forward by appending a new action.

### Presence
Live connection state for a session.
Presence is:
- derived from authenticated WebSocket state
- in-memory
- ephemeral
- not canonical history
- safe to project to the current Cave session audience
- The Greenroom public profile projection is server-resolved from profile state, not client claims
- Grant's Cabin is invite-only and only appears on the map when a venue grant exists
- Catharsis is a signed-in audience venue with invitation/ticketing still deferred
- The Cave stage now uses a right-click context menu for move/reveal/hide/remove/info actions on placed elements

---

## Critical Modeling Decisions Already Settled

### 1. Observation comes before reaction and speech
Order of implementation is intentional:
1. observe world
2. observe action stream
3. react
4. speak

This was chosen on purpose and should not be casually reversed.

### 2. Library ownership, not venue ownership
Elements are reusable across venues.
Do not attach reusable assets directly to one venue unless that is explicitly session-local or venue-local by design.

### 3. Producer is not infrastructure owner
Producer authority is in-app only.
Operator authority is infrastructure-level.

### 4. Venue should not become a junk drawer
Do not force unrelated systems into venue runtime logic just because they live on the same lot.

A likely long-term separation:
- venue runtime layer
- institutional/admin layer (school, CRM, records, staffing)
- shared identity/location/library layer
- profile/public identity layer

### 5. Append-only action stream is the canon rule
Actions are ordered by server-assigned `moment_id`.
History is extended forward, not overwritten backward.

---

## Current File / Repo Layout

Repo root:
- `/opt/victory`

Important directories:
- `backend/` — Go backend code
- `database/migrations/` — SQL migrations and seed files
- `Construction/` — design/spec material
- `docs/` — operator-facing memory and guidance

Suggested future additions:
- `scripts/`
- `install/`

---

## Current Runtime Setup

### Deployment truth
Docker remains the install/deployment standard.

### Current development mode
For faster development:
- Postgres runs in Docker
- backend can run directly on host with Go
- this avoids slow full container rebuilds during active coding

### Important rule
Do not confuse:
- **development convenience**
with
- **install contract**

Fresh-server install target should still be Dockerized and reproducible.

---

## Current Services

### Postgres
Container:
- `victory-postgres`

Current database:
- `victory`

### Backend
Docker container exists and is valid for deployment:
- `victory-backend`

During active development, backend may instead be run directly on host with:
- `go run ./cmd/victory`

---

## Current Ports

### In development
- backend: `8081`
- postgres: bound to localhost for host-Go development

### Long-term
Backend should eventually sit behind reverse proxy.
Postgres should remain non-public.

---

## Current Database Objects

Tables currently in use:
- `users`
- `locations`
- `location_memberships`
- `lots`
- `venues`
- `sessions`
- `session_participants`
- `actions`

Current operator-reference files:
- `Construction/Kernels/kernel-6-action-authority.md`
- `Construction/Kernels/kernel-7-presence-attribution.md`
- `libraries`
- `elements`
- `venue_layout_elements`
- `sessions`
- `session_participants`
- `actions`

Enums:
- `location_role`
- `session_status`
- `element_state`

---

## Current Seeded World

Location:
- `amurray.family`

Lot:
- `main-lot`

Venue:
- `the-cave`

Library:
- `house-library`

Element:
- `first-fire`

Placement:
- `first-fire` placed on `stage` in `the-cave`
- anchor is `center-front`

Active session:
- one live session exists for `the-cave`

---

## Identity Model (current)
User identity currently includes:
- stable server `id`
- mutable `handle`
- mutable `display_name`

Current join behavior:
- `handle` is anchor identity for v1
- `display_name` can change
- server creates or updates user by handle
- user is joined to active `the-cave` session
- default role assigned at join is currently `audience`

This is expected to expand later for:
- actor personas
- character names
- production/session-specific display identities

---

## Current API / Transport

### HTTP
- `GET /health`
- `GET /api/world/the-cave`
- `POST /api/session/the-cave/join`

### WebSocket
- `GET /ws/the-cave`

Current websocket behavior:
- sends initial snapshot on connect
- accepts `ping`
- returns `pong`
- accepts `react/emote`
- persists reaction
- broadcasts persisted reaction to connected observers

---

## Proven Behaviors

### Snapshot behavior
Client receives:
- location
- lot
- venue
- active session
- placed elements
- prior recorded actions

### Live reaction behavior
Client can send:
- `react/emote`

Server does:
- validate participant belongs to session
- validate reaction kind
- assign next `moment_id`
- persist action into `actions`
- broadcast resulting action to all connected clients

### Late joiner behavior
Later-joining observers receive historical recorded actions in snapshot, then continue receiving live actions.

This behavior is important. Do not lose it accidentally.

---

## Known Workflow Issues

### Slow Docker rebuilds
Full Docker backend rebuild has taken roughly ~55 seconds during development.
This is developer friction, not evidence of runtime lag for users.

### Current chosen workaround
Use host-installed Go for active backend development while keeping:
- Docker Postgres for data
- Docker packaging for deployment/install truth

This hybrid mode is intentional for now.

---

## Security / Exposure Notes

Current testing uses direct access on backend port `8081`.
This is acceptable for development/testing.

Long-term expectation:
- reverse proxy in front
- backend not casually public
- Postgres internal only
- secrets moved out of inline Compose values into `.env`

---

## Immediate Next Build Target
In priority order:

1. `perform/speak`
2. actor-capable session path
3. full two-user kernel proof
4. minimal browser-facing UI
5. later: note cards and index cards
6. later: reveal/hide and stronger role-aware visibility

---

## Warnings to Future Builders

### Do not:
- move reusable elements into venue ownership by default
- turn producer into host/server authority
- collapse school/admin/CRM logic directly into the theatrical runtime layer
- replace append-forward actions with destructive mutation of history
- optimize for abstract scale before proving lived interaction

### Do:
- preserve clean world vocabulary
- preserve separation of operator and producer authority
- preserve library-vs-placement distinction
- preserve observer-first action flow
- preserve event-driven runtime rather than whole-state replacement

---

## Kernel 10 Info Booth + Mailbox

### Delivery Model
- Info Booth is a public map modal.
- Mailbox is durable authenticated inbox delivery.
- Chat stays immediate; mailbox stays durable.

### Security Rules
- Users can only read their own inbox.
- Message creation is operator/dev only.
- Client identity claims are never trusted for messages.

### Data Model
- Messages are stored in `messages`.
- Public API shape includes:
  - `id`
  - `to_user_id`
  - `from_user_id`
  - `subject`
  - `body`
  - `created_at`
  - `read`
- Message bodies are capped around 250 characters.

### Routes
- `GET /api/messages`
- `GET /api/messages/{id}`
- `POST /api/messages`
- mailbox page: `/mailbox/`

### Test Insert
Use SQL for a quick operator test message:

```sql
INSERT INTO messages (to_user_id, from_user_id, subject, body)
VALUES (
  '<recipient-user-id>',
  NULL,
  'System message',
  'Mailbox foundation is live.'
);
```

### Restart
- Backend restart is required after code changes.

---

## Kernel 11 Note Cards

### Delivery Model
- Note cards are mailbox messages with `message_type = note_card`.
- They are durable and read later.
- They are not live chat and do not pop up in real time.

### Sender Rules
- Sender must be authenticated.
- Sender is resolved from the Cave session server-side.
- Client-provided `from_user_id` is never trusted.

### Recipient Rules
- Prefer a visible/current Cave participant.
- Support `to_participant_id` when available.
- Only allow recipients visible in the Cave roster with these roles:
  - Director
  - Cast
  - Crew
- Fall back to the current director mailbox.

### Data Model
- Note cards live in `messages`.
- Context fields:
  - `venue_slug`
  - `session_id`
- Body limit is 250 characters.

### Routes
- `POST /api/note-cards`
- `GET /api/messages`
- `GET /api/messages/{id}`

### Frontend
- Cave now has a send-note-card form.
- Trailers links directly to Mailbox.
- Publish saves the current draft first, then copies it into the public projection.

### Restart
- Backend restart is required after the schema/bootstrap and route changes.

## Kernel 13 Workshop Placement

### Placement Model
- Workshop mode is the source surface for card creation.
- `act/place_element` is the server-authoritative placement action.
- Venue trays/backstage are distinct from stage/worldspace.
- Placement uses `x = 0`, `y = 0` until drag/drop exists.

### Venue Enablement
- `venues.config.index_cards_enabled` controls whether a venue appears in the send-to-venue list.
- `GET /api/workshop/venues` returns the enabled targets the current user can access.
- The Cave is enabled for index card placement.

### Routes
- Workshop entry: `/venues/workshop/`
- Workshop mode on the Cave shell: `/venues/the-cave/?mode=workshop`

## Kernel 15: Producer's Office + Director's Chair

- New map venues:
  - `producers-office`
  - `directors-chair`
- Map visibility now includes notification counts for pending request queues.
- `GET /api/requests/incoming` returns the producer/director review queue.
- `POST /api/requests/respond` approves or denies a pending request and writes the grant or membership server-side.
- `GET /api/productions` returns the current scoping productions for invite creation.
- Producers can create director/cast/crew/audience invites from the Producer's Office.
- Directors can create cast/crew/audience invites from The Director's Chair.
- Audience invites are venue-scoped. Director/cast/crew invites are production-scoped.
- Request cards now foreground display name, not login handle.

### Operator Flow
- Open Workshop from the map.
- Create or edit an index card.
- Select a target venue from the enabled list.
- Send to venue tray.
- Place on stage.
- Reveal to audience through the existing visibility system.

## Kernel 22 Showing Model

- Presence is who is connected now; showing is the durable theatrical record.
- Reconnects should reuse the same showing instead of creating a new one.
- Actions, chat, reactions, reveal/hide, overlay, and card placement now record a `showing_id`.
- `rehearsal` means audience view off.
- `live` means audience view on.
- `closed` means no new chat or stage actions.
- The Cave venue chat lives in the bottom panel.
- `perform/speak` remains the separate stage speech lane above the fire.
- There is no chat-above-fire configuration in Kernel 22.

## Kernel 23 Story-First Character Cards

- Character cards are profile-linked personas, not alternate accounts.
- Producers and directors can draft character cards implicitly.
- Cast and crew need a `character_card:draft` permission grant before drafting.
- `permission_grants` is the generic capability table introduced by this kernel.
- `current_session_personas` stores the active character card for a user in a session.
- The Cave supports `persona/equip` and `persona/unequip` over WebSocket.
- Equipping or unequipping broadcasts `presence/update`.
- New actions record the active persona in `payload.actor_persona` and project it through `actor.persona`.
- Existing actions without persona data still project `persona: null`.
- Dice, grid, tokens, HP, initiative, and fog remain outside this kernel.

## Kernel 61 / 61A — Player Workbook, Trailer Face, and Legacy Profile Migration

**Status: PARTIAL.** Full ledger in `Construction/OperatorLogs/kernel-61-reportback.md` and `kernel-61A-reportback.md`. This section is the durable operator reference; the reportbacks are the point-in-time evidence record.

### Account vs Player Workbook vs Trailer Face vs Character Workbook

These are four distinct layers — do not conflate them:

- **Account** (`users` table + `auth.*`): stable UUID, sign-in handle (read-only outside admin tooling), email (private, securely editable — see below). This is authentication and accountability, not presentation.
- **Player Workbook** (`player_profile_*` tables, `backend/internal/playerprofile/`): a *real user's* private, catalogue-driven pages, typed profile events, and current facts. One per account, created on first touch (`EnsureWorkbook`). Reachable at `/venues/trailers/workbook.html`.
- **Trailer Face**: the owner-curated *compiled social projection* of eligible Player Workbook facts — not a separate data store, a projection (`BuildTrailerFace`) over Workbook facts + `player_profile_face_overrides`. Owner view: `/venues/trailers/face.html`. Social (another authenticated user's) view: `/venues/trailers/view.html?id=<workbook_id>`.
- **Character Workbook** (`character_workbook_*`, `character_face_overrides`, Kernel 53/59A): a *fictional character's* sheet — completely separate schema, completely separate Face-override mechanism, lives in Greenroom. A user can own many characters but has exactly one Trailer. Do not route player-identity work through Character Workbook tables or vice versa.

### Stage-name ledger

- `player_stage_name_history`, keyed by the account UUID (not the workbook ID) — survives even if the Workbook itself were ever rebuilt.
- Append-only: a change closes the current open interval (`ended_at = NOW()`) and opens a new one in one transaction (`ChangeStageName` in `stagename.go`). A partial unique index (`idx_player_stage_name_history_current`) enforces exactly one open row per user at the DB level, not just in application code.
- Idempotent: resubmitting the same normalized (case/whitespace-insensitive) name is a no-op, no duplicate row.
- **Cannot be deleted through any ordinary endpoint** — `DELETE /api/player-profile/events/{id}` only ever targets `player_profile_events`, a different table entirely, so a stage-name ledger ID passed there 404s. There is no delete path for this table anywhere in the product, including for the operator.
- UI: inline "Edit" control next to the stage name on `face.html`; read-only ledger view on `workbook.html`'s History tab.

### Ordinary History / event deletion

- `player_profile_events` (page commits, legacy imports) are owner-deletable, unlike the stage-name ledger.
- Deleting an event triggers `RecomputePlayerFacts`, which fully rebuilds `player_profile_facts` from whatever events remain (fold in chronological order, last write per field wins) — not a patch, a full recompute, so there's no drift possible between events and facts.
- The Workbook's History tab computes an accurate **client-side** deletion-impact preview before the owner confirms, by replaying the same fold logic in JS. That JS copy is not shared with the Go implementation — if `DeriveEffectiveFacts` in `facts.go` ever changes, the client-side preview in `workbook.html` needs a matching update or it will silently drift.

### Targeted profile invalidation (websocket)

- Endpoint: `/ws/player-profile` (`backend/internal/network/profile_ws.go`) — deliberately **not** built on the existing `ServeVenueWS`, because that requires an active venue *session* (Cave/Catharsis-style), which Trailers has no concept of. This is a standalone, auth-only websocket.
- Client sends `{"type":"watch_profile","profile_id":"<workbook_id>"}` after connecting. The server only ever recognizes that one inbound message type — everything else is silently dropped, and nothing is ever relayed from one client to another, so a client cannot forge a `player_profile/projection_updated` event.
- Server pushes `{"type":"player_profile/projection_updated","profile_id","projection_version","changed_dimensions","ts"}` to every client currently watching that `profile_id` — owner's other tabs and another user's Trailer-viewer tab both use the exact same mechanism; there is no separate "owner channel."
- The payload is deliberately just an opaque ID + version, never facts — clients always refetch through the real authenticated HTTP endpoint, the push is invalidation-only.
- Wired into every mutation via the `ProjectionChangeNotifier` callback pattern (matches the existing `characters.ProjectionChangeNotifier` shape) — `network.BroadcastPlayerProfileProjectionInvalidation`, passed into `main.go`'s player-profile route registrations.

### Secure email behavior

- `PATCH /api/account/email` (`backend/internal/identity/account_email.go`). Owner-only (resolved from session, never a client-supplied target ID), format-validated, case-insensitive-unique-checked, and **real password reauthentication** via the same `VerifyPassword` Argon2id check used at login.
- **Accounts with no password credential (Discord-only signup) are explicitly refused** (`password_reauth_unavailable_for_this_account`) rather than given a weaker confirmation path — there is currently no safe step-up reauth for provider-only accounts. This is a known, deliberate gap, not an oversight: fixing it means building a real Discord-OAuth step-up flow, which was out of scope for this pass.
- Email is never written into any Player Workbook table — it cannot appear in an event, History entry, fact, or Face under any circumstance, by construction (it isn't a catalogue field and `IsReservedFieldKey` blocks it from ever becoming a Face override target even if someone tried).
- UI: `/account/` — "Change Email" button reveals an inline current-password + new-email form.

### Legacy `/api/profiles/*` status

- **Closed (Path B: deprecate/disable), not adapted.** All 6 routes (`GET /me`, `GET /public`, `POST /me/save`, `POST /me/publish`, `POST /admin/save`, `POST /admin/publish`) now return `410 Gone` with `{"error":"deprecated_use_player_profile_workbook"}` — none of them read or write `performer_profiles` anymore.
- `performer_profiles` itself is **not dropped** — left in place, untouched, as historical/migration-audit data (per Kernel 61 §11.5). It was migrated into the new tables once, idempotently, during Kernel 61.
- The old Trailers editor (`frontend/venues/trailers/index.html`) is now a redirect stub to `face.html`, not a working form — kept as a URL (nothing that links to `/venues/trailers/` breaks), but there is no legacy UI left that reads or writes through the old surface.
- `frontend/account/index.html`'s profile summary panel was migrated to read `GET /api/player-profile/me` instead of the deprecated route.

### Straturli / cabin preservation

- Nothing in Kernel 61/61A touches `users`, `location_memberships`, `access_grants`, `character_cards`, venue tables, or operator resolution (`OPERATOR_HANDLE`/`OPERATOR_USER_ID` env-based, unrelated to any Player Workbook table).
- Live-reverified each session: Straturli's UUID, handle, operator status, and Grant's Cabin access all unchanged. Straturli's own stage name was never touched by any test — the account owner changed it themselves through the real UI between sessions (confirmed live as "Grant A. Murray", not a leftover migration artifact like the original "The Starmaker").
- A pre-existing, unrelated flaky test (`internal/network`'s `TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow`) leaks orphaned `bridge_operator_*` fixture users/rows into the live DB when run — confirmed via `git stash` that this happens with or without any Kernel 61A code present. Not fixed (out of scope), but do not mistake those rows for Kernel 61A test residue if seen in the `users` table.

## Kernel 62 — Private Player Relationships (My People)

### Private directional relationship model

- A relationship record is **directional and observer-private**: `player_relationships(observer_user_id, subject_user_id)`, unique per pair, CHECK against self. "B has notes about A" implies nothing about A→B, and A's own My People list is unaffected.
- The record follows the subject's **stable account UUID**, so stage-name changes never detach it — but externally the subject is only ever addressed by their opaque Player Workbook ID (`resolveSubjectUserID` mirrors K61's `resolveUserIDForWorkbookID`). Raw account UUIDs are struct-tagged out of every JSON response (`ObserverUserID`/`SubjectUserID` are `json:"-"`), and there is a unit test that marshals every view type and greps for the UUIDs.
- Package `backend/internal/playerrelationships/` deliberately mirrors `playerprofile` (catalogue → validation → events → full-replace fact recompute → HTTP). Same `{ok,data}` envelope, same `writeError` shape.

### Subject invisibility rule

- The subject can never see that the record exists. Every relationship route proves `observer_user_id == session user` inside `loadRelationshipOwned`; failure is `relationship_not_found` → **404, never 403**, so a non-owner cannot distinguish "not mine" from "doesn't exist."
- There are no enumeration surfaces: no "who has notes about me," no counts, no global note search. The observer identity has no request field at all — it cannot be spoofed, only derived from `victory_session`.
- No websocket/notification is emitted for any relationship mutation. The only live update on the person page is the **subject's own public Face header** via the existing `/ws/player-profile` watch — public data the observer could see anyway.

### Qualitative dropdown vocabulary

- Words, not numbers, per Kernel 62 §6: trust (`unknown/cautious/developing/trusted/deeply_trusted`), closeness (`…/distant/familiar/friendly/close/core_relationship`), reliability (`…/inconsistent/usually_reliable/reliable/highly_reliable`), communication ease (`…/difficult/uneven/workable/easy/very_easy`), state (`active/quiet/strained/rebuilding/archived`). Fixed sets in `vocab.go`; server rejects any other key; UI renders labels served by `/api/player-relationships/catalogue`.

### Archive behavior

- Archive/unarchive only — **no relationship delete exists** (journal-entry delete does, as soft delete). Archive sets `archived_at` AND `relationship_state='archived'`; unarchive clears both. The state dropdown cannot set `archived` directly (`ValidateSettableStateKey` excludes it), so `archived_at` and the state can never disagree. List filter: Active (default) / Archived / All.

### No notifications for follow-ups

- Follow-ups are stored rows with `open/done/dismissed` (+ UI reopen). There is deliberately no scheduler, reminder, email, calendar, or websocket path for them anywhere in the package — if someone later "helpfully" adds due-date alerts, that is a spec violation, not a missing feature.

### Shared context limits

- `ProjectSharedContext` surfaces **only** what the server can verify: productions where both users hold active `memberships` rows, each side's roles, and the overlap start (later of the two earliest memberships). Pure fold (`FoldSharedProductions`), unit-tested, safely empty. Nothing inferred — no attendance, chat frequency, closeness, or "you may know." The UI labels the panel "Victory can currently verify" and shows an honest empty state; the observer's own Shared Work and Play page is separate manual private notes.

### Privacy caveat about self-hosted operators

- The privacy copy says exactly "Only you can see these notes. They are not shared with this person." — application-level enforcement only. A self-hosted operator with database access can technically read `player_relationship_*` tables; nothing claims cryptographic secrecy, and no doc should.

## Kernel 63 — Discord Test Fixture-Leak Cleanup and Back-to-Map Navigation

### Discord automatic-reconcile channel-mapping bug (real production fix)

- `saveDiscordChannelMapping` (`backend/internal/identity/discord_channel_mapping.go`) now wraps `created_by_user_id` in `NULLIF($11, '')::uuid` before casting, matching the guard already present on `parent_discord_channel_id`. Before this fix, `ReconcileDiscordBootstrap` — the goroutine that runs automatically on every real server boot — always called the mapping-save path with an empty `createdBy` string, and the bare `::uuid` cast of `""` crashed on every single spec. The crash was swallowed into `summary.Failed` rather than surfaced, so **automatic startup channel-mapping reconcile has never actually persisted a mapping** since this code shipped; only the manual, authenticated `/api/discord/channel-mapping/repair` HTTP path (which supplies a real user ID) ever worked. If you're debugging "why didn't the bot channels get mapped automatically," this is why — check whether a human has ever hit the manual repair endpoint for the location in question.
- Found this by fixing test fixtures far enough that `TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand` finally exercised the automatic-reconcile path for the first time in its history; it had always failed earlier (at channel-listing mock gaps) before reaching this bug.

### First Theater / Catharsis back-pill is hover-hidden (correction to prior assumption)

- Both venues' `header-pill--back` sits inside `.header-right`, and both have `.top-bar[data-open="false"] .header-right { opacity:0; visibility:hidden; display:none; }` — the pill is **not visible by default**, only when the header is hovered/pinned open. Don't assume these two venues already have adequate "back to map" affordance just because a pill exists in the markup; verify with `getComputedStyle`, not a DOM-presence check.

### `back-to-map.js` shared component

- `frontend/lib/back-to-map.js` mirrors `venue-account-badge.js`'s mount-detection pattern exactly (`.launchbar`/`.toolbar`/`.header-right`/`.top-actions` host detection, floating fallback). Add `<script src="/lib/back-to-map.js"></script>` for default mounting, or `data-back-to-map="floating"` on the tag to force the floating pill regardless of host — required for any page whose obvious mount point is hover-hidden by default (see above). The floating pill sits top-**left** (`z-index:95`) deliberately, so it never collides with the account badge's top-right float.
- `document.currentScript` must be read synchronously at the top of the IIFE, not inside the deferred `DOMContentLoaded` callback — it reads back `null` by the time that callback fires. This bit the first draft of the component; fixed before it shipped.

### Operator-only venues can't be browser-proven with a plain test account

- `access.ResolveVisibleVenues` only returns the full venue list for the literal env-configured operator (`OPERATOR_HANDLE`/`OPERATOR_USER_ID`) — a `location_memberships.role='producer'` grant is NOT the same thing and does not unlock every venue. Middle School Stage and Stage Template are currently operator-only (not in any of the `authenticated_surface`/`approved_performer_surface`/`performer_surface`/`owned_workbook_surface` branches), so a disposable producer-bootstrapped test account correctly sees the forbidden-screen there, not the real shell. Do not mutate `OPERATOR_HANDLE` to work around this in a test/proof script — it's a single, shared, environment-wide identity, not something to reassign temporarily. Verify such pages statically (served-HTML content check) instead.

### Live-DB test residue

- 12 `bridge_operator_testmirrorvictorychattodiscordpostsmessageandpersistsbridgerow_*` fixture users (and their FK-linked sessions/memberships) were cleaned up as part of this kernel — the leak source (missing `discord_session_threads` cleanup, see reportback) is now fixed, so this should not recur.
- ~25 other older stale test users (`tester1`, `kernel49test...`, etc.) remain, explicitly untouched — unrelated to the tests this kernel fixed. Named as a follow-up sweep candidate, not folded into this kernel's cleanup.
