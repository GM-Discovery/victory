# Kernel 7 Presence + Attribution Protocol

## What is enforced

- Presence is derived from authenticated WebSocket state, not client claims.
- The Cave now tracks who is currently connected in memory, keyed by session and user.
- Speech and reaction actions are enriched server-side with actor display name, handle, and role.
- The Cave client renders a live "Who is here?" panel and attributed action log.

## Presence model

- Presence lives in `backend/internal/network/presence.go`.
- Presence entries are keyed by session ID and user ID.
- A user only disappears from the roster after their last connection for that session closes.
- The registry is in-memory and intentionally non-canonical.

## Identity resolution

On WebSocket connect, the server resolves:

- `user_id`
- `handle`
- `display_name`
- `role`
- `session_participant_id`

This is done from the authenticated session cookie and the existing session membership tables.

## Messages

The Cave WebSocket now sends:

- `snapshot` for the replayable world state
- `presence/snapshot` with `users`
- `presence/join` with `user`
- `presence/leave` with `user`
- `action` for stored speech/reaction/reveal/hide events

Presence messages carry:

- `user`
- `users`

## Where to look in code

- WebSocket identity + presence flow: `backend/internal/network/ws.go`
- Presence registry: `backend/internal/network/presence.go`
- Action actor attribution: `backend/internal/actions/react.go`
- Cave replay attribution: `backend/internal/world/snapshot.go`
- Cave UI roster + action log: `frontend/venues/the-cave/index.html`

## Operator toggles / testing

There is no DB toggle for presence itself.

To verify:

- open two sessions in The Cave
- confirm join shows in the roster
- close one tab and confirm the roster updates
- send speech and reactions and confirm the actor name + role badge render

## Rebuild notes

- Backend rebuild/restart is required when presence code changes ship.
- No new environment variables were added.
- No new dependencies were added.

## Future seam

The current seam is:

- authenticated user
- active session participant
- in-memory presence registry
- action attribution from server-side identity

Future kernels can attach richer projection or audit behavior without changing the client trust model.
Status
PARTIAL

Kernel 7’s presence registry and attribution seam are implemented and proven in-process, but I could not run a live browser/WebSocket listener in this sandbox because local socket listeners and DB connections are blocked here. The server-side dispatch path, duplicate-tab dedupe, explicit presence events, and server-resolved actor attribution are all covered by a passing Go evidence test.

What Was Built

Added in-memory Cave presence tracking with duplicate-tab dedupe in backend/internal/network/presence.go
Added explicit presence event handling for:
presence/snapshot
presence/join
presence/leave
Wired Cave WebSocket handling through a testable server-side dispatch seam in backend/internal/network/ws.go
Ensured speech, reaction, and reveal/hide action broadcasts carry server-resolved actor attribution
Updated The Cave UI to render:
a “Who is here?” panel
speaker attribution
reaction attribution
role badges
Removed client-side trust for outgoing actor identity in frontend/lib/actions.js
Added operator docs for presence + attribution behavior and test command references
Evidence (MANDATORY)
Passed kernel evidence test:

=== RUN   TestKernel7PresenceAndAttributionEvidence
    kernel7_integration_test.go:34: PRESENCE first connect snapshot=1 joined=true
    kernel7_integration_test.go:40: PRESENCE duplicate connect snapshot=1 joined=false
    kernel7_integration_test.go:46: PRESENCE audience connect snapshot=2 joined=true
    kernel7_integration_test.go:58: PRESENCE final disconnect snapshot=1 left=true
    kernel7_integration_test.go:73: PRESENCE SNAPSHOT {"type":"presence/snapshot","users":[{"connected_at":"2026-05-01T02:31:50Z","display_name":"Audience Grant","handle":"audience","role":"audience","user_id":"user-audience"}]}
    kernel7_integration_test.go:80: PRESENCE JOIN {"type":"presence/join","user":{"connected_at":"","display_name":"Producer Mira","handle":"producer","role":"producer","user_id":"user-producer"}}
    kernel7_integration_test.go:87: PRESENCE LEAVE {"type":"presence/leave","user":{"connected_at":"","display_name":"Producer Mira","handle":"producer","role":"producer","user_id":"user-producer"}}
    kernel7_integration_test.go:155: SPEECH {"data":{"actor":{"display_name":"Producer Mira","handle":"producer","role":"director","user_id":"user-producer"},"actor_id":"user-producer","id":"action-speak","moment_id":1,"payload":{"text":"The fire answers."},"scope":null,"session_id":"session-kernel7","target":null,"ts":"","type":"perform/speak","visibility":null},"type":"action"}
    kernel7_integration_test.go:173: REACTION {"data":{"actor":{"display_name":"Audience Grant","handle":"audience","role":"audience","user_id":"user-audience"},"actor_id":"user-audience","id":"action-react","moment_id":2,"payload":{"kind":"applause"},"scope":null,"session_id":"session-kernel7","target":null,"ts":"","type":"react/emote","visibility":null},"type":"action"}
    kernel7_integration_test.go:175: KERNEL7_PROOF PASS
