# Victory Current State

## Purpose
This document is the current-state canon for Victory as of Kernel 30.

Older notes in `Construction/OperatorLogs/` and older kernel docs remain useful as history, but this file is the current source of truth when they disagree.

## Kernel State
- Current kernel label: **Kernel 30**
- Current kernel purpose: **Cave Organization + Renderer Containment**
- Product state: **kernel active; Kernel 30 is in progress**
- Important note: kernel numbers are labels, but from this point forward they should stay stable once assigned.

## Current Stack
- Frontend: static HTML, CSS, and inline JavaScript under `/opt/victory/frontend`
- Backend: Go `1.25` in `/opt/victory/backend`
- Reverse proxy/static serving: Caddy via `/opt/victory/Caddyfile`
- Database: PostgreSQL `16-alpine`
- Container orchestration: Docker Compose
- Key Go dependencies:
  - `github.com/gorilla/websocket`
  - `github.com/jackc/pgx/v5`
  - `golang.org/x/crypto`
  - `golang.org/x/image`

## Runtime Modes
- Dev mode:
  - Postgres in Docker
  - backend run on host with `go run ./cmd/victory`
  - common ports: `8081` and sometimes `18081`
- Install mode:
  - backend in Docker as `victory-backend`
  - Postgres in Docker as `victory-postgres`
  - Caddy proxies `/api/*` and `/ws/*` to backend `8081`

## Current Venue List
Live venue rows currently present:
- `the-cave` - presentation venue
- `greenroom` - profile/character venue
- `trailers` - profile drafting venue
- `workshop` - workshop venue
- `info-booth` - public info venue
- `producers-office` - office venue
- `directors-chair` - plaza venue
- `grants-cabin` - cabin venue
- `catharsis` - plaza venue
- `construction` - public construction venue
- `victory-theater` - venue
- `audition-hall` - audition venue
- `library` - public reference venue
- `warehouse` - restricted storage venue
- `soil-experts` - public placeholder venue

## Current Route / API Surface
Auth and identity:
- `POST /api/auth/signup`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `POST /api/auth/password-reset/request`
- `POST /api/auth/password-reset/confirm`
- `GET /api/session/me`

Invites, requests, and production access:
- `POST /api/invites`
- `POST /api/invites/accept`
- `POST /api/requests/create`
- `GET /api/requests/mine`
- `GET /api/requests/incoming`
- `POST /api/requests/respond`
- `GET /api/productions`
- `GET /api/map/visibility`

Profiles:
- `GET /api/profiles/me`
- `GET /api/profiles/public`
- `PATCH /api/profiles/me/save`
- `POST /api/profiles/me/publish`
- `POST /api/profiles/admin/save`
- `POST /api/profiles/admin/publish`

Character cards:
- `GET /api/character-cards/me`
- `POST /api/character-cards`
- `PATCH /api/character-cards/{id}`
- Legacy/dormant authority routes still exist, but current product behavior does not rely on them:
  - `POST /api/character-card-permissions`
  - `POST /api/character-card-permissions/revoke`

Mailbox and note cards:
- `GET /api/messages`
- `GET /api/messages/{id}`
- `POST /api/messages`
- `POST /api/note-cards`

World, session, and venue runtime:
- `GET /api/world/the-cave`
- `POST /api/session/the-cave/join`
- `GET /api/workshop/venues`
- `POST /api/index-cards`
- `POST /api/workshop/assets`
- `GET /api/assets/{id}`
- `GET /api/showings`
- `GET /api/showings/{id}/review`
- `GET /api/director-console/current`
- `POST /api/showings/{id}/audience-view`
- `POST /api/showings/{id}/close`
- `POST /api/showings/start`
- `POST /api/venues/{slug}/chat-policy`
- `GET /health`

WebSocket:
- `GET /ws/the-cave`

## Current WebSocket Actions And Events
Client-to-server actions currently handled in The Cave:
- `ping`
- `react/emote`
- `chat/message`
- `perform/speak`
- `act/reveal_element`
- `act/hide_element`
- `act/show_overlay`
- `act/hide_overlay`
- `act/place_element`
- `create/index_card`
- `update/index_card`
- `delete/index_card`
- `persona/equip`
- `persona/unequip`

