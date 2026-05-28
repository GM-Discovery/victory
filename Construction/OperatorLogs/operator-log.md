# Operator Log — VICTORY

This file records meaningful implementation milestones and notable operational decisions.
Keep entries factual.

Current status and roadmap now live in:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [roadmap.md](/opt/victory/Construction/roadmap.md)

This file should stay historical and chronological.

---

## 2026-05-20 — Kernel 33 Operator Bootstrap + Canon Capture

### Backend
- Added an operator bootstrap CLI:
  - `go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>`
  - `go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>`
  - `go run ./cmd/victory-bootstrap producer --handle <victory_handle>`
- The bootstrap path grants only `producer` at the location scope already used by Victory access checks
- The command is idempotent and reports whether the producer grant already existed
- Unknown Discord user IDs fail clearly instead of creating a new grant

### Canon / Documentation
- Captured Kernel 32 as complete in the current canon
- Captured the live-site follow-on work:
  - Terms page
  - Privacy page
  - favicon asset
  - favicon wiring
  - `/auth/*` proxy routing
  - Discord env wiring
  - local `.env` guidance
- Updated roadmap/current-state/operator notes to reflect Kernel 33

### Notes
- Discord login remains identity only
- Victory still authorizes producer authority
- Operator/bootstrap authority stays separate from normal app authority

## 2026-05-19 — Kernel 32 Discord OAuth Primary Login

### Backend
- Added Discord OAuth start/callback routes:
  - `GET /auth/discord/start`
  - `GET /auth/discord/callback`
  - API aliases at `GET /api/auth/discord/start` and `GET /api/auth/discord/callback`
- Added provider discovery route:
  - `GET /api/auth/providers`
- Added Discord identity linking and OAuth state storage helpers
- Preserved the existing Victory session cookie model for post-OAuth login

### Database
- Added `auth.discord_identities`
- Added `auth.oauth_states`

### Frontend
- Added a Discord login button on `/login/`
- Kept local handle/password login intact

### Notes
- Discord OAuth authenticates identity only
- Victory still authorizes users through its own sessions, memberships, and roles
- No Discord bot, guild, channel, or voice integration was added

## 2026-03-25 — Foundation stood up

### Infrastructure
- Confirmed Docker already installed and active on `murray-vserver`
- Created `/opt/victory`
- Started dedicated Postgres container:
  - `victory-postgres`
- Chose Docker as deployment/install truth
- Chose host-Go + Docker-Postgres hybrid mode for faster development iteration

### Schema / Database
- Applied initial schema migration
- Established tables for:
  - users
  - locations
  - memberships
  - lots
  - venues
  - libraries
  - elements
  - placements
  - sessions
  - participants
  - actions
- Added `handle` to users
- Seeded one active live session for `the-cave`

### World Seeding
- Seeded location `amurray.family`
- Seeded lot `main-lot`
- Seeded venue `the-cave`
- Seeded library `house-library`
- Seeded reusable element `first-fire`
- Seeded default placement of `first-fire` into `the-cave`

### Backend
- Created first Go backend
- Added DB connection layer
- Added `/health`
- Added `/api/world/the-cave`
- Fixed timestamp scanning bug in snapshot loader
- Fixed empty actions serialization (`[]` instead of `null`)
- Added `POST /api/session/the-cave/join`
- Added websocket endpoint `/ws/the-cave`

### Identity
- Established current v1 identity model:
  - stable user id
  - handle
  - display name
- Verified join behavior:
  - create/update user by handle
  - join active session
  - default role currently audience

### WebSocket / Observation
- Verified websocket connection to `/ws/the-cave`
- Verified initial snapshot delivery
- Verified ping/pong behavior

### Reaction Loop
- Implemented `react/emote`
- Verified reaction validation
- Verified server-assigned `moment_id`
- Verified persistence to `actions`
- Verified broadcast to connected websocket clients
- Verified late joiner sees prior reaction in snapshot
- Verified live reaction delivery to later-connected observer

### Current Status
Kernel 1.2 remains PARTIAL.

