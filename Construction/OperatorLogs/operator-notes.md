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

## Kernel 64 — DB Test Isolation and Live-DB Safety Gate

### The dedicated test database lives on the same Postgres server as the live one

- There is only one Postgres instance in this deployment (`victory-postgres`, port `127.0.0.1:5432` published to host). The dedicated test database (`victory_test` by convention) is just a different database name on that same server, not a second server. If you ever see `TEST_DATABASE_URL` rejected for its host, that's a bug in the safety-gate rules — the discriminator is supposed to be the database *name* (must contain `test`, must not be `victory`/`postgres`/production-looking), never the host. An earlier version of `scripts/test/require-isolated-database.sh` got this backwards (rejected the host outright) before this kernel fixed it — if you ever see that pattern reintroduced, it's a regression.

### Raw SQL migrations are not the whole seed story

- Several venues — `first-theater`, `catharsis`, `middle-school-stage`, `warehouse`, `workshop`, `library`, `trailers`, `grants-cabin`, `construction`, `audition-hall`, `soil-experts` — are created by Go-side `Ensure*Surface` bootstrap functions that `cmd/victory/main.go`'s `main()` runs on every real server boot (e.g. `internal/access.EnsureKernel16VenueSurface`), not by any file under `database/migrations/`. A database that only has the SQL migrations applied is missing all of these. `scripts/test/setup-test-database.sh` and `reset-test-database.sh` both build and briefly boot the real `victory` binary against the target database for exactly this reason, then kill it — the binary is never meant to keep running as part of test-DB setup, just to execute its idempotent startup bootstrap once. If a future kernel adds a new `Ensure*Surface` function and forgets to wire it into `main.go`'s startup sequence, this same "works live, fails against a fresh DB" pattern will resurface — check `main.go`'s `Ensure*` call list first when a DB-touching test fails only against `victory_test`.

### Two organic, never-migrated pieces of live-DB state that tripped up test fixtures

- The live database has a `productions` row for the `amurray-family` location (`"Main Production"`) that was created out-of-band at some point — no migration file or Go bootstrap function creates it. `showings.EnsureForSession` requires at least one production to exist for the session's location (`production_required` error otherwise). `internal/network/discord_chat_bridge_test.go`'s fixture now creates its own (`ON CONFLICT (location_id, slug) DO NOTHING`, so it's safe against both a fresh DB and the live DB's existing one) rather than assuming one is already there.
- Similarly, `gateway-thread-location` (a second, separate `locations` row) exists live but isn't created by the canonical migration list (which instead seeds `victory-theater` as the Kernel 42 neutral install location, per `025_kernel42_neutral_install_location.sql`) — a reminder that the live database's history and a from-empty migration replay are not byte-identical, and that's expected, not a bug to chase.

### The safety gate is duplicated in two languages, on purpose

- `backend/internal/dbtest.ValidateTestDatabaseURL` (Go) and `scripts/test/require-isolated-database.sh`'s `require_isolated_database` function (shell) implement the same rules independently — Go tests can't shell out per-test cheaply, and the shell scripts run before any Go test binary exists, so there's no single point both can share without adding a dependency the other doesn't need. If the rules ever change (e.g. the required marker stops being the substring `"test"`), update both. Each file's header comment points at the other.

### `go test ./...` behavior change

- Before this kernel, `go test ./...` connected straight to the live `victory` database with no env var involved at all (three test files, each with a function like `openDiscordTestPool` hardcoding the live connection string) and silently `t.Skip`'d if Postgres was unreachable — meaning a broken DB connection looked like a pass, not a failure. Both of those are gone: `TEST_DATABASE_URL` is required and unsafe/missing values now `t.Fatalf`. If you see `go test ./...` suddenly "failing" after pulling this kernel's changes, you almost certainly just need to run `scripts/test/setup-test-database.sh` once and export `TEST_DATABASE_URL` — see `dev-workflow.md`'s "Database Changes" section.

## Kernel 65 — Third Place Headshot Commons MVP

### Third Place venue and visibility rule

- `third-place` is seeded exactly like `trailers`/`greenroom` (idempotent `INSERT ... WHERE NOT EXISTS`, migration 038) and is deliberately given the *same* map-visibility rule as `trailers` in `access.ResolveVisibleVenues` — one UNION arm now reads `v.slug IN ('trailers', 'third-place')` instead of `v.slug = 'trailers'`. That rule requires an active `location_memberships` row with role in `producer/director/cast/crew` (see `IsPerformerRole`) — a plain `audience`-role account won't see the *map tile*, but the underlying page and every `/api/third-place/*` route only ever check "is this an authenticated session," no role check at all. This mirrors Trailers exactly: the venue tile is performer-gated, the API/page itself is not. Don't "fix" this apparent inconsistency without re-reading Kernel 61A/65's own precedent first — it's intentional.

### Headshot terminology and one-active-per-account rule

- The product term is **Headshot** (the presence marker) inside **Third Place** (the venue). An earlier concept called "Faceprint" was explicitly abandoned — don't resurrect that name in copy, code, or docs.
- "At most one active Headshot per account" is enforced by a **partial unique index** (`uq_third_place_headshots_active_user`, `WHERE removed_at IS NULL AND status = 'active'`), and `LeaveHeadshot`'s `INSERT ... ON CONFLICT (user_id) WHERE removed_at IS NULL AND status = 'active' DO UPDATE` targets that exact predicate. This is race-safe at the database level, not just "idempotent because the Go code checks first" — worth knowing if a future kernel is tempted to add a second write path to this table without going through `LeaveHeadshot`.
- `created` (new row vs refreshed existing row) is determined via the `(xmax = 0)` Postgres idiom in the same `RETURNING` clause as the upsert — a single round trip, no separate existence check beforehand. If you need this "was it actually inserted" signal elsewhere in the codebase, this is the pattern to reach for instead of a read-then-write.
- The table has **no column that could hold Face content** at all — `id, user_id, status, placed_at, removed_at, created_at, updated_at`. "History never snapshots old Face content" is a schema fact, not a policy someone could accidentally violate by adding a field.

### Live projection, not stored content

- A Headshot's stage name, portrait, and headline facts are recomputed on every single read via `playerprofile.ProjectTrailerFace` (Kernel 61A) — `internal/thirdplace` imports `playerprofile` and `playerrelationships` directly and duplicates none of their logic. If Trailer Face projection ever changes shape, Third Place picks it up automatically with no migration of its own.
- Relationship state (`Add to My People` vs `Open My Notes`) is computed per viewer via `playerrelationships.GetRelationshipBySubjectProfile`, called fresh inside `ProjectHeadshot` — never stored on the Headshot row, never visible to the owner or to any other viewer. A third project-owned test (`TestProjectHeadshotRelationshipStateForViewer`) specifically proves a second, unrelated viewer sees `none`/`Add to My People` even after a first viewer has already added the same owner to their own My People.

### Live update reuses Kernel 61A's existing socket unchanged

- `frontend/venues/third-place/index.html` opens one `watchPlayerProfile(profileId, ...)` (from `frontend/lib/player-profile-ws.js`, untouched) per visible Headshot card and refetches the whole commons list on any invalidation, rather than patching one card's fields in place. No backend change was needed for this — `/ws/player-profile` was already generic enough to reuse as-is.

### Browser-proof residue

- Disposable accounts `k65_owner_*`, `k65_viewer_*`, `k65_third_*` (one timestamp suffix, the final successful debugging run) remain on the live DB, holding no elevated privileges — left deliberately, per Kernel 62 precedent. Four earlier debugging runs' worth of the same pattern (12 accounts total) were found and deleted before finishing this kernel; if you see more `k65_*` accounts accumulate from a future re-run of `scripts/smoke/kernel65-third-place-browser.js`, the same cleanup is safe (`DELETE FROM users WHERE handle LIKE 'k65_%'` — cascades through the Headshot, relationship, and session rows via `ON DELETE CASCADE`).

## Kernel 66 — Show Run, Audience Program, and Roster MVP

### JSON struct tags are load-bearing, not cosmetic

- A Go struct returned directly through `writeOK(w, map[string]any{"show_run": sr})` must carry `json:"snake_case"` tags on every field, or `encoding/json`'s default (the capitalized Go field name, e.g. `"ID"`, `"ShowFormat"`) ships to the frontend silently wrong. This actually happened: `ShowRun`, `RosterMember`, and `AudienceBlock` in `backend/internal/showruns/types.go` were written without tags, compiled fine, and passed every unit test — because the tests assert on Go struct fields directly, never on marshaled JSON. It only surfaced when `fresh-install.sh --local`'s new HTTP-level assertions tried to parse a real response body. **Lesson: a struct that will ever cross an `http.HandlerFunc` boundary needs `json` tags from the moment it's written, and only an actual HTTP round-trip (fresh-install, live proof, or an HTTP-layer test asserting on raw response bytes) will catch a missing one.**

