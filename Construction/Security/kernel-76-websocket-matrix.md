# Kernel 76 — WebSocket Matrix

**Source:** `backend/cmd/victory/main.go` lines 934–937; handlers in
`backend/internal/network/ws.go` and `profile_ws.go`.
**Evidence:** a Go `gorilla/websocket` client dialled the live public host
(`wss://victory.amurray.family`) as anonymous and as an authenticated outsider. Raw output is
reproduced below.

---

## 1. Endpoints

| Path | Handler | Scope | Anonymous | Authenticated outsider |
|---|---|---|---|---|
| `/ws/the-cave` | `ServeCaveWS` | venue `the-cave` | rejected | rejected |
| `/ws/catharsis` | `ServeVenueWS(…, "catharsis")` | venue `catharsis` | rejected | rejected |
| `/ws/first-theater` | `ServeVenueWS(…, "first-theater")` | venue `first-theater` | rejected | rejected |
| `/ws/player-profile` | `ServeProfileWS` | the session user's own profile | rejected | own scope only |

---

## 2. Authorization sequence

`ServeVenueWS` and `ServeCaveWS` enforce, in order:

1. `access.CurrentUserIDFromRequest` — session cookie resolved server-side.
2. `access.UserCanAccessVenueSlug(userID, venueSlug)` — venue admission.
3. `identity.ResolveActiveVenueSessionID` — active Session participation.

Each failure writes a typed error frame and closes. The venue slug is bound at registration
time in `main.go`, not read from the client, so a connection cannot name a venue it was not
routed to. Actor identity and role are resolved from the session, never accepted from the
client.

---

## 3. Observed results

```
anon /ws/the-cave                  connected, first_frame={"error":"forbidden","type":"error"}
anon /ws/player-profile            connected, first_frame={"error":"not_authenticated","type":"error"}
outsider(alice) /ws/catharsis      connected, first_frame={"error":"forbidden","type":"error"}
outsider(alice) /ws/first-theater  connected, first_frame={"error":"forbidden","type":"error"}
outsider(alice) /ws/player-profile connected, read_err=i/o timeout   <- own socket, no error frame
bad-origin blocked                 connected, first_frame={"error":"forbidden","type":"error"}
```

Reading these:

- **Anonymous is rejected on every venue socket.** No subscription, no presence, no broadcast.
- **An authenticated outsider is rejected exactly as an anonymous one is.** Holding a Victory
  account grants nothing at a venue you are not admitted to — the finding that mattered on the
  HTTP side (K76-H02) has no WebSocket equivalent.
- **`/ws/player-profile` accepts the outsider and then goes quiet.** That is correct: it is her
  own profile socket, scoped to her own user id, with nothing to broadcast.

## 4. Cross-scope isolation

`ServeVenueWS` closes over a fixed `venueSlug` per registration, so "subscribe to Production
B's venue while holding Production A's session" is not expressible in the protocol — there is
no client-supplied venue, show, or session parameter to forge. Isolation is structural rather
than checked, which is the stronger form.

Session revocation propagates on the next connection: `CurrentUserIDFromRequest` re-resolves
the cookie at dial time, so a revoked cookie cannot open a new socket. An **already-open**
socket is not torn down when its session is revoked — see K77-03.

## 5. Origin checking

```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" { return false }
    return origin == "http://"+r.Host || origin == "https://"+r.Host
}
```

Strict same-host comparison, and an absent `Origin` is rejected rather than allowed. This is
the correct posture and closes cross-site WebSocket hijacking.

## 6. Limits

| Control | Before | After |
|---|---|---|
| Inbound frame size | unbounded | 256 KiB (`SetReadLimit`, K76-M03) |
| Message type allowlist | switch on `type`, unknown ignored | unchanged |
| Connection count per user | none | none — K77-04 |
| Message rate | none | none — K77-04 |

## 7. Open items

- **K76-L01** — the upgrade completes before authentication. Authorization is still correct;
  the cost is that an anonymous client can drive a full handshake before rejection.
- **K77-03** — revoking a session does not disconnect sockets already open under it.
- **K77-04** — no per-connection or per-message rate limiting.