--- PASS: TestKernel7PresenceAndAttributionEvidence (0.00s)
Package sanity check:

ok  	victory/backend/internal/actions	(cached)
ok  	victory/backend/internal/network	(cached)
?   	victory/backend/internal/world	[no test files]
And the tree is clean of formatting issues:

git diff --check
passed.

How to Run (Operator Steps)

From the repo root, go to the backend:
cd /opt/victory/backend
Run the kernel evidence test:
GOCACHE=/tmp/victory-gocache go test ./internal/network -run TestKernel7PresenceAndAttributionEvidence -v
Run the backend sanity pass:
GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/network ./internal/world
For normal dev iteration, keep following the active dev workflow in Construction/Process/workflow/dev-workflow.md.
Operator Notes (CRITICAL)

Presence is intentionally in-memory and non-canonical.
Multiple tabs for the same user are deduped in the visible roster.
Presence messages now use explicit event types, not a generic wrapper.
The server, not the client, decides the actor for speech/reaction/reveal/hide actions.
The evidence path here is an in-process Go test because this sandbox blocks listener sockets and DB connections.
No new dependencies or environment variables were added for Kernel 7.
Blockers & Workarounds
BLOCKER:
I could not run a live browser/WebSocket/DB listener in this sandbox.

CAUSE:
The sandbox blocks local socket listeners and DB socket access, so httptest.NewServer, live curl, and DB-backed integration paths are not available here.

WORKAROUND:
I added and ran a pure in-process Go evidence test that exercises the actual presence registry, explicit presence event formatting, and the WebSocket dispatch seam with stubbed persistence.

OPERATOR ACTION REQUIRED:
Run the same go test commands on the real host environment if you want a live browser/WebSocket confirmation in addition to the in-process proof.

Deviations from Kernel

No product behavior deviation.
The verification method deviated from the ideal live browser/WS path because the sandbox blocked listener-based proof.
The kernel’s required semantics were still exercised: deduped presence, explicit join/leave/snapshot messages, and server-resolved actor attribution.
Known Issues

I did not capture a live browser screenshot of the Cave roster or attribution UI in this sandbox.
I did not run a live wscat session here for the same sandbox networking reason.
If you want the final “human eyeball” confirmation, that should be done on the host environment where the port listener is reachable.
Next Recommended Step

Open The Cave in a real browser on the host environment and confirm the roster, speaker names, and reaction attribution visually.
If you want, I can next turn the in-process proof into a host-side end-to-end checklist with exact commands for the browser and wscat.
Files Changed / Created

/opt/victory/backend/internal/network/ws.go
/opt/victory/backend/internal/network/presence.go
/opt/victory/backend/internal/network/kernel7_integration_test.go
/opt/victory/backend/internal/network/hub.go
/opt/victory/backend/internal/actions/react.go
/opt/victory/backend/internal/actions/reveal.go
/opt/victory/backend/internal/world/snapshot.go
/opt/victory/frontend/venues/the-cave/index.html
/opt/victory/frontend/lib/actions.js
/opt/victory/Construction/Kernels/kernel-7-presence-attribution.md
/opt/victory/Construction/OperatorLogs/operator-log.md
/opt/victory/Construction/OperatorLogs/operator-notes.md
/opt/victory/Construction/OperatorLogs/Security-notes.md
/opt/victory/Construction/Process/Construction — Contracts.txt
/opt/victory/Construction/Process/workflow/dev-workflow.md

kernel-7-presence-attribution.md
Document · MD

Open


operator-notes.md
Document · MD

Open


operator-log.md
Document · MD

Open


Security-notes.md
Document · MD

Open


dev-workflow.md
Document · MD

Open

13 files changed
+507
-120