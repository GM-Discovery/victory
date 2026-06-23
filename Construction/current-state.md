# Victory Current State

## Purpose
This document is the current-state canon for Victory as of Kernel 50.

Older notes in `Construction/OperatorLogs/` and older kernel docs remain useful as history, but this file is the current source of truth when they disagree.

## Kernel State
- Current kernel label: **Kernel 50**
- Current kernel purpose: **Warehouse Token Placement + Grid Snapping v1**
- Product state: **Kernel 50 is complete**
- Important note: kernel numbers are labels, but from this point forward they should stay stable once assigned.

## Current Stack
- Frontend: static HTML, CSS, and inline JavaScript under `/opt/victory/frontend`
- Shared frontend shell helper: `frontend/venues/shared/venue-shell.js`
- Shared Pixi helper: `frontend/lib/victory-pixi-stage.js`
- Shared Pixi grid helper: `frontend/lib/victory-pixi-grid.js`
- Shared First Theater camera helper: `frontend/lib/victory-stage-camera.js`
- Backend: Go `1.25` in `/opt/victory/backend`
- Reverse proxy/static serving: Caddy. Repo config lives at `/opt/victory/Caddyfile`; the current live shared Caddy container mounts `/opt/bread-exchange/Caddyfile` and serves Victory from `/opt/victory/frontend`.
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
- `first-theater` - stage venue with a dedicated map layer
- `stage-template` - hidden internal shell scaffold, not map-visible
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
- `GET /api/auth/providers`
- `GET /auth/discord/start`
- `GET /auth/discord/callback`
- `GET /api/auth/discord/start`
- `GET /api/auth/discord/callback`
- `POST /api/auth/signup`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `POST /api/auth/password-reset/request`
- `POST /api/auth/password-reset/confirm`
- `GET /api/session/me`

Operator bootstrap:
- `go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>`
- `go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>`
- `go run ./cmd/victory-bootstrap producer --handle <victory_handle>`

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
- `GET /api/workshop/assets?asset_type=map`
- `POST /api/workshop/assets/token`
- `POST /api/index-cards`
- `POST /api/workshop/assets`
- `GET /api/assets/{id}`
- `GET /api/assets/{id}/content`
- `GET /api/warehouse/storage`
- `PATCH /api/warehouse/storage/settings`
- `GET /api/warehouse/assets`
- `GET /api/warehouse/assets/{id}`
- `DELETE /api/warehouse/assets/{id}`
- `GET /api/showings`
- `GET /api/showings/{id}/review`
- `GET /api/director-console/current`
- `POST /api/showings/{id}/audience-view`
- `POST /api/showings/{id}/close`
- `POST /api/showings/start`
- `POST /api/venues/{slug}/chat-policy`
- `GET /api/venues/{slug}/map`
- `POST /api/venues/{slug}/map`
- `DELETE /api/venues/{slug}/map`
- `GET /api/venues/{slug}/grid`
- `PUT /api/venues/{slug}/grid`
- `GET /api/discord/audio/status`
- `GET /health`

Discord audio remains a Discord pass-through surface. Victory does not capture or stream audio.
Voice-state participant presence is tracked from Discord Gateway events. Active speaker detection and per-user volume controls remain deferred because the current bot/Gateway path cannot truthfully provide them.
Venue shells now share reusable helper methods for slug normalization, slot registration, presence preview rendering, and safe refresh hooks.
First Theater keeps its original stage façade; the active map renders in the shared world layer and does not replace the façade or the existing DOM controls. The map can be removed via the map editor's Remove Map action, and supports a `display_mode` of `theater` (masked to the proscenium opening) or `fullscreen` (stretched to the full canvas, façade hidden).
A persistent square/hex grid renders in that same world layer, aligned with the active map and pinned cards, while unpinned cards stay in the fixed overlay layer above the interactive view. Grid configuration (type, hex orientation, cell size, offsets, opacity, line width, line style, visibility) is server-persisted per venue in `venue_grid_configs` and is visual-only — no snapping or measurement. Configured via a "Configure Grid" stage context-menu item and callout panel with live preview, Save, Close (reverts to last saved), Reset, and Hide/Show.
First Theater now has a personal browser camera over that shared map/grid world. Middle-mouse drag pans, wheel zoom centers toward the cursor, edge scrolling respects the playable stage rectangle, and the visible `− / 100% / + / Fit` control stays fixed in the safe interface region. Camera state is personal and browser-local, keyed by user/browser plus venue and active map, and `Fit` resets to 100%.
Index cards default to Pin to Screen. Cards may Attach to Map or Pin to Screen without jumping, Floating is a temporary drag state, map-attached cards move/scale with the world, screen-pinned cards keep a stable viewport size, overlay cards stay readable above the interactive view, and the card update path now carries optional pin metadata for `world_x`, `world_y`, `screen_x`, `screen_y`, and `pin_mode`. Move Here and Duplicate now respect that same placement mode so cards stay in the correct space without accidental extra copies.
Shift+Ping broadcasts a Director-and-above focus ping within the current venue, animates the recipient camera in about 250ms, and leaves the browser's personal camera persistent afterward.
The First Theater map editor now activates existing map assets directly from the asset list, and the grid renderer follows the rendered map bounds so larger maps stay fully covered when zoomed out.
Workshop token preparation now lives in The Cave's `mode=workshop` surface, with circle/square/hex/raw previews, token uploads, and installation-wide warehouse storage policy wired to the Producer's Office.
First Theater now consumes reusable Warehouse token assets through an `Add Token` picker, and placed tokens persist through refresh with grid-aware sizing and snap/free placement.
Warehouse asset reads still resolve deleted or missing assets to the construction fallback image instead of leaving placements blank.
Director focus/broadcast and touch controls remain deferred.

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