Server-to-client event shapes currently emitted:
- `snapshot`
- `action`
- `error`
- `pong`
- `presence/snapshot`
- `presence/join`
- `presence/leave`
- `presence/update`
- `showing/update`
- `venue/update`

## Current Database / Domain Concepts
Identity and access:
- `users`
- `auth.sessions`
- `auth.password_credentials`
- `auth.password_reset_tokens`
- `invites`
- `memberships`
- `location_memberships`

World model:
- `locations`
- `lots`
- `venues`
- `libraries`
- `elements`
- `placements`
- `sessions`
- session participants and runtime identity links

Runtime history and review backbone:
- `actions`
- `showings`
- `messages`

Profile and character surfaces:
- `performer_profiles`
- `character_cards`
- `current_session_personas`
- `permission_grants` still exists in schema, but current Greenroom drafting no longer depends on it

Important language:
- **auth session** means the cookie-backed authenticated account session
- **connection/presence** means live WebSocket state, not durable history
- **showing** means the reviewable run-state record attached to a session
- **production run** is not the same thing as video capture
- **production** means production-scoped world/authority context
- **venue** means a surfaced experience and rules context

## Current Live-Table Behavior
The Cave is the full-feature proving-ground venue.

Current visible behavior in The Cave:
- signed-in, access-checked session join
- live snapshot load
- presence roster
- audience-view curtain when the director turns audience view off
- stage speech
- venue chat
- reactions
- index card editing and placement
- reveal/hide and overlay actions
- persona equip/unequip from existing character cards
- history replay through snapshot + action stream

Current Pixi proving-ground behavior:
- First Theater is the PixiJS stage spike venue
- PixiJS is experimental unless proven otherwise; it is renderer-only, not app authority
- live Cave snapshot and session actions can be used to judge renderer coexistence
- DOM overlays remain separate from the Pixi canvas so controls stay outside the stage layer
- Kernel 29 is complete as a proving-ground spike
- Kernel 30 is now organizing the proving-ground UI and cleaning up the remaining affordances

Strategy:
- tools may be built visibly in The Cave first
- once stable, they should be hidden into overlays, drawers, context menus, or cleaner surfaces
- a clean template venue will later be extracted from The Cave
- future venues should descend from that cleaned template rather than re-inventing runtime behavior separately

## Current Greenroom / Trailers Split
- Greenroom:
  - public profile display
  - character dressing room
  - character creation and editing
  - sheet links as metadata references on character cards
- Trailers:
  - performer profile drafting/publishing
  - editor hidden until revealed
  - public-facing de-anonymization/profile surface

## Current Known Working Flows
- Sign in and resolve a session-backed account identity
- Load map visibility
- Open The Cave, load snapshot, join the active session, receive presence
- Send stage speech with server-resolved actor attribution
- Send venue chat with server-side storage and authority checks
- Send reactions and see them broadcast and persisted
- Create and update index cards
- Place workshop cards into enabled venues
- Create mailbox messages and note cards
- Draft and publish performer profile fields in Trailers
- Draft and edit character cards in Greenroom
- Attach `sheet_links` metadata to character cards
- Equip and unequip an existing character persona in The Cave
- Open the Director's Chair and control the current showing with live audience-view, chat-policy, presence, and overlay controls
- Review closed showings in the Director's Chair with readable event cards

## Current Known Gaps
- Video recording does not exist and is not a near-term priority
- Cave UI still exposes proving-ground tool density and needs later organization
- Template venue extraction has not happened yet
- Showing Review currently covers closed showings only
- Kernel 30.2 is a consolidation pass for scraps, drift, and unfinished work; it should document more than it invents
- Character sheets are links/references only, not playable sheet records
- No rules-engine execution, dice, stats, HP, initiative, grid, tokens, or fog
- Browser-level character-sheet save confusion still needs direct front-end reproduction even though live authenticated create and PATCH both succeed against the backend
- Starting a brand-new showing is still deferred; the live console can close a showing and control the current one, but it does not yet create a fresh run on demand
- Some older docs still describe earlier kernel truths and are now historical

## Recording Language
- **Showing Review** means review of actions, chat, reactions, notes, and showing/session history
- **Video Recording** means future capture of rendered audiovisual output

Victory is currently pursuing **Director Console / Showing Review / proving-ground hardening**, not near-term video recording.