Reason:
- observation path works
- reaction path works
- speech path not yet implemented
- full actor→audience→reaction proof not yet complete

### Current Confirmed Working Stack
- world snapshot
- persistent join
- websocket observer stream
- recorded reaction broadcast

### Current Known Frictions
- Docker rebuilds are slow for active coding
- host-Go development requires correct localhost DB configuration
- install workflow and dev workflow are not yet formalized into scripts/docs

### Recommended Next Step
Implement `perform/speak` using the same action pipeline:
- validate joined actor
- assign `moment_id`
- persist action
- broadcast to observers

---

## 2026-05-01 — Presence + Attribution added

### Backend
- Added an in-memory Cave presence registry keyed by session and user
- Added explicit presence WebSocket event types:
  - `presence/snapshot`
  - `presence/join`
  - `presence/leave`
- Server now resolves actor identity for speech, reactions, and reveal/hide actions before broadcast
- Multiple tabs for the same user are deduped in the visible presence roster

### Frontend
- Added a plain "Who is here?" panel in The Cave
- Added visible speaker attribution and role badges to chat/history lines
- Added visible reaction attribution to chat/history lines
- Removed client-side actor ID trust from outgoing action payloads

### Operational Notes
- Presence is in-memory and intentionally non-canonical
- Backend restart is required for code changes to appear
- No new dependencies were added for this kernel

---

## 2026-05-01 — Identity surface + presence repair

### Backend
- Added a shared server-resolved identity surface in `backend/internal/identity/session_identity.go`
- Cave presence and action attribution now share the same identity resolver
- Added `persona: null` to presence and action payloads
- Display labels now fall back cleanly instead of rendering blank roster entries

### Frontend
- The Cave roster now renders the shared identity surface
- The Cave action log now renders identity labels from the same shape as presence
- The empty roster message is explicit: no one is currently in The Cave

### Operational Notes
- The Cave does not allow anonymous presence
- `persona` is intentionally `null` for Kernel 8
- Backend restart is required for identity/presence changes

---

## 2026-05-02 — Greenroom + Trailers profile surface

### Backend
- Added `performer_profiles` as the server-owned draft/publish profile table
- Added logged-in-only profile routes for:
  - `GET /api/profiles/me`
  - `GET /api/profiles/public`
  - `PATCH /api/profiles/me/save`
  - `POST /api/profiles/me/publish`
- Added producer-gated admin profile update/publish routes for the temporary override seam
- Added Greenroom and Trailers venue seeds
- Added map visibility for Greenroom and Trailers to signed-in users

### Frontend
- Added `frontend/venues/greenroom/index.html`
- Added `frontend/venues/trailers/index.html`
- Added shared identity display helpers in `frontend/lib/identity.js`
- Added map routing and venue icons for Greenroom and Trailers
- Added explicit public-data warnings and public field markers in Trailers
- Expanded the profile surface with expressive public fields for favorite fun, most relaxed, favorite color, favorite artist, favorite food, favorite song, favorite place, favorite movie or show, hidden talent, and ideal day

### Operational Notes
- The profile image upload is stored as a data URL in the profile table for now
- The profile surface does not store birthdates, SSNs, or credit-card-adjacent data
- Age is represented only as a public performance range string
- Greenroom and Trailers are signed-in venues, not anonymous venues
- Backend restart is required for the new routes and seeds to appear

## 2026-05-02 — Info Booth + Mailbox foundation

### Backend
- Added a durable `messages` table bootstrap and migration
- Added authenticated message routes:
  - `GET /api/messages`
  - `GET /api/messages/{id}`
  - `POST /api/messages`
- Added operator-only message creation for dev/test delivery

### Frontend
- Added a public Info Booth modal on the map
- Added a dedicated Mailbox page at `/mailbox/`
- Added inbox list + message detail rendering

### Operational Notes
- Info Booth is public and modal-only
- Mailbox is authenticated-only and server-owned
- Users can only read their own messages
- Message bodies are capped at about 250 characters
- Backend restart is required for the new routes to appear

