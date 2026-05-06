# Operator Notes — VICTORY Foundation

## Purpose of this file
This file exists to keep future builders from re-arguing settled concepts, repeating solved mistakes, or building against the wrong model.

Read this before making structural changes.

---

## Current Status
Kernel 1.2 (`the-cave`) is **PARTIAL**, but the foundation is real and working.

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
- Kernel 7 evidence is covered by `go test ./internal/network -run TestKernel7PresenceAndAttributionEvidence -v`
- Kernel 8 adds a shared identity surface and `persona: null` in action/presence payloads
- The Cave does not allow anonymous presence
- Presence labels must never render blank; fall back through display name, handle, shortened user id, then `Unknown Participant`
- `elements.context_class` is now persisted in Postgres so props, scenery, cards, and future classes do not depend only on UI inference
- `prop` means mobile stage object; `scenery` means fixed set piece / anchor; `first-fire` is treated as fixed scenery for now
- Kernel 21 venue chat is a separate `chat/message` lane and must not be conflated with `perform/speak`
- The Cave chat panel is venue-scoped, session-backed, and collapses after inactivity

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
