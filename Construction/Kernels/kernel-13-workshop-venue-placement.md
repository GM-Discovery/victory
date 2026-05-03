# Kernel 13 Workshop to Venue Placement v1

## What is enforced

- Elements are created in workshop mode first.
- `act/place_element` records placement server-side.
- Venue trays/backstage are distinct from stage/worldspace.
- Reveal/hide still uses the existing visibility spine.
- The client can request placement, but the server decides.

## Venue enablement

- Venue placement is enabled by `venues.config.index_cards_enabled`.
- The Cave is enabled for index card placement.
- Workshop mode is the source surface for creation and sending.

## Action flow

1. Create or edit an index card in workshop mode.
2. Select an enabled target venue.
3. Send the card to the venue tray with `act/place_element` and `layer = tray`.
4. Place the card on stage with `act/place_element` and `layer = stage`.
5. Reveal the card with `act/reveal_element`.

## Backend surfaces

- Authority: `backend/internal/actions/authority.go`
- Placement storage: `backend/internal/actions/place.go`
- Index cards: `backend/internal/actions/indexcard.go`
- WebSocket dispatch: `backend/internal/network/ws.go`
- Enabled venue lookup: `backend/internal/access/visibility.go`
- API route: `backend/cmd/victory/main.go`
- Migration: `database/migrations/010_kernel13_workshop_placement.sql`

## Frontend surfaces

- Workshop entry: `frontend/venues/workshop/index.html`
- Cave workshop mode / venue tray / stage: `frontend/venues/the-cave/index.html`
- Map routing: `frontend/app.js`

## Operator notes

- Backend restart is required after code changes.
- `GET /api/workshop/venues` returns enabled target venues for the logged-in user.
- Workshop mode is opened from the map via the Workshop pin.
- Audience cannot create or place cards.
- Producer and director can create, send, and place cards.

