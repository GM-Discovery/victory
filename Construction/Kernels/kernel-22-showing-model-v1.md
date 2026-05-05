# Kernel 22 Showing Model v1

## What is enforced

- WebSocket presence is still ephemeral and tracks who is connected now.
- Showing is the durable theatrical record for what is happening in The Cave.
- A showing survives reconnects, browser crashes, and WebSocket drops.
- Chat, speech, reactions, reveal/hide, overlay actions, and card placement now attach to a showing.
- Closing a showing blocks new chat and stage actions.

## Showing model

- Showing fields:
  - `id`
  - `production_id`
  - `venue_id`
  - `run_id` nullable
  - `status` (`rehearsal`, `live`, `closed`)
  - `audience_view_enabled`
  - `started_at`
  - `ended_at` nullable
  - `created_by`
- The Cave currently uses a single active showing attached to the live session.
- Rejoining or reconnecting does not create a new showing if one already exists.

## Data attachment

- `actions.showing_id` is populated for Cave actions.
- Existing Cave actions were backfilled to the active showing.
- `chat/message` stays separate from `perform/speak`.
- Venue chat now lives in the bottom panel, not above the fire.

## Access / lifecycle

- `rehearsal`
  - audience view off
  - tools usable
- `live`
  - audience view on
  - performance active
- `closed`
  - no new chat/actions
  - review only

## Backend surfaces

- showing bootstrap and helpers: `backend/internal/showings/showings.go`
- snapshot projection: `backend/internal/world/snapshot.go`
- action writers: `backend/internal/actions/*.go`
- session join bootstraps the active showing: `backend/internal/identity/join.go`

## Frontend surfaces

- Cave still renders the stage, cards, and venue chat in separate lanes:
  - `frontend/venues/the-cave/index.html`

## Operator notes

- No new environment variables were added.
- No new dependencies were added.
- Backend restart is required after code changes.
- If you close the showing manually in SQL, new Cave actions should be rejected with `showing_closed`.
- The current live Cave session already has a showing row and its historic actions have been backfilled.
