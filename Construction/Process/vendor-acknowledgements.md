# Vendor Acknowledgements

## Current Note
This is the current dependency and infrastructure acknowledgement file.

The older [historical copy](/opt/victory/Construction/OperatorLogs/vendor-acknowledgements.md) remains as record, but this file should track the live stack.

## Infrastructure
- Docker
- Docker Compose
- Caddy
- PostgreSQL 16 Alpine

## Backend Runtime
- Go `1.25`

## Current Go Dependencies
- `github.com/gorilla/websocket`
- `github.com/jackc/pgx/v5`
- `golang.org/x/crypto`
- `golang.org/x/image`

## Client / Renderer Dependencies
- PixiJS `7.4.3`
- loaded from `https://cdn.jsdelivr.net/npm/pixi.js@7.4.3/dist/pixi.min.js`

Indirect dependencies currently present in `go.mod`:
- `github.com/jackc/pgpassfile`
- `github.com/jackc/pgservicefile`
- `github.com/jackc/puddle/v2`
- `golang.org/x/sync`
- `golang.org/x/sys`
- `golang.org/x/text`

## Current Usage Notes
- `gorilla/websocket` powers Cave WebSocket transport
- `pgx` and `pgxpool` power Postgres access
- `x/crypto` provides Argon2 password hashing
- `x/image` is available for current asset/image handling work
- PixiJS is an experimental stage/worldspace renderer in First Theater only; it is not the source of app truth
- PostgreSQL `pgcrypto` remains relevant for `gen_random_uuid()`
- Discord OAuth is an external platform/API integration, not a bundled runtime dependency
- Kernel 32 uses the Go standard library for OAuth requests, so no new Go OAuth package was added
- Kernel 33 also adds no new external dependency; the operator bootstrap CLI reuses the existing Go and pgx stack

## Windows Installer Bundle (Kernel 100, `windows-native`)
The authoritative, version-pinned list for the Windows consumer installer/launcher (Velopack, cloudflared, PostgreSQL EDB binaries, Caddy, plus the .NET/NuGet packages VictoryLauncher.csproj pulls in) lives in `packaging/windows/VictoryLauncher/THIRD_PARTY_NOTICES.md`, not here — it ships alongside `VictoryLauncher.exe` in every install (wired via `CopyToOutputDirectory` in the csproj), which this repo-level file cannot do. Keep it in sync with `VictoryLauncher.csproj`/`backend/go.mod` rather than duplicating its contents here.
One note worth recording at this level: using Cloudflare's Tunnel *service* (as opposed to redistributing the open-source `cloudflared` binary, Apache 2.0) is separately subject to Cloudflare's own terms of service, between the Operator and Cloudflare directly — Victory does not accept that on the Operator's behalf, and nothing in the installer requires a Cloudflare account for the default Quick Tunnel path.

## Dependency Philosophy
- keep external dependencies small and intentional
- prefer the standard library where practical
- document new runtime or security-sensitive dependencies when introduced

## Historical Note
This file supersedes older vendor snapshots that were written closer to Kernel 2.
