# Current Architecture Manifest

**Produced by:** Kernel 99, 2026-09-12, per spec §53. This is current state, not history — see `Kernel Index.md`/`Kernel Lineage.md` for how it got this way. Update this file when architecture changes; it is not auto-generated.

---

## Services

- **Backend:** single Go binary (`backend/cmd/victory`), serves HTTP + WebSocket on one port.
- **Recovery tool:** `backend/cmd/victory-recover` — break-glass CLI (Kernel 76), documented in `Docs/Operator/Break-Glass Recovery Guide.md`.
- **Database:** PostgreSQL, single schema, 114 embedded checksummed migrations (`backend/migrations/`), applied via `internal/migrate` in filename order — no separate migration tool required.
- **Reverse proxy / TLS (self-hosted/Linux path):** Caddy.
- **Remote access (Windows consumer path, Kernel 100):** Cloudflare Tunnel — Quick Tunnel by default, Named Tunnel optional; backend binds loopback-only; tunnel is outbound-only (no port forwarding).

## Frontend technologies

- Hand-rolled JS/DOM and Pixi-based venues (the majority — the original architecture from Kernels 1-30 onward).
- **Vue** — introduced Kernel 94, scoped to Storyboards only as of this writing. Kernel 95 (sitewide visual unification, never executed) was meant to carry Vue/visual-language patterns further; it has not happened yet — do not assume Vue elsewhere in the product.
- **Pixi.js** — stage rendering engine for spatial venues (the-cave, First Theater, Catharsis). Introduced Kernel 29.

## WebSocket services

Single `Hub`/`Client` model (`backend/internal/network`), keyed by connection pointer (not by user — multiple tabs get independent clients). All stage-mutation and chat/dice actions route through `actions.CanAct`; no parallel HTTP path exists for the same actions.

## Data directories

- **Linux/server:** configurable via environment (see `Docs/Developer/Setup Guide.md` and `Docs/Operator/Backup Guide.md` for the exact variables); database is Postgres-managed, asset files live under a configured storage root.
- **Windows consumer (Kernel 100):** `%LocalAppData%\Victory` — database, assets, and configuration all live under this single directory, per the First-Time Operator Guide.

## Install/runtime modes

Four distinct modes, per `Construction/Process/workflow/dev-workflow.md` (updated this kernel):
1. **Local developer mode** — direct Go run + Postgres, hot-reload frontend assets.
2. **Packaged/install mode** — the embedded-migration, single-binary path.
3. **Server/Linux mode** — Caddy-fronted, systemd-managed, manual/scripted backup (Kernel 77's automation is available but not on-by-default).
4. **Windows consumer mode (Kernel 100)** — Velopack installer/updater, Cloudflare Tunnel remote access, no Git/dev tooling required by the end user.

## Major domain packages (backend/internal/)

- `access` — venue access, Location role resolution (`CurrentLocationRoleForLocation`, `CurrentDefaultLocationRole`, `IsOperatorUser`).
- `participation` — Show Run roster resolution (`ResolveParticipationContext`, `LegacyLookupVenueRole`) — the canonical seat of Cast/Player/Crew authority (Kernel 66, hardened Kernel 97).
- `showruns` — production/management authority (`CanManageShowRun`), Show Run lifecycle.
- `actions` — the single authority gate (`CanAct`) for all WebSocket stage/chat/dice mutations.
- `stageobjects` — canonical stage-object state, visibility projection (`CanPerceive`, `IsBackstageRole`), cue control (Kernel 90).
- `showings`, `shows` — Showing/Show domain model.
- `scenes` — Scene domain object, live-venue composition capture.
- `storyboards` — Storyboards/Timeline domain (Kernel 80-82, 94).
- `ewrite` — Markdown+FTS document system (Kernel 78-79).
- `audienceadmission` — Showing-scoped Audience admission (Kernel 71, 93).
- `identity` — accounts, Discord OAuth/server-link/gateway, invites.
- `dbtest` — shared test-database entry point (`OpenTestPool`) for all DB-touching tests.

## Deployment components

- Embedded migration runner (no external migration tool).
- `victory-recover` break-glass CLI.
- Support bundle generator (Windows, Kernel 100 — `SupportBundle.cs`; no equivalent yet for the Linux/server path).

## Updater/connectivity components (Kernel 100)

Velopack-based Windows updater: checks every 4h/15min, applies frontend-only updates immediately, gates backend updates behind a quiet window (2:30 AM local) or an explicit session-closed signal (`/showtime <code> end`). Single release channel; version is a build counter, not semver. No auto-updater exists for the Linux/server path.