### `fresh-install.sh`'s migration list is a hardcoded array, not a glob

- `scripts/smoke/fresh-install.sh` applies migrations from a literal bash array (`migrations=(...)`), not by globbing `database/migrations/*.sql`. A new migration file existing on disk is not enough — it must be added to that array explicitly, or fresh-install silently stops one migration short with no error. Kernel 66's migration 039 was missed on the first run this way; caught only because the subsequent Show Run assertions failed with a table-does-not-exist-shaped error.

### No in-app Production-creation flow exists anywhere

- Both `producers-office`'s existing production picker and this kernel's own Show Run creation form only ever *consume* `GET /api/productions` — there is no endpoint or UI anywhere in Victory that creates a `productions` row. `grep -rn "INSERT INTO productions"` across the whole backend only turns up test fixtures. On a genuinely fresh install, `/api/productions` returns an empty list and neither producers-office nor Show Runs' create form has anything to offer. This is a pre-existing gap, not something Kernel 66 introduced; both `fresh-install.sh` and the live proof for this kernel worked around it by inserting a fixture/using a real pre-existing production directly. Worth a future kernel if it starts blocking real onboarding.

### Location-scoped vs. global authority

- `access.CurrentLocationRole(ctx, pool, userID)` (Kernel-16-era) ignores which location is actually in play — it returns the caller's single globally-best active role across *every* location they belong to. This was fine for `network/director_console.go`'s single-location-install use case but would be a real authority bug for Show Runs, where a Producer at one location must not manage a run at a different one. `access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)` (new) adds the missing `location_id` filter; `access.HasActiveLocationMembership` and `access.HasAnyManageableLocation` round out the set. Any future location-scoped feature should use the new functions, not `CurrentLocationRole`.

### New users get an active Audience location membership automatically

- Observed live, not by reading signup code: a freshly signed-up disposable account could immediately self-join a Show Run as Audience and saw the `show-runs` venue tile with zero manual role grant. This implies signup auto-grants an active `audience` `location_memberships` row at the neutral install location. Kernel 66 relied on this being consistent with `HasActiveLocationMembership`'s semantics but did not investigate or change the signup flow itself — worth confirming explicitly if a future kernel's authority model depends on it.

## Kernel 67 — Show Instance Model and Show Run Bridge

### `showings` was audited and deliberately left untouched — `shows` is a new, separate table

