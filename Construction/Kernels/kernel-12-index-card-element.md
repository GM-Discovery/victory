# Kernel 12 Index Card Element v1

## What is enforced

- Index cards are Elements, not a separate card universe.
- Index cards live on the existing element/action/visibility spine.
- The Cave is the first venue wired for index cards in v1.
- Director and producer can create and edit index cards.
- Cast and crew can view revealed index cards.
- Audience does not see index cards by default.
- `act/reveal_element` and `act/hide_element` remain the visibility mechanism.

## Card model

Index cards are stored as `element_type = 'index_card'` and carry server-owned data:

- front_text
- back_text
- color
- created_by
- created_by_display_name
- created_by_handle
- created_by_role
- created_at
- updated_at
- production_id when available
- session_id
- venue_slug

## Rules

- Save-based append history is recorded through `create/index_card` and `update/index_card` actions.
- Card content is capped at 2000 characters total across front and back.
- Directors can reveal cards to cast/crew with the actor layer.
- Directors can reveal cards to audience with the audience layer.
- Hiding uses the existing visibility action path.

## Backend surfaces

- authority: `backend/internal/actions/authority.go`
- storage + projection: `backend/internal/actions/indexcard.go`
- websocket handling: `backend/internal/network/ws.go`
- snapshot projection: `backend/internal/world/snapshot.go`

## Frontend surfaces

- Cave tray/editor: `frontend/venues/the-cave/index.html`

## Operator notes

- No new environment variables were added.
- No new dependencies were added.
- Backend restart is required after code changes.
- Index card saves are appended as actions and materialized into the element row.
- The tray shows only cards visible to the current user.
- The reveal layer selector in The Cave controls whether the selected card is shared with cast/crew or audience.
