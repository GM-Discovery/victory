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
- `react/emote` works over WebSocket
- `react/emote` is persisted into `actions`
- `react/emote` is broadcast to connected observers
- later-joining observers see prior recorded actions in snapshot history

Not yet complete:
- `perform/speak`
- explicit actor-role path
- full two-user kernel proof: actor speaks, audience perceives, audience reacts, actor perceives reaction

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
5. later: note cards
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