- Kernel 66's own dictionary note predicted that a future pre-live scheduling capability would extend the Kernel 22 `showings` table. Kernel 67 tested that prediction against the real code before trusting it, and found it unsafe: `showings.session_id` is `NOT NULL UNIQUE REFERENCES sessions(id)`, lazily created on first action-write via `EnsureForSession`, and read/written from 20+ call sites (every file in `backend/internal/actions/`, `network/session_control.go`, `identity/discord_mic.go`, `identity/join.go`). Loosening that constraint to let a Show exist before any session does would have touched all of them.
- **Lesson for future kernels: a prior kernel's own dictionary note is a prediction, not a fact — verify it against the current code before building on it, the same way you'd verify any other assumption.** This is the second time in two kernels this has mattered (Kernel 66 corrected an assumption in the *original* draft spec; Kernel 67 corrected an assumption *Kernel 66itself* had written down).
- The actual model built: a new `shows` table (`show_run_id → show_runs`, 7-value status enum distinct from Show Run's 5), plus a nullable `sessions.show_id` column. `showings` was not renamed, not schema-altered, and no existing call site referencing it was touched.

### Authority functions were exported, breaking the usual small-helper-duplication convention on purpose

- `backend/internal/showruns/authority.go`'s `canManageShowRun`/`canViewShowRun` were renamed to `CanManageShowRun`/`CanViewShowRun` (exported) specifically so `backend/internal/shows` could call the identical location-scoped authority check Kernel 66 built, rather than writing a second copy. This codebase's usual convention (see Kernel 66's `resolveProfileUserID` comment) is to duplicate tiny helpers per-package rather than introduce cross-package coupling for small things — that convention was deliberately broken here because authority/authorization logic drifting between two near-identical copies is a correctness and security risk, not a stylistic one. If you're searching for the old lowercase names and can't find them, this is why.

### Regression-testing an additive column is still worth doing explicitly

- Adding `sessions.show_id` as a nullable column is about as low-risk as a schema change gets, but Kernel 67 still explicitly re-ran the full `internal/network` and `internal/actions` suites (not just the new `internal/shows` tests) before calling this PASS, rather than assuming "nullable and additive" was self-evidently safe. It was — no regression — but the verification took two extra minutes and removed all doubt from the reportback.

## Kernel 68 — Venue Visibility Gates, Stage Management Surface, and Production Onboarding

### A spec's stated behavior can conflict with a pre-existing, unrelated rule — check what "new account" actually gets by default before writing the gate

- The spec asked for a Third Place readiness gate and, almost in passing, required "new account can see Trailers." Reading the *existing* `access.ResolveVisibleVenues` SQL showed Trailers and Third Place shared one `IN ('trailers', 'third-place')` branch requiring `producer/director/cast/crew` location role — and `identity/auth.go`'s signup flow grants a plain self-registered account only `audience` role. Those two facts together mean the literal required test ("new account sees Trailers") was **unsatisfiable without also changing Trailers' own rule**, something the spec's prose never explicitly asked for. Caught by tracing the actual default role a fresh signup gets, not by re-reading the spec more carefully — the spec can't tell you what a "new account" actually has by default, only the signup code can. Surfaced to the operator via `AskUserQuestion` before writing any code, since loosening a pre-existing restriction is a bigger behavioral change than "add a gate."

### A test helper's blank-string default can silently stop being blank-safe

- `thirdplace/headshots_test.go`'s `setStageNameAndFace(t, pool, userID, stageName, portraitURL, shortIntro)` treats `""` for the last two params as "don't set this field" — a deliberate, reasonable design for a fixture helper. But `http_test.go`'s `TestHandleMeFullLifecycle` called it with both trailing params blank, meaning the only fact ever created was a stage name-adjacent nothing — the account had a stage name and *zero* visible Face facts. That was invisible before Kernel 68 because nothing checked for a visible fact; the moment `HandleMe`'s POST started requiring Trailer Face readiness, this previously-passing test failed. **Lesson: when a new gate depends on "did the fixture actually create the state it looks like it created," go re-check what existing call sites of a shared test helper actually populate — a helper being *capable* of setting up real state doesn't mean every caller used it that way.**

### Tightening an authority check requires tracing every route that currently uses the looser one, not just the one route the spec names

- The spec's example rule for Stage Management ("visible if Operator OR Producer OR Director OR crew") was written as a map-visibility rule, but the map tile is not the only way to reach backstage data — `GET /api/show-runs/{id}`, `GET /api/show-runs/{id}/shows`, and `GET /api/shows/{id}` all previously used the same loose `CanViewShowRun` (any active membership) the Audience Program route uses. Hiding only the map tile while leaving those three reachable via a bookmarked/guessed URL would have satisfied the letter of "hide the tile" while leaving the actual backstage JSON just as open as before. Found by grepping every call site of `CanViewShowRun` across `showruns` and `shows`, not by re-reading the one HTTP handler the spec's example implied. The fix (`CanViewBackstage`, layered on top of `CanManageShowRun` rather than replacing it) had to thread through three handlers plus one listing query, not one.

## Kernel 69 — Scene Library and Show Staging Model

### A shared SQL column-list constant is only safe for the table shape it was written for

- This codebase's convention (`shows.go`, `showruns.go`, and now `scenes.go`) is a package-level `const xColumns = "id::text, ..."` string reused across every single-table `SELECT`/`INSERT ... RETURNING`/`UPDATE ... RETURNING` for that row. That pattern breaks silently-until-runtime the moment the same constant gets pasted into a multi-table `JOIN` query without table-qualifying its column names — Postgres returns `ERROR: column reference "id" is ambiguous` at query time, not a compile error. `scenes/placements.go`'s `ListPlacementsForShow` (a `show_scene_placements JOIN scenes`) hit exactly this. **Lesson: when writing a JOIN query, do not reuse an existing unqualified `xColumns` constant — write an explicit `p.`/`s.`-qualified column list for that query specifically**, even if it duplicates a few column names already present in the shared constant.

### Live-proof scripts against the real deployed server must send the session cookie manually, not via curl's cookie jar

- `docker-compose.yml` sets `COOKIE_SECURE` to default `true`, so every `Set-Cookie: victory_session=...` response from the live backend carries the `Secure` attribute. curl's cookie jar (`-c`/`-b`) correctly refuses to re-send a `Secure` cookie over a plain-HTTP connection — and a live-proof script running inside a disposable container on the Docker-internal network (`victory_victory_internal`) talks to `victory-backend:8081` over plain HTTP, since Caddy is the only TLS terminator and sits outside that path. The fix is not a code change (the `Secure` behavior is correct and must not be loosened) — extract the raw token from the `Set-Cookie` response header with a one-line `grep`/`sed` and pass it back explicitly as `-H "Cookie: victory_session=..."` on every subsequent request instead of relying on `-b`/`-c`. **Any future live-proof script that talks to the real deployed container over the internal Docker network (as opposed to `fresh-install.sh`'s own disposable, non-Secure-cookie stack) will hit this and should use the manual-header approach from the start.**

## Kernel 70 — Persistent Show Stage, Rehearsal Workspace, and Go Cue Foundation

### A column rename breaks migration replay unless the whole rename is gated, not just the new migration

- This repo's `setup-test-database.sh`/deploy flow has no migrations-tracking table — it replays every `.sql` file in `database/migrations/` from scratch on every run, relying entirely on each file's own idempotency (`IF NOT EXISTS`, `ON CONFLICT`, etc.). A rename in a *later* migration (`scenes.production_id` → `source_production_id` in 043) silently breaks an *earlier* migration's replay (042's `CREATE INDEX ... ON scenes(production_id)`), because — confirmed empirically — `CREATE INDEX IF NOT EXISTS` still fully resolves its column references even when the index name already exists and it's about to no-op; `CREATE TABLE IF NOT EXISTS` does not have this problem, since it short-circuits before validating its body at all. These two `IF NOT EXISTS` variants are not equivalent in this respect.
- The rename statement itself (`ALTER TABLE ... RENAME COLUMN`) is also not naturally idempotent — there is no `RENAME COLUMN IF EXISTS` in PostgreSQL — so simply re-running the renaming migration a second time fails too, since the source column no longer exists.
- **Fix pattern used**: (1) remove/adjust the stale line in the *earlier* migration so it no longer references the renamed column at all (safe — it's non-destructive index-creation SQL, not data), and (2) wrap the entire rename+backfill+constraint-swap sequence in a single `DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE ...) THEN ... END IF; END $$;` block. PostgreSQL only resolves a PL/pgSQL branch's column/table references when that branch actually executes — an untaken `IF` branch is never parsed against the catalog, unlike a top-level `CREATE INDEX IF NOT EXISTS` statement. **Lesson: any future column rename must (a) audit every earlier migration for `CREATE INDEX`/`ADD CONSTRAINT`/other DDL referencing the old name and neutralize those lines, and (b) put the actual rename inside a guarded PL/pgSQL block, not bare top-level SQL — verify by actually running the full migration replay twice (fresh install, then re-run setup on top of the already-migrated database), not just once.**

### A struct field's JSON tag is a leak surface the moment the struct is reused across a backstage and an audience-facing handler

- `Show.VariablesJSON`/`CurrentShowScenePlacementID` were added as normal `json:"..."`-tagged fields, following the exact pattern every other Show field already uses. This was wrong: `HandleShowProgram` (Audience-viewable, gated only by `CanViewShowRun`, which any active location member including Audience passes) does `writeOK(w, map[string]any{"show": s, ...})` — serializing the *entire* `Show` struct, not a curated subset. Any backstage-only field added to `Show` this way ships straight to Audience, silently, with no per-endpoint code change needed to cause the leak.
- Caught by a tripwire test (`json.Marshal` a Show with a known secret string in the new field, assert the string and the field name are both absent from the output) mirroring the same technique Kernel 69 used for `director_notes` on `AudienceScenePlacement` — but Kernel 69's curated type was a *separate, narrower* struct built for exactly one audience-facing purpose, so this class of bug couldn't occur there. `Show` is a single wide struct reused by both backstage and audience-facing handlers, which is the actual root cause.
- **Fix**: `json:"-"` on both fields, with backstage handlers (`HandleShowByID`, already gated behind `CanViewBackstage`) adding them explicitly to their own response map instead. **Lesson: before adding a new field to any Go struct that gets serialized by more than one HTTP handler, grep every handler that touches that struct and check the weakest authority gate among them — if any of those handlers is audience-reachable, the new field needs `json:"-"` plus explicit backstage-only exposure, not a plain JSON tag, regardless of how "obviously backstage" the field's purpose seems.**

### A same-named sibling file one directory up can be the real bootstrap for a module tree that looks unwired

- `frontend/venues/{first-theater,catharsis}/runtime/` contains a `session-sync.js`/`socket.js`/`socket-controller.js` module tree with UMD factory exports (`createSessionSync`, `createSocketController`) that are never referenced by name anywhere inside `runtime/` itself or in `index.html`'s static `<script src>` tags — grepping for the factory names across the venue directory (excluding the files' own definitions) returns nothing, which looks exactly like an orphaned, never-finished refactor. It is not: the real caller is `frontend/venues/{first-theater,catharsis}/runtime.js` — a *sibling* file at the venue root, same base name as the `runtime/` directory but not inside it — which `index.html` loads via a dynamically-assigned `script.src = "/venues/.../runtime.js?v=..."` rather than a static tag, so a plain grep of `index.html`'s `<script src=` lines misses it entirely.
- Confirmed genuinely live (not just theoretically reachable) by finding real `node --test`-run unit test files at `/opt/victory/tests/first-theater/*.test.js` exercising these exact modules, and by tracing `runtime.js`'s `createSessionSync({...})`/`createSocketController({...})` calls and their downstream `sessionSync?.applySnapshot?.(snapshot)`/`sessionSync?.handleSocketMessage?.(msg)` wrapper functions.
- **Lesson: before concluding any module/factory in this codebase is dead code, (1) check for a same-named-but-un-suffixed sibling file at the parent directory level (`runtime.js` next to a `runtime/` folder is exactly this shape), (2) check for dynamic `script.src =` assignment, not just static `<script src=` tags, and (3) check `/opt/victory/tests/` for a `node --test` file exercising the suspect module before assuming it's unreachable.** All three checks together took real time but were the only way to avoid silently shipping changes to genuinely dead code.

## Kernel 71 — Two-Punch Show Tickets, Character Participation, and Showtime

### A shared roster-add function's `ON CONFLICT DO UPDATE` upsert path is a second, easy-to-miss bypass route