`update/index_card` and `act/duplicate_element` now also accept optional pin metadata for First Theater card attachment behavior.

Server-to-client event shapes currently emitted:
- `snapshot`
- `action`
- `error`
- `pong`
- `venue/focus_ping`
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
- `auth.discord_identities`
- `auth.oauth_states`
- `invites`
- `memberships`
- `location_memberships`

Current live-site support:
- Terms of Service page exists at `/legal/terms/`
- Privacy Policy page exists at `/legal/privacy/`
- Favicon asset exists at `frontend/assets/favicon.png`
- Favicon is wired into the main site, login/account/mailbox, legal pages, and venue shells
- Victory Theater map icon uses the same favicon asset
- `/auth/*` live proxy routing is fixed so Discord OAuth reaches Victory
- Discord environment wiring is present in Docker Compose and `.env`
- `.env` stays local and is ignored by git

World model:
- `locations`
- `lots`
- `venues`
- `libraries`
- `elements`
- `placements`
- `sessions`
- session participants and runtime identity links
- `venue_active_maps` - one active map placement per venue (asset, fit, crop, scale, safe margin, `display_mode`)
- `venue_grid_configs` - one grid configuration per venue (type, hex orientation, cell size, offsets, opacity, line width, line style, visibility)
- `warehouse_storage_settings` - installation storage policy for hard cap, upload cap, warning thresholds, retention default, and token variant sizes
- `assets` now also carries durable warehouse metadata such as `name`, `shape`, `default_grid_width`, `default_grid_height`, `retain_original`, `status`, `crop_x`, `crop_y`, `zoom`, `stored_bytes`, `last_used_at`, and `deleted_at`

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

Operator rule:
- Discord OAuth authenticates a person
- Victory authorizes the person
- operator/bootstrap authority can grant producer authority after identity exists
- producer remains the highest normal in-app authority

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
- the portable overlay panel is now mounted from the shared venue shell helper so First Theater can compare Pixi and overlay together
- the overlay proof marker is smoke-only; it is hidden in normal First Theater mode and only appears when explicitly running smoke tests
- Kernel 29 is complete as a proving-ground spike
- Kernel 30 organized the proving-ground UI and cleaned up the remaining affordances

Strategy:
- tools may be built visibly in The Cave first
- once stable, they should be hidden into overlays, drawers, context menus, or cleaner surfaces
- a clean template venue was extracted from the organized stage shell as `stage-template`
- future venues should descend from that cleaned template rather than re-inventing runtime behavior separately
- Middle School Stage remains the source shell and proves the edge-drawer grammar without dragging The Cave clutter along
- First Theater now shows the portable overlay over Pixi so renderer and overlay can be compared side by side

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
- Open Middle School Stage as a producer-only shell with top bar, edge drawers, and collapsed chat drawer
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
- Add/replace/remove the First Theater stage map, including `theater` and `fullscreen` display modes
- Configure, preview, align, save, hide, show, and reset a persistent square or hex grid over the First Theater map

## Current Known Gaps
- Video recording does not exist and is not a near-term priority
- Cave UI still exposes proving-ground tool density and needs later organization
- Template venue extraction has not happened yet
- Middle School Stage is shell-first, not a full theater product
- Showing Review currently covers closed showings only
- Kernel 30.2 was a consolidation pass for scraps, drift, and unfinished work; it documented more than it invented
- Character sheets are links/references only, not playable sheet records
- First Theater grid (Kernel 47) is visual-only; no rules-engine execution, dice, stats, HP, initiative, snap-to-grid, tokens, fog, or pan/zoom yet
- Collapsed header blur may visually overlap the top edge of the First Theater map (known deferred layout issue, not addressed in Kernel 47)
- Browser-level character-sheet save confusion still needs direct front-end reproduction even though live authenticated create and PATCH both succeed against the backend
- Starting a brand-new showing is still deferred; the live console can close a showing and control the current one, but it does not yet create a fresh run on demand
- Some older docs still describe earlier kernel truths and are now historical

## Recording Language
- **Showing Review** means review of actions, chat, reactions, notes, and showing/session history
- **Video Recording** means future capture of rendered audiovisual output

Victory is currently pursuing **Director Console / Showing Review / proving-ground hardening**, not near-term video recording.