## 2026-05-03 — Note Card delivery system

### Backend
- Added `POST /api/note-cards`
- Note cards now store as `message_type = note_card`
- Note cards carry Cave context:
  - `venue_slug`
  - `session_id`
- Recipient resolution prefers visible Cave participants and falls back to the director mailbox

### Frontend
- Added a Cave note-card sender form
- Added mailbox context rendering for note cards
- Added a Trailers link to open Mailbox directly

### Operational Notes
- Note cards are durable mail, not chat
- Body limit is 250 characters
- Sender identity is server-resolved from the Cave session
- Backend restart is required for the new route and schema bootstrap

## 2026-05-03 — Index Card Element v1

### Backend
- Added `create/index_card` and `update/index_card` action handling
- Index cards now materialize as `elements.element_type = 'index_card'`
- Index cards persist front/back/color plus server-owned creator and production/session metadata
- Directors and producers can create and edit cards; cast/crew/audience are filtered by the existing visibility spine

### Frontend
- Added an Index Cards tray/editor inside The Cave
- Added card selection, back-side preview, and save history rendering
- Added a visibility layer selector for sharing cards with cast/crew or audience

### Operational Notes
- Index cards are Elements, not a separate card universe
- Saves append action history and refresh the Cave snapshot
- Card content is capped at 2000 characters total
- Backend restart is required for the new action handlers to appear

## 2026-05-03 — Workshop to Venue Placement v1

### Backend
- Added `act/place_element` for index card placement
- Added `backend/internal/actions/place.go` for server-side placement persistence
- Added `GET /api/workshop/venues` for enabled target venues
- Venue enablement now keys off `venues.config.index_cards_enabled`

### Frontend
- Added Workshop mode to the Cave card surface
- Added Send to Venue dropdown and tray/stage placement buttons
- Added a Workshop redirect page from the map

### Operational Notes
- Workshop is the source surface for cards
- Tray/backstage and stage are distinct placement surfaces
- Backend restart is required for placement and venue-list changes

## 2026-05-06 — Kernel 24 and 25 reconciliation checkpoint

### Runtime / Character Notes
- Greenroom is the character dressing room
- Trailers is the performer profile drafting/publishing surface
- The Cave keeps live persona use rather than full character editing
- Character card drafting is currently available to performer roles without separate draft-grant workflow
- Character sheet links are metadata references on character cards, not playable sheet records

### Documentation Notes
- Added `Construction/current-state.md` as current canon
- Added `Construction/roadmap.md` as active roadmap
- Added placeholder kernel docs for 22–25 naming reconciliation
- Older notes are being marked historical/superseded instead of silently overwritten

## 2026-05-17 — Kernel 31 Middle School Stage shell

### Backend
- Seeded `middle-school-stage` as a producer-only venue surface
- Added producer visibility for `middle-school-stage`

### Frontend
- Added `frontend/venues/middle-school-stage/index.html`
- Added `middle-school-stage` routing in the map app
- Added a top status bar, left/right edge drawers, a collapsed bottom chat drawer, and a center-first stage shell layout
- Added a placeholder right-click context menu on a targetable stage object

### Operational Notes
- The Cave remains untouched and continues to serve as the proving ground
- Pixi remains outside The Cave
- Full live control parity is still Cave-specific; Middle School Stage is currently a shell-first proof

## 2026-05-17 — Kernel 32 template extraction

### Frontend
- Added a hidden `stage-template` venue scaffold as the reusable shell descendant
- Mounted the portable overlay from `frontend/venues/shared/venue-shell.js` in First Theater so Pixi and overlay can be compared together
- Refactored Middle School Stage to use the shared venue shell helper for preferences and presence preview behavior

### Documentation
- Updated the current-state canon and roadmap to treat Kernel 32 as the template extraction pass
- Renumbered the downstream roadmap so Discord OAuth becomes the next identity kernel

### Operational Notes
- The Cave was left untouched
- The hidden template scaffold is not map-visible
- First Theater now shows the portable overlay above Pixi as the comparison surface