- The obvious fix for "an ordinary Director shouldn't be able to unilaterally grant Player status" is to gate the *update* path (`UpdateRosterMemberRole`, called when PATCHing an existing row's role). That alone is not enough: `AddRosterMember`'s single `INSERT ... ON CONFLICT (show_run_id, user_id) DO UPDATE SET role = $3` statement can *also* silently promote an existing non-player row to Player, via the exact same call a Director would use to add someone as `guest`/`crew` in the first place — re-calling it with `role="player"` for a user who already has any active row reactivates/promotes them through the INSERT path's own upsert, never touching `UpdateRosterMemberRole` at all. **Lesson: when gating a role transition behind new authority, check every function whose SQL includes `ON CONFLICT ... DO UPDATE` against the same columns you're trying to protect — an "add" function is often also a disguised "update" function, and a single-purpose guard on the function named `Update*` will miss it.** Fixed by adding the identical Operator-only guard to `AddRosterMember` itself, unconditionally (both the fresh-insert and the upsert-promotion cases), rather than trying to distinguish them.

### Go's test cache will hide a shared function's behavior change in packages you didn't touch

- After gating `AddRosterMember`/`UpdateRosterMemberRole` behind a new authority check, `go test ./...` (no flags) reported everything green — including three pre-existing tests, in `cues`, `shows`, and `showruns` itself, that called `AddRosterMember(..., "player", ...)` directly as an ordinary (non-Operator) actor and should have started failing immediately. They didn't fail, because Go's test cache had a cached PASS result from before the guard was added, and none of those three test *files* had been edited in this session, so `go test` considered them unchanged and skipped re-running them. Only `go test -count=1 ./...` (which forces every package to actually re-execute, cache or not) surfaced the three real failures. **Lesson: after changing the behavior of any widely-shared function (not just its signature — a pure behavior/authority change on an unchanged signature is invisible to Go's staleness check in the same way), always run at least one full `-count=1` pass before calling the suite green — a normal cached `go test ./...` run only proves "packages I edited still pass," not "nothing downstream broke."**

### `psql -tAc` on an `INSERT ... RETURNING` can concatenate the command tag onto the result if you strip all whitespace at once

- A smoke-test fixture did `docker exec ... psql -tAc "INSERT INTO character_cards (...) VALUES (...) RETURNING id;" | tr -d '[:space:]'` to capture a generated UUID into a shell variable — a pattern that had worked reliably elsewhere in the same script for plain `SELECT` queries. For this `INSERT ... RETURNING`, psql's tuples-only (`-t`) output still included a trailing `INSERT 0 1` command-completion line on its own line below the returned UUID; `tr -d '[:space:]'` strips the newline between them along with everything else, silently concatenating `<uuid>` and `INSERT01` into one malformed string that then failed a downstream query with `invalid input syntax for type uuid`. **Lesson: when capturing a single value from `psql -tAc` for an `INSERT`/`UPDATE`/`DELETE ... RETURNING` (as opposed to a plain `SELECT`), pipe through `head -n1` before stripping whitespace, so only the actual result line survives — blind `tr -d '[:space:]'` alone is only safe for query shapes that never print a trailing command tag.**

### A new route pattern can collide with an existing Go 1.22+ `http.ServeMux` wildcard pattern in a way `go build`/`go vet` never catches

- `GET /api/shows/by-code/{code}` was registered alongside the already-existing `GET /api/shows/{show_id}/program`. Both compile fine and both pass `go vet` — the collision is a *runtime* panic at `mux.HandleFunc` registration time (`ServeMux.register` calls `pattern.conflictsWith`), because Go's pattern-matching can't statically prove `by-code` will never be used as a `{show_id}` value, and vice versa; two same-segment-count patterns with a wildcard in different positions are genuinely ambiguous to the router, not just to a human skimming the route table. This was caught immediately by `scripts/smoke/fresh-install.sh` (the backend fails to start at all, a hard crash, not a silent misroute) — but only because that script actually boots a real compiled binary; `go build`/`go vet`/unit tests never exercise `main()`'s route registration and would have shipped this straight through. **Lesson: any new route whose first path segment could ever collide with an existing wildcard segment at the same position (e.g. adding a literal-first-segment route next to an existing `{id}`-first-segment route one level deeper) needs an actual `fresh-install.sh`-style real-server-boot check, not just `go build`/`go vet` — prefer a query parameter over a new path segment when in doubt, since query parameters never participate in `ServeMux` pattern-conflict resolution at all.**

## Kernel 80 — Storyboards Core

### `network` importing both `identity` and `ewrite` (transitively) means neither can import a package that itself needs `network`

- `network` imports `identity` directly and `ewrite` transitively (`network` → `actions` → `characters` → `ewrite`). `storyboards` needs `network` for its live WS events (board-watch primitives on `Hub`), which means `identity` → `storyboards` and `ewrite` → `storyboards` would each close a cycle the moment either package tried to call into `storyboards` directly (e.g. for an authority check or a data lookup). Both `ewrite/links.go` (`storyboardCardEditorAllowed`) and `identity/account_deletion.go`/`account_export.go` hit this and worked around it by inlining the small amount of raw-SQL/authority logic they actually needed, with a comment pointing at the canonical version in `storyboards/authority.go` to keep in sync. **Lesson: before adding a cross-package call from `identity` or `ewrite` into any package that imports `network` (or any package `network` itself depends on), check the import graph first — `go build` will refuse silently late, not warn early, and the fix is almost always "duplicate the small piece you need with a comment," not "restructure the dependency direction."**

### Two floating nav widgets assume every page has a `<header>` tag to dock into, and fall back to overlapping fixed positioning otherwise

- `venue-account-badge.js` and `back-to-map.js` both look for `document.querySelector("header.header") || document.querySelector("header")` to find a place to render inline; if neither exists, they silently fall back to `position: fixed` in a page corner (top-right / top-left respectively) with a fairly high z-index. Storyboards' two new pages use a `.page-header` `<div>`, not a `<header>` tag (matching several other venues, e.g. writers-room), so both widgets fell back to floating — and one page's real header-action buttons (Sharing/Export/Archive, pinned far-right by `justify-content:space-between`) rendered directly underneath the account badge, silently intercepting clicks. Found only by real browser automation (Playwright), not by any static check. **Lesson: any new page without a literal `<header>` tag needs deliberate CSS clearance (padding-left/right on whatever sits in the corners) for both of these widgets, or should include an actual `<header>` element with a `.launchbar`/`.toolbar`/`.header-right`/`.top-actions` child for them to dock into properly instead of floating at all.**

### A written kernel spec is not proof of product intent — verify against the actual mental model before treating "matches the spec text" as done

- Kernel 80's spec explicitly required "cells support multiple ordered cards" as a named, tested pass criterion, and left card visual design, drag-and-drop primacy, column-insertion position, and color palette unconstrained. All of it was built exactly as written, with full test/browser evidence. A live post-deploy design review then surfaced that several of these choices don't match what Grant actually wanted (single card per cell, not multiple; drag-and-drop as the primary interaction, not a secondary fallback; cards that visually match the existing canvas index-card, not a plain bordered div; the red/rose/black palette used almost everywhere else, not writers-room's amber/gold). None of this was a spec violation — it was the spec under-specifying (or, in the multi-card case, apparently mis-specifying) the actual product intent. **Lesson: for any kernel with a significant new UI surface, a spec that reads as complete and internally consistent is not the same guarantee as "matches what the operator pictures" — where the spec is silent or where a requirement seems surprising relative to how the rest of the product looks/behaves, it's worth a direct one-line confirmation before building the untested assumption at full scope, rather than after.** See `Construction/OperatorLogs/kernel-80-reportback.md` §8 for the full accounting and the scoped continuation this produced.

## Kernel 74 — Locked Door Intentions, Ra Guided Dialogue, and Participant-Local Tutorial Handoff

### An "omit from the payload" authority gate protects discovery, never invocation

- The door hotspot was gated by removing the element from a Player's snapshot until they had recorded `kessa_intro_completed` — a genuinely strong reveal gate: a Player with no milestone has no door in their payload at all, so no amount of client-side forgery can render one. That was mistaken for the whole gate. A Player who learned the interaction id by any other route (a second Character who had already unlocked it, a shared screen, a replayed request, a Director reading it aloud) could still POST straight to the endpoint and be served, because the *endpoint* never checked the milestone — only the snapshot builder did.
- The kernel spec had asked for both halves separately ("client-forged progress does not reveal the door" AND "Player cannot unlock the door before Kessa completion"), and it was easy to read the second as merely restating the first. It does not. Fixed with `merchant.requireBindingMilestone`, re-checking the same milestone against the same rows on open/submit/dialogue. **Lesson: whenever an authority gate is implemented by omitting something from a response, write down separately what stops a caller who already knows the identifier — visibility filtering and invocation authorization are two different gates, and a payload-shaping gate is worth exactly nothing against a direct request.**

### A UI control that cannot act must still be prevented from blocking

- An `interaction_hotspot` left unbound (no participant interaction attached) still rendered its full-size, pointer-capturing PIXI region. On the live Courtyard this box sat over Kessa's token and swallowed every click meant for her, answering with a polite "cannot be used right now" — so the *reported* symptom was "I can't click Kessa," several layers away from the actual cause. The element had been left behind by a cleanup pass and was assessed as harmless because it could not *do* anything; nobody asked what it still *rendered*.
- Two rules came out of it, both now regression-tested: an unbound or disabled hotspot is fully inert (no pointer events at all, invisible to non-backstage viewers, faint outline for authoring), and hotspots render BENEATH tokens — a hotspot is a region over map art, and map art belongs behind the people standing on it. **Lesson: when deciding whether a non-functional UI element is safe to leave in place, the question is not "can it do anything?" but "does it still occupy space, capture input, or draw?" — an inert-by-authority element that is still interactive-by-rendering is a strictly worse failure than a working one, because its error message points away from the real problem.**

### A `kind`-gated coordinate conversion silently ignores every new kind you add

- `geometry.js` converted Kernel 73A's normalized 0-1 composition coordinates into stage pixels inside `if (kind === "token")`. Adding `interaction_hotspot` as a new composition kind meant hotspots fell through to a generic mid-stage fallback that ignores stored position entirely — so the door rendered in the middle of the map regardless of what was saved, `y: 0.35` and `y: 0.075` were pixel-identical, and re-authoring the position in Scene Setup appeared to do nothing at all. This masked the two bugs above: all three presented to the operator as the single symptom "the box is in the wrong place."
- **Lesson: when adding a new `kind`/`type` value to a system that already stores data in a special coordinate space, grep for every `kind ===` / `switch (kind)` branch that reads position, size, or geometry BEFORE testing visually — a missed branch does not error, it silently falls back to a default that looks like a tuning problem and will absorb hours of re-tuning a value that is never read.**

### Scoping a new per-participant table: check every sibling table's key, not just the new one

- Kernel 74 added four Character-scoped tables (`participant_tutorial_progress`, `participant_freeform_submissions`, `participant_dialogue_topic_views`) and one that was not: `participant_local_projections` carried a `character_card_id` column but keyed its uniqueness index and its lookup on `(user_id, show_id)` only. The consequence surfaced only in live play — a Player who finished the tutorial with one Character and then switched stayed stranded on the tutorial-handoff map, presenting as a Character who had never met Kessa standing outside a courtyard they had never left.
- The test suite had a Character-switch isolation assertion, but it only covered the door hotspot — the thing the gate had just been built for. The same question was never asked of the projection. **Lesson: when a kernel introduces several tables that share a scoping rule, write the scoping assertion as a loop or checklist over ALL of them rather than proving it once on the feature you were thinking about — and when one table in a family carries a column it does not use in its keys or queries, treat that as a defect signal, not as headroom for later.**

### Victory has two independent "current Character" values, and they did not talk to each other

- `active_user_characters` (set by the Greenroom picker via `/api/character-cards/{id}/activate`) and `show_run_roster_members.character_card_id` (set by Audition Hall, and made canonical for Show participation by Kernel 71) were fully independent. Every Kernel 74 surface keys off the roster selection, so switching Characters in the Greenroom changed nothing about tutorial state — the operator switched Characters, observed no change, and reasonably reported it as a Kernel 74 bug.
- Resolved on the operator's explicit approval by making `ActivateCharacterCard` write through to the roster selection (scoped: `player` rows only, non-archived Show Runs only, same Location as the Character). **Lesson: before keying new per-participant state off an existing "currently selected X" column, grep for every other place the product lets a user change X — if more than one writer exists and they do not write through to each other, the new feature will appear broken in exactly the surface the user actually uses, and the bug report will land on the new kernel rather than on the pre-existing split.**

### A smoke script must never end a live session it did not open

- `scripts/smoke/kernel74-tutorial-browser.js` needs an active Session at Catharsis, and live production had one open since 2026-07-04 belonging to a different Show. Because `world.LoadVenueSnapshot` resolves the most recent active Session at a venue, every Player read in the script resolved someone else's Show and failed with `no_active_session`. The tempting fix — end the stale-looking Session and carry on — would have silently disrupted the operator's own live rehearsal state.
- The script now preflights, compares the resolved `show_id` against its own, and aborts with an explicit PRECONDITION message naming the conflicting Show and offering `K74_VENUE_SLUG` as the alternative. **Lesson: any smoke/acceptance script that needs exclusive use of a shared live resource must detect a foreign claim and stop with a message naming it, not reclaim it — and the backend-test convention of standing up a dedicated throwaway venue (`prepare_end_to_end_test.go`) exists precisely because this hazard is not unique to browser scripts.**

## Kernel 81 — Storyboards Presentation Rework

### A computed layout value must have exactly one owner, or two individually-correct computations will silently collide

- `computeGridLayout` (`grid-model.js`) assigns every structural element a CSS Grid line number. The "+ row in Band X" button's line was originally computed a second time, independently, directly in `board.html` (`lastMemberRow's line + 1`, or `bandHeaderLine + 1` if the band had no rows yet). Both computations were individually reasonable and both compiled/rendered without error — but whenever a band had exactly one row, they landed on the same line number as the *next* band's bar, because `computeGridLayout`'s own running line counter had no idea a "+row" button was going to consume a line board.html hadn't told it about. The only symptom was a real click failing: Playwright reported `<span>Act Two</span> ... subtree intercepts pointer events` on the "+ row in Band 1" button. No static check catches this class of bug — both line numbers are individually valid-looking integers; the defect only exists in the relationship between two computations that don't know about each other.
- Fixed by making `computeGridLayout` reserve the line itself (`addRowLine`), even for viewers whose role won't render anything into it — an unconditionally-reserved-but-sometimes-empty line is simpler and safer than a conditionally-reserved one. **Lesson: when a layout/ordering scheme has one authoritative "next available line/index/slot" counter, every consumer of that scheme — including UI affordances that only render for some viewers — must go through the same counter, never compute its own "one past the last real element" independently. If two pieces of code both compute "the next line," they will eventually disagree, and the disagreement will not show up until a specific data shape (here: exactly one row in a band) makes them collide.**

### A resource's storage/quota scope and its real access-control authority are not always the same thing

- Every Victory asset (`assets` table) is stored and quota-accounted against a producer + location — this was true from long before Storyboards existed, and `userCanReadAsset`'s authorization check is built entirely around that scope (location membership). Storyboards card images are uploaded into that same system (they have to be — it's the only asset-storage machinery in the repo), scoped to the *board owner's* producer membership since boards have no Production of their own. But a card image's *real* authority isn't "who has a role at the owner's storage location" — it's "who has a Storyboard grant on the board the card belongs to." These two things happen to coincide for most real users of a single-family-instance deployment (most people ARE members of the one shared location), which is exactly why a throwaway test account with genuinely zero location memberships anywhere was needed to expose the gap: a real, legitimately-shared Storyboard collaborator with no other relationship to Grant's location got a flat 403 loading an image they were plainly authorized to see the card containing.
- Fixed with a targeted early-branch check in `userCanReadAssetConsideringEwrite` (mirroring the existing eWrite-publication branch already there for the same class of problem — Kernel 79 had already solved this exact shape of bug once for eWrite-embedded images) rather than trying to make location membership itself aware of Storyboards. **Lesson: whenever a new feature reuses an existing storage/quota system by threading its own resource through that system's existing scope (a producer, a location, an owner), audit that system's *authorization* check separately from its *storage* scope — they were designed to move together, and a new caller can easily satisfy one while violating the other's actual intent. If the existing authorization check already has a precedent for "resource's real authority lives elsewhere" (eWrite's publication-visibility branch did), extend that pattern rather than inventing a new one.**

## Missing `permission_requests` table (2026-08-07, not a kernel — a bug fix)

### A feature can be fully built, routed, and documented in its own UI copy, and still never have worked for anyone but its author

- `permission_requests` was read and written by `identity/requests.go` and `permissions.go` since the feature first shipped (2026-03-30, pre-kernel era) — with no migration ever creating the table, in either the live production database or the test database. Every `/api/requests/create` call, including Audition Hall's own documented default flow ("The default here is auto-access for Catharsis"), has been failing with a 500 the entire time. This was found only because a genuinely new second account (not the pre-existing Operator account) finally exercised the path for the first time in the project's history.
- Two things conspired to hide it completely: (1) the only account that existed before this fix was the Operator account, which — being an Operator — never needed to submit a request through this flow at all; (2) the one OTHER read of the same table, `access/visibility.go`'s notification-badge-count query, is wrapped in `if err == nil` — a missing table there just silently skips badge counts rather than surfacing anywhere. The two write-path handlers (`requests.go`, `permissions.go`) have no such tolerance and hard-fail, but nothing had ever called them.
- Zero test coverage existed for either handler before this fix, which is exactly the gap that let it ship silently non-functional and stay that way for months. **Lesson: "the code compiles, the route is registered, the UI defaults are documented" is not evidence a write path actually works — only a real end-to-end exercise (a test, or in this case a real second user) proves a table you're inserting into actually exists. When a new account-facing flow has literally never been used by anyone but the person who wrote it, treat "nobody's hit a bug" as "nobody's tried it" rather than "it's solid." A read path that tolerates a missing dependency can mask a write path that doesn't, for an arbitrarily long time — audit both together, not just the one that happens to be exercised by normal usage.**

### An unconstrained (no-FK) reference column is sometimes the *more* correct design, not a shortcut

- `storyboard_cards.image_asset_id` deliberately carries no `REFERENCES assets(id)` clause. Two independent, pre-existing asset-lifecycle paths would each defeat a real FK in a different way: `identity/account_deletion.go` hard-`DELETE`s an uploader's own unused assets (a `RESTRICT` FK would block account deletion outright; `CASCADE` would silently null/lose the reference the same way `SET NULL` would); `assets.tombstoneWarehouseAsset` soft-deletes the row in place (a FK survives this one fine, but only this one). The product requirement — "a deleted image leaves a gravestone, not silent disappearance" — specifically needs the reference to survive *both* paths, including the one no FK action could satisfy. **Lesson: before defaulting to "add a foreign key" for a new reference column, check what every existing deletion path for the referenced table actually does — if the referenced entity has more than one deletion mechanism (a soft-tombstone AND a real hard-delete, here), a single FK action can only correctly handle one of them, and an intentionally unconstrained column plus an application-level "resolve or gravestone" convention at read time may be the only design that satisfies every path.**

## Kernel 81A — Storyboard Structural Slugs (2026-08-07)

### A generic "retry on unique-violation" handler must check *which* constraint fired, not just that one did

- `allocateUniqueSlug`'s retry loop originally treated any Postgres `23505` from its `insert` callback as "the slug candidate was taken by a concurrent writer, try the next one." `storyboard_columns`/`storyboard_bands`/`storyboard_rows` each also carry an older, unrelated `UNIQUE(scope, sort_order)` constraint (Kernel 80) whose value is computed via an unsynchronized `SELECT COUNT(*)` before the `INSERT` — a genuine, pre-existing concurrency bug that nothing had ever exercised until this kernel's own concurrency test finally ran multiple structural creations against the same board at once. When that constraint fired instead of the slug one, the generic handler misdiagnosed it as a slug collision and retried with a *different slug* — which can never fix a `sort_order` collision, since the retry loop never touches `sort_order` at all. The result was a silent, slow march to `ErrSlugAllocationExhausted` that reported the wrong problem entirely.
- Fixed by checking `pgErr.ConstraintName` for the specific index name before retrying; every other unique-violation now propagates as the real error it is. **Lesson: a "retry on unique-violation and try the next candidate" pattern is only safe when there is exactly one unique constraint that candidate could ever violate. The moment a table carries more than one — even one that predates and has nothing to do with the feature you're adding — a generic `pgErr.Code == "23505"` check will eventually catch someone else's collision and retry in a way that can never resolve it. Check the constraint name, not just the SQLSTATE, whenever more than one unique constraint could plausibly fire on the same statement.**

### Finding a real pre-existing bug while testing a new, narrowly-scoped feature does not obligate you to fix it there

- The `sort_order`/`sort_order_in_band` race above is real and still live in `AddColumn`/`AddBand`/`AddRow` today — two genuinely concurrent structural-creation requests against the same board can still fail with a raw constraint-violation error, independent of anything Kernel 81A touched. Fixing it was explicitly out of scope ("keep this bounded... do not redesign Storyboards"), so it wasn't fixed — only worked around narrowly enough that it can't corrupt the *new* feature's own correctness guarantee, and the concurrency regression test retries at the client level (exactly what a real caller already has to do against this pre-existing hazard) rather than silently absorbing it into the new code. **Lesson: "I found a bug while testing something else" and "I should fix this bug right now" are different decisions — when a task has an explicit scope boundary, the right move is to (1) make sure the new work doesn't get corrupted or mask the old bug, (2) document the old bug clearly enough that it isn't lost, and (3) leave it for a task whose scope actually includes it. Silently fixing it feels helpful but violates the boundary the user set; silently ignoring it (or worse, letting a retry loop paper over it) is worse.**

## Kernel 82 — Storyboards Timeline Mode

### A failed browser-proof assertion is a bug report, but the bug might be in the test's own setup, not the feature

- The Kernel 82 Playwright proof's first run reported `pan_moved_vertically: false` — the middle-mouse pan code moved `scrollLeft` correctly but `scrollTop` never changed. The reflexive read is "vertical panning is broken, symmetric bug to the horizontal case that just happened to work." Before touching any code, the actual DOM state was checked: `document.getElementById('board-scroll').scrollHeight` was **not greater than** its `clientHeight` at that point in the test — the board simply didn't have enough rows yet to ever produce a nonzero `scrollTop`, regardless of what the panning code did. A dedicated follow-up script added rows until real vertical overflow existed (`scrollHeight: 1202` vs `clientHeight: 547`) and the identical pan gesture moved `scrollTop` from 518 to 655 immediately.
- **Lesson: when a browser-proof assertion fails on one axis/dimension/branch but not its structural twin, check whether the *precondition* for that assertion (here: "there is anything to scroll") actually held before assuming the code path itself is asymmetric or broken. The fix for a false-negative test result is a better test setup, not a code change chasing a bug that was never there — and the fastest way to tell the two apart is to inspect the actual runtime state (`scrollHeight`/`clientHeight`, in this case) at the moment of failure rather than re-reading the implementation looking for what "must" be different between the two code paths.**

### A reusable "queryRower" abstraction pays for itself the second time it's needed, not just the first

- Kernel 81A's `allocateUniqueSlug` was written to accept a `queryRower` interface (satisfied by both `*pgxpool.Pool` and `pgx.Tx`) specifically so it could run either standalone (`AddColumn`) or inside an already-open transaction (`CreateBoard`'s atomic default-column/band/row creation). Kernel 82's `CreateTimelineBoard` needed the exact same shape — a multi-insert transaction that also needs board-scoped unique slugs (columns, a band, a row, and now four Reference Panel fields, all inside one transaction) — and required zero changes to `allocateUniqueSlug` itself to reuse it a third time. **Lesson: when a helper's interface is generalized slightly beyond its first caller's literal need (here: "works inside a transaction too," not just "works standalone"), that generalization is easy to dismiss as speculative/premature at the time. It stops looking speculative the moment a second, unrelated feature needs the exact same shape without asking for a new capability — the cost of the slightly-wider interface was paid once, and it was already amortized by the second use.**

## Kernel 83 — Venue Leadership & Turn State

### When a spec asks for a "generic capability" and names one consumer, build the split as two files from the start, not one file to refactor later

- Kernel 83 could have been written entirely inside `backend/internal/storyboards` — Storyboards was the only consumer named in the spec, and nothing would have failed a test if Group Leader/Current Turn logic just lived there with a comment saying "this is meant to be generic." Instead it was split from the first line of code into `backend/internal/venuecoordination` (a package that imports nothing feature-specific and has never heard of a "board" or a "tier") and `storyboards/coordination.go` (Storyboards' own authority wrapper, the only place that resolves a viewer tier or touches `storyboard_grants`). **Lesson: "generic capability, one real consumer for now" is a request to draw the module boundary where the *future* consumer will need it, immediately — not a request to build it coupled and extract it later. The cost of the split (one extra file, one extra layer of function calls) is trivial; the cost of *not* splitting is a second venue's kernel starting with a refactor instead of a wrapper file.**

### Finding the narrowest existing "session-shaped" primitive is real design work, not a formality

- Storyboards had no session/presence concept at all before this kernel (Kernel 80's own code comment said so). The spec required session-scoped state anyway, and explicitly anticipated this exact situation: "investigate the narrowest existing lifecycle primitive and document the chosen mapping." The actual investigation mattered — `network.Hub.BoardWatcherUserIDs` (built in Kernel 80 for an unrelated purpose, per-viewer hidden-card event delivery) turned out to already answer "who is live here right now" precisely enough to define session start/end as its 0↔1 watcher-count transitions, with zero new schema. The alternative — inventing a new `storyboard_sessions` table with explicit open/close semantics — would have worked too, but would have been a second, parallel concept to the watcher-tracking that already existed, and would have needed its own reconciliation logic for "what if the last watcher's tab just closes without a clean session-end signal" (which watcher-count transitions handle for free, since a closed tab simply stops being a watcher). **Lesson: before adding a new stateful concept to make a feature's lifecycle requirements clean, check whether an existing piece of ephemeral, in-memory, connection-scoped state already answers the same question the new concept would — session-shaped meaning is often already implicit in what "currently connected" already tracks.**

### An assumption baked into a testing *technique* doc can go stale independently of the product it documents

- The Two-Browser Verification Technique in `kernel-maker-field-guide.md` (written during Kernel 61/61A) led with `POST /api/auth/signup` for disposable test accounts, and every Storyboards kernel since (62, 65, 80, 81, 82) used it successfully without incident. Kernel 83's proof hit `password_signup_closed` on the first attempt — production had moved to Discord-only login at some point between those kernels and this one, and nothing about *this* kernel's own work would have surfaced that; it was purely bad luck that Kernel 83 happened to be the one that needed five fresh accounts on the day this had already changed. The field guide already documented a fallback (direct `auth.sessions` fixture-row insertion) for the narrower "existing account, unknown password" case; it worked immediately once applied to "brand-new account" too. **Lesson: a reusable technique doc that depends on a specific external endpoint's availability is itself a piece of product surface that can drift out of date, silently, without any change to the code the technique was written to test. When a documented technique's first step fails, don't debug the immediate error in isolation — check whether the technique's own precondition changed, and if a fallback already exists in the same doc, prefer it over inventing a new one.**

## Kernel 84 — Canonical Reconciliation & Runtime Cleanup

### A flake three kernels called "accumulated test-DB state" was actually a one-line test bug — the label stuck because no one re-ran the suite twice in a row without a reset

- `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent` had been failing intermittently since at least Kernel 81A, and every reportback that hit it (81A, 82, 83) explained it the same way: "pre-existing accumulated test-DB state from many prior sessions," always confirmed clean by doing a fresh `reset-test-database.sh` and re-running once. That explanation was never actually verified against the *mechanism* of accumulation — it was pattern-matched from a real, different, earlier incident (fixtures that depended on the live database's organic state, Kernel 64). Kernel 84 ran the full suite twice in a row on the *same freshly-reset* database, minutes apart, no other session involved, and reproduced the failure on demand. The real bug: the test asserted a hardcoded absolute revision number for an edit it made itself, which only happened to be correct the first time any test in that database's lifetime touched the seeded publication. **Lesson: "confirmed clean after a reset" only proves the state *before* your one test run was clean — it says nothing about whether your own test leaves the database in a state that would break a second identical run. If a flake's accepted explanation has never actually been reproduced on demand, it hasn't been diagnosed, it's been guessed at with a plausible-sounding label that happened to match a real, earlier, different bug.**

### A generic capability's "reconciliation" kernel is exactly when its edge cases surface — treat a documentation kernel's own required regression pass as real testing, not paperwork

- Kernel 84's spec required a "focused live browser regression," explicitly not a re-proof of anything already proven — the kind of task that's easy to treat as a checklist formality once the interesting code-repair work (the WS context fix) is done. Writing that regression script for real (not a stub) surfaced a genuine, previously-unknown Storyboards bug: appending a column via the generic "+ Column (end)" action on a Timeline board places it after the `Ending` boundary column, because `AddColumn` has no `column_role` awareness at all — only the UI's "Insert left/right" menu items avoid this, via a client-side create-then-reorder call the raw API path doesn't get for free. Nothing about Kernel 84's own spec asked for a Timeline-column stress test; it fell out of building filler columns to force horizontal overflow for an unrelated middle-pan check. **Lesson: a "boring" reconciliation/regression kernel's required smoke tests are not lower-value than a feature kernel's — real interaction with a real system under any new angle (here: adding several columns programmatically, something no prior kernel's proof had done) reliably finds gaps that manual code review of the "interesting" changes alone would not. Write the regression script to actually exercise the system, not to perform the minimum needed to tick the box.**

## Kernel 87 — Cartograph: Playwright click hangs and fixed-coordinate collisions with floating UI chrome

### Locator-based `.click()` can hang indefinitely in this sandbox even when every actionability check has already passed — use raw `page.mouse` coordinates instead

- Across several Kernel 87 verification runs, Playwright's `page.click(selector)` against `.cartograph-toolbar` buttons intermittently hung well past any sane timeout. The call logs showed every actionability step succeeding (`element is visible, enabled and stable`, `scrolling into view`, `done scrolling`) and then stalling silently at `performing click action` (or, once, past `click action done` at `waiting for scheduled navigations to finish` — nothing about a plain toolbar button click should ever wait on navigation). `document.elementFromPoint` at the button's own center confirmed the button really was the top element with `pointer-events: auto`, `display: block`, `visibility: visible` — there was nothing to click *through*. This was not caused by the toolbar's new docked-panel positioning (`#overlay-root`, `.stage-dom-overlay { pointer-events: none }` with children opting back in via `pointer-events: auto`) — that layering is correct and was independently verified via the same `elementFromPoint` check. The likely cause is this host's tight memory (a 3.7GB box already running Postgres + two Docker containers + a Go backend + this session's own harness, with swap frequently near its 2GB ceiling — `free -h` showed 965Mi free / 2.0Gi/2.0Gi swap during one hang), which can stall Chromium's CDP input-dispatch pipeline in a way that never actually times out cleanly.
- **Fix: skip the locator entirely.** Read the target element's `getBoundingClientRect()` via `page.evaluate`, then drive `page.mouse.move(cx, cy)` → `page.mouse.down()` → `page.mouse.up()` directly. This never hung once, across dozens of subsequent clicks, in the same environment. Also set `page.setDefaultTimeout(10000)` (or similar) globally so any *other* Playwright action that does hang fails fast with a clear `TimeoutError` instead of stalling the whole run — several early debugging cycles were wasted because a hang had no timeout and looked identical to "still doing legitimate work."

## Kernel 91 — Campus Tours: three real bugs a spotlight/tour overlay will reproduce in any project if built the naive way

### A full-viewport "dimming" overlay div will intercept every click itself, no matter what click-gating logic runs after it

- `tour-engine.js`'s first version used one `<div class="tour-scrim">` at `position: fixed; inset: 0` to dim the background, with click-gating logic deciding whether a given click "counted" as hitting the spotlighted target. It never worked: the browser resolves `event.target` by physical hit-testing, and the scrim was physically on top of the whole page — every click's `event.target` was the scrim (or a decorative child of it), never the real element visually spotlighted underneath. Reported by Grant plainly: "I can't click anything on the map." **Lesson: if an overlay needs to let clicks through to one real element beneath it, build the hole structurally — a real gap in the DOM's hit-testable geometry (e.g. four rectangles framing the target instead of one full-screen div) — never a single intercepting element plus click-handler logic trying to guess whether a click "really" landed on the thing underneath it. The guess is unnecessary work standing in for a five-minute layout fix.**

### A `MutationObserver` that watches the same subtree it writes styles into will trigger itself in an infinite loop

- The same engine's position-tracking used `new MutationObserver(() => positionOverlay())` observing `document.body` with `{ childList: true, subtree: true, attributes: true }` — and `positionOverlay()` itself sets `style.top/left/width/height` on elements living inside that exact subtree (the overlay is appended to `body`). Every reposition is itself an attribute mutation, which re-fires the observer, which repositions again — an unbounded loop pinning a CPU core, invisible in code review because nothing throws or crashes, only visible as degraded performance. It surfaced as a uBlock Origin "this page is slowing down your browser" warning, which is exactly the signature that heuristic exists to catch. **Lesson: before wiring up a `MutationObserver` (or any observer) with `attributes: true` on a subtree, check whether your own code writes styles/attributes into that same subtree in response to the observer firing. If so, either exclude `attributes` from what's watched, filter out mutations whose target is inside your own managed elements, or both — otherwise the observer is watching itself.**

### A click that also triggers real navigation will race an ordinary `fetch()` and can silently lose the write

- After fixing the two bugs above, Grant reported the tour "stuck in a loop of resetting to Audition Hall." The actual cause: a click-gated tutorial step's target was a venue pin whose own click handler does `window.location.href = ...` — a real page navigation, starting the instant the click fires. The tour's own step-completion call was an ordinary `fetch()`, and navigation can (and did) abort it before the response landed, so nothing was ever recorded server-side; every return to the map found no completion row and restarted the tour from scratch. This is not specific to tour UI — **any click handler that both persists state via a network call and can also trigger navigation (a link, a form submit, an assigned `location.href`) is exposed to this race**, and it will not show up in testing unless the test specifically waits to see whether the write survived the navigation, not just whether the call was issued. **Lesson: when a click both writes to the server and can trigger navigation in the same gesture, use `navigator.sendBeacon` (or `fetch(..., { keepalive: true })` as a fallback for beacon-unsupporting browsers) instead of a plain `fetch()` — `sendBeacon` is specifically designed by the browser to survive the page unloading immediately after the call returns. If the write also needs to represent partial progress (not just a final terminal state), consider a separate resumable-cursor row rather than overloading the terminal-completion record, so a resume-from-here read doesn't have to distinguish "finished" from "got partway and the page changed out from under it."**

### A failed browser-proof assertion is a bug report, but the bug might be in the test's own setup, not the feature

- A test script that picks a draw-start point by a fixed `canvasBox.width * 0.12, canvasBox.height * 0.15`-style offset (a reasonable-looking "near a corner, off-center" choice) will silently land on whatever floating chrome happens to be docked there instead of the canvas — the click/drag then does nothing, with no error, because the pointerdown never reaches the stage's own handler. Two real chips were hit during Kernel 87 verification: the Discord voice **`#discord-mic-status`** chip (top-left) and a **`.presence-preview`** badge (also near a corner, but its exact position isn't fixed — it appeared in a different quadrant across runs depending on who else was "present"). Both are legitimate, permanent, correctly-implemented UI — this is not a bug in either of them, and nothing about Kernel 87's own toolbar-placement fix caused it (the toolbar itself, docked top-right, was accounted for and avoided from the start).
- **Fix: probe, don't guess.** Before starting a drag/click sequence meant to hit the canvas, call `document.elementFromPoint(x, y)` at the intended coordinates first and confirm `tagName === "CANVAS"`; if not, try the next candidate point from a small list spread across different quadrants. This is a few lines of code and makes the test immune to whatever chips happen to be mounted in a given session, rather than requiring every future test author to enumerate and dodge the current set of floating chrome by hand. **Lesson for future Playwright work in this repo: never trust a fixed percentage-of-canvas coordinate for a "should definitely be empty space" click target — probe the actual DOM at that point first, because this stage has accumulated (and will keep accumulating) legitimately-positioned floating UI that a static coordinate has no way to know about.**

## Kernel 92 — Showtime Composition & Showing Scheduler

### A UI mounting point's own visibility precondition can make it structurally impossible to host the feature you're about to add — check it before wiring anything up

- The kernel doc's own wording pointed at the in-stage Kernel 89 Director toolbar ("top-right tool stack") as the obvious home for a new Showtime button. That toolbar's visibility gate is `id && canManage`, where `id` comes from `window.VictoryStageKernel88Bridge.getShowID()`, defined in `runtime.js` as `() => currentSnapshot?.session?.show_id || ""` — it requires an *already-live* session snapshot. Wiring the "begin the show" button into a surface that only appears once a show has already begun some other way would have shipped something that looked correct in a code diff and was unusable in practice, only discoverable by actually trying to open the popup before any session existed. **Lesson: before adding a control to an existing UI surface, read that surface's own visibility/mount condition, not just its layout — a container that is itself gated on the very state your new feature is meant to create cannot be where that feature's entry point lives, no matter how good a fit the spec's prose makes it sound.**

### `net/http`'s `ServeMux` panics at boot on a route conflict — a URL namespace can be "already taken" by an unrelated older feature with a coincidentally similar name

- The natural REST path for a new "Showing" scheduling endpoint was `/api/showings`. That path was already registered — by the pre-existing, unrelated Kernel 22 `showings` package (a live audience-visibility review surface, a different concept entirely that happens to share the English word). Go's standard `http.ServeMux.HandleFunc` panics immediately at registration time on an exact pattern conflict, which means this class of bug cannot silently ship — it fails loudly the moment the binary starts — but it also means it is caught at deploy time, not at code-review time, unless the existing route table is grepped first. **Lesson: before choosing a new REST path, `grep` the existing route registration (usually one central `main.go` mux-wiring block) for the exact string, not just for the feature name in prose — two unrelated features can independently want the same English noun as their URL segment, and the mux enforces uniqueness the hard way (a boot-time panic) rather than a compile-time or review-time signal.**

### A `producer`/`director` location role grants manage-authority over *every* Show Run at that location, not just ones a given actor or test created — a shared long-lived test database will silently leak other kernels' fixture rows into a new test's result set

- `showruns.ListShowRunsVisibleToUser`'s non-operator branch grants `CanManage` to anyone with an active `producer`/`director` `location_memberships` row at a Show Run's location — correct product behavior (a location's producers/directors jointly manage everything at that location), but it means a new test's disposable fixture user, granted `producer` at the shared `amurray-family` location the same way every other kernel's test fixtures have for over a year, sees *every* Show/Show-Run ever created there and never cleaned up, not just its own. The first version of `showing_list_test.go`'s alphabetical-sort assertion failed with dozens of interleaved leftover Shows from unrelated prior kernels' test runs (`K87v2 Show`, `K88 Fixture Show...`, `K90 Visibility Show...`) mixed into what was expected to be a clean 3-item list. **Lesson: a test asserting the exact contents or order of a listing endpoint scoped by role/authority (not by a hard foreign key to something the test itself created) must filter its own assertions down to IDs it created (e.g. by `ShowRunID`), never assume the result set is empty-but-for-what-this-test-inserted — a shared, long-lived, never-fully-reset test database accumulates other tests' real, valid, correctly-authorized fixture rows indefinitely, and role-based authority is often broader than "just what I made."**

### A same-day scheduling offset for a test fixture is wall-clock-dependent in a way an offset of 48h+ is not

- The same test originally scheduled its three fixtures at `now + 10h / +20h / +30h`, expecting all three to land in the same "Upcoming" bucket. Depending on what time of day the suite happens to run, `+10h`/`+20h` can fall on the same UTC calendar day as `now` (landing in "Today" per this kernel's bucketing rule) while `+30h` does not (landing in "Upcoming") — a flake that only reproduces at certain times of day and passes cleanly the rest of the time, which makes it easy to mistake for already-fixed once a re-run happens to pass. **Lesson: when a test fixture's scheduled offset needs to reliably land in a specific day-boundary-relative bucket ("Upcoming," "this week," "next month"), use an offset comfortably past the boundary in the worst case (here: 48h+, since even at 23:59 the boundary is at most ~24h away) rather than an offset that "usually" clears it — the failure mode is silent and time-of-day-dependent, not a hard failure every run.**

### `docker exec <container> <cmd>` does not forward a heredoc's stdin unless `-i` is passed — the command exits successfully with zero output and zero effect, not an error

- Cleaning up this kernel's browser-proof fixture rows via `docker exec victory-postgres psql -U victory -d victory <<'SQL' ... SQL` produced no output at all (not even psql's usual `DELETE n` per statement) and left every row untouched — no error was printed, the exit code was 0, and it looked exactly like "ran, deleted nothing because nothing matched." The actual cause: `docker exec` without `-i` does not attach the container process's stdin to the calling shell at all, so the heredoc's SQL never reached `psql`'s stdin — `psql` was invoked with no input and simply exited. Adding `-i` (`docker exec -i victory-postgres psql ...`) fixed it immediately, with the expected `DELETE n` line per statement appearing. **Lesson: a `docker exec ... <<'HEREDOC'` construct that runs "successfully" with suspiciously empty output is worth checking for a missing `-i` before assuming the SQL itself was wrong or that the WHERE clause matched nothing — `docker exec` is not `docker exec -i` by default, and the failure mode is silence, not an error.**

## Kernel 94 — Storyboards Vue Rebuild: two Vue/CSS traps that will recur in Kernel 95's much larger Vue surface

### A bare, call-shaped global reference in a Vue template (`@click="window.print()"`) can silently resolve to `undefined`, while the same global used in an assignment on the very next line works fine

- `Toolbar.js`'s new "Export PDF" button used `@click="window.print()"` and threw `Cannot read properties of undefined (reading 'print')` on every real click — caught only because the button was actually clicked in a Playwright test, not by code review or a visual screenshot (the button rendered fine; it just didn't work). The adjacent, pre-existing "Export JSON" button's `@click="window.location.href = store.apiBase + '/export'"` — same `window` global, same component, same template — worked without incident. Vue's runtime template compiler treats a bare identifier differently depending on whether it's the callee of a call expression or the target of an assignment when deciding whether to leave it as a true global reference (relying on the compiled template's `with(_ctx)` scope to fall through) or rewrite it as a property read on the render context; a call-shaped reference to a global not in Vue's small hardcoded allowlist (`Math`, `Date`, `JSON`, `console`, etc. — `window` is not on it) can get treated as if it should resolve on the component instance, which it obviously does not. **Lesson: never call a raw browser global directly as a call expression inside a Vue template string (`@click="window.foo()"`, `@click="document.bar()"`) — wrap it in a real method in `setup()` and bind the method by reference instead (`@click="exportPdf"`). This is now the standing convention for every Storyboards component and should be the default the moment Kernel 95 starts writing its own inline handlers.**

### Wrapping an existing flex item in a new intermediate `<div>` silently orphans any CSS rule that targeted it by descendant selector for `flex`/`min-width` — and the breakage only shows up once real content is wide/tall enough to need it

- Pass 2 wrapped `.board-scroll` (a flex item of `#board-root`, with its own `#board-root .board-scroll { flex: 1 1 auto; min-width: 0; }` rule letting it shrink to fit and scroll internally) inside a new `.board-column` div, to add an empty-state invite line above it. `.board-scroll` is no longer a *direct* child of `#board-root` — it's `.board-column`'s child now — so that rule, still selecting `.board-scroll` by descendant combinator, kept matching the element but had zero effect: `flex`/`min-width` only do anything on an element that is itself a flex item of the flex container the property is meant to constrain it within. `.board-column` (the real flex item now) had no `min-width: 0` of its own, so it refused to shrink below its content's full intrinsic width and the whole board overflowed the page instead of scrolling internally — invisible until Pass 5 seeded a board with enough columns to actually exceed the viewport; every earlier 3–4-column test board was narrow enough to never trigger it. **Lesson: whenever an existing flex item gets wrapped in a new intermediate element (for *any* reason — an empty-state line, a badge, a loading spinner), re-audit every CSS rule that named the original element specifically for `flex`/`min-width`/`max-width`/`flex-basis` and move it to whichever element is now the actual direct flex child — a descendant selector will keep matching and silently do nothing, with no warning, and small/sparse test content will not surface it. This is exactly the shape of bug Kernel 95's shared shell/tray rebuild is likely to reproduce, since it explicitly restructures existing chrome around new wrapper elements at a much larger scale.**
