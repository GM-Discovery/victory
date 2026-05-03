# Kernel 8 Identity Surface + Presence Repair

## What is enforced

- Cave presence now uses a shared server-resolved identity surface.
- Logged-in Cave participants appear in the roster with non-blank labels.
- Speech, reaction, and replayed snapshot actions share the same identity shape.
- `persona` is present in the action and presence contract and is currently `null`.
- The Cave does not allow anonymous presence.

## Identity surface

Server-resolved identity now includes:

- `user_id`
- `handle`
- `display_name`
- `role`
- `session_id`
- `session_participant_id`
- `persona`

The current fallback order for display labels is:

1. `display_name`
2. `handle`
3. shortened `user_id`
4. `Unknown Participant`

## Presence model

- Presence lives in `backend/internal/network/presence.go`.
- Presence entries are keyed by session ID and user ID.
- Multiple tabs for the same user are deduped in the visible roster.
- A user only disappears from the roster after their last connection for that session closes.
- The registry is in-memory and intentionally non-canonical.

## Messages

The Cave WebSocket now sends:

- `snapshot` for the replayable world state
- `presence/snapshot` with `users`
- `presence/join` with `user`
- `presence/leave` with `user`
- `action` for stored speech/reaction/reveal/hide events

Every action and presence payload now carries:

- `actor` or `user`
- `persona: null`

## Where to look in code

- Identity resolver: `backend/internal/identity/session_identity.go`
- WebSocket identity + presence flow: `backend/internal/network/ws.go`
- Presence registry: `backend/internal/network/presence.go`
- Action actor attribution: `backend/internal/actions/react.go`
- Reveal attribution: `backend/internal/actions/reveal.go`
- Cave replay attribution: `backend/internal/world/snapshot.go`
- Cave UI roster + action log: `frontend/venues/the-cave/index.html`

## Operator toggles / testing

There is no DB toggle for presence itself.

To verify:

- run `go test ./internal/network -run TestKernel8IdentitySurfaceAndPersonaNull -v`
- open two sessions in The Cave
- confirm join shows in the roster
- close one tab and confirm the roster updates
- send speech and reactions and confirm the actor name + role badge render

## Rebuild notes

- Backend rebuild/restart is required when identity or presence code changes ship.
- No new environment variables were added.
- No new dependencies were added.

## Future seam

The current seam is:

- authenticated user
- active session participant
- in-memory presence registry
- action attribution from server-side identity
- `persona: null` placeholder for future production characters

Future kernels can attach characters and richer projection without changing the client trust model.
