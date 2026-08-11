# Transient / Static Stage Effect Contract (Kernel 86)

## The problem

Kernel 86 needed a presentation record for a theatrical dice projection — audience, duration, pin state — derived from an already-canonical `roll/dice` Action, without becoming a second source of roll truth. The audit explicitly ruled out reusing the existing singular `Overlay` (`backend/internal/actions/overlay.go`): it is element-bound, persistent, singular, and not queue/expiry-oriented — none of which fit a rapid sequence of transient, audience-scoped, auto-expiring effects.

## Package: `backend/internal/stageeffects`

Follows the same shape `backend/internal/venuecoordination` (Kernel 83's Group Leader/Current Turn registry) already established for this codebase: an in-memory, session-scoped `Registry`, no DB table, nothing to migrate, and a strict division of labor — the registry knows nothing about Victory roles or authority; it only tracks "which effects exist right now for this session, and which are pinned." The caller (`network/ws.go`) decides who may create/pin/dismiss an effect, using `backend/internal/rollaudience` for the roll's own audience.

```go
type Effect struct {
    ID, Type, SourceActionID, SessionID, ShowID, CohortID, Audience, ActorID, Label string
    Payload    map[string]any
    CreatedAt  time.Time
    DurationMs int
    Pinned     bool
}
```

`SourceActionID` is the one link back to the canonical `roll/dice` Action — the effect never carries its own copy of the dice result beyond a curated display `Payload` (actor/label/dice/modifier/total/explosion_count), and nothing computes a new roll from it.

## Lifecycle

- **Create** (`Registry.Create`): assigns a random `fx_`-prefixed ID and `CreatedAt`, always starts unpinned.
- **Transient expiry**: a never-pinned effect is swept lazily — on the *next* registry call that touches its session — once `DurationMs + expiryGrace(5s)` has elapsed. No background goroutine or ticker; this mirrors the codebase's existing preference for lazy, on-access cleanup over scheduled sweeps.
- **Pin** (`Registry.Pin`): marks `Pinned = true`, which exempts it from the sweep indefinitely.
- **Dismiss** (`Registry.Dismiss`): removes the effect outright — there is no "unpin but stay transiently visible" state, only pinned or gone (spec §10).
- **Reconnect** (`Registry.ListPinned`): returns every currently-pinned effect for a session; `network/ws.go`'s connect handler re-checks each one against the *connecting* viewer's current audience authority via `rollaudience.VisibleToViewer` before sending — never trusting what the effect was created with, matching how snapshot filtering itself always re-derives from current state.
- **EndSession**: clears everything for a session (mirrors `venuecoordination.Registry.EndSession`'s contract).

## Why in-memory, not a DB table

Per spec §6.1: "do not create a second durable dice-history universe." The `roll/dice` Action already is the durable record; a Stage Effect only needs to survive as long as it's visually relevant, which is bounded (a few seconds transient, or until explicitly dismissed if pinned) — the same reasoning that keeps `venuecoordination`'s Group Leader/Current Turn state in memory rather than a table.

## Wiring: one process-wide registry, not threaded through every handler

`stageEffectRegistry` is a package-level var in `network/ws.go`, matching how `rollDiceLimiter` and `wsMessageLimiter` are already declared in that same file, rather than a constructor parameter threaded through `ServeVenueWS`/`ServeCaveWS`/`handleVenuePayload`/`handleCavePayload` (which would have required touching every existing caller and test of those functions, including `network/dice_roll_test.go`'s pre-existing `handleCavePayload(hub, nil, client, payload)` call). This differs from `venuecoordination.Registry`, which *is* threaded explicitly into Storyboards' HTTP handlers — that registry needs per-call dependency injection for testability across multiple call sites; `stageEffectRegistry` has exactly one consumer (`ws.go`'s `roll/dice` case and the new `stage_effect/pin`/`stage_effect/dismiss` cases) and no DB/pool dependency of its own.

## Authority: `stageEffectAuthorized`

- The roller may always pin/dismiss their own effect.
- Director+ (`rollaudience.IsDirectorPlus`) may pin/dismiss any effect **except** one whose `Audience == "private"` — a Private roll has no authorized viewer besides its own roller in the first place, so no one else should be able to touch its presentation state either. This check short-circuits before any DB lookup when the requester is the actor or the effect is Private, so a forged dismiss attempt against a Private effect never needs the recipient-resolution machinery at all.
- Everyone else is rejected with `{"type":"error","error":"forbidden"}`.

## New WS message types

```text
client -> server: "stage_effect/pin"     {effect_id}
client -> server: "stage_effect/dismiss" {effect_id}
server -> client: "stage_effect"          (on roll/dice, alongside the existing "action" message)
server -> client: "stage_effect_pinned"
server -> client: "stage_effect_dismissed"
server -> client: "stage_effects/pinned"  (once, right after "snapshot", on connect/reconnect)
```

All delivery for these — creation, pin confirmation, dismiss confirmation — goes through the same targeted-delivery path described in `Construction/Network/targeted-live-delivery.md`, using the roll's own resolved `audienceMode`/`cohortId`, never a separate broadcast.
