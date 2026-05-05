# Kernel 21 Venue Chat v1

## What is enforced

- Venue chat is a separate `chat/message` lane from `perform/speak`.
- The Cave is the first venue with chat enabled.
- Chat is session-scoped and persists through the action log.
- Chat is visible to everyone in the venue.
- Chat input is capped at 250 characters.
- The chat panel is bottom-aligned and can collapse after inactivity.

## Backend surfaces

- Chat action storage: `backend/internal/actions/chat.go`
- WebSocket dispatch: `backend/internal/network/ws.go`
- Venue chat policy and permissions: `backend/internal/actions/authority.go`

## Frontend surfaces

- Venue chat panel: `frontend/venues/the-cave/index.html`

## Venue config

- `chat_enabled`
- `talking_enabled`

## Notes

- `perform/speak` stays separate and continues to render as stage speech.
- Chat does not appear above the fire.
- Chat history is rebuilt from the session action log after refresh/rejoin.
