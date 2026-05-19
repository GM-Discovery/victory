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

## Dependency Philosophy
- keep external dependencies small and intentional
- prefer the standard library where practical
- document new runtime or security-sensitive dependencies when introduced

## Historical Note
This file supersedes older vendor snapshots that were written closer to Kernel 2.
