# Kernel 35 — Discord Server Link and Bootstrap

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-05-31
**Implementation status:** IMPLEMENTED — status not recorded (no surviving reportback; recovered from migrations/commit)
**Evidence sources:** commit `adce59d` ("feat(discord): implement Discord server link and bootstrap functionality"); migrations `backend/migrations/019_kernel35_discord_server_link.sql`, `backend/migrations/020_kernel35_discord_server_bootstrap.sql`; `backend/internal/identity/discord_server_link.go`, `backend/internal/identity/discord_server_bootstrap.go`

## Reconstructed purpose

Give a Location Operator a way to link a real Discord server to their Victory installation and bootstrap the initial server-side configuration Victory needs (channel/role scaffolding) before later kernels (36-39) could build channel mappings, mic control, chat bridging, and gateway ingestion on top of it.

## What evidence proves was built

The commit's own diff introduces `discord_server_bootstrap.go` (GET/POST handlers for managing Discord server settings) and `discord_server_link.go` (installation, OAuth-style callback handling, and unlinking) verbatim in the same commit as the two migrations bearing this kernel's number. No spec file or reportback survives; the migration filenames themselves are the primary evidence of the kernel boundary.

## Files/systems affected

`backend/internal/identity/discord_server_link.go`, `backend/internal/identity/discord_server_bootstrap.go`, `backend/migrations/019_kernel35_discord_server_link.sql`, `backend/migrations/020_kernel35_discord_server_bootstrap.sql`.

## Known deviations / later corrections

None found specific to this kernel. The broader Discord-mic subsystem it founded was later hardened by Kernel 40 ("Runtime Hardening + House Mic Regression Harness").

## What this kernel handed to the next kernel

The server-link/bootstrap surface Kernel 36 built channel mappings on top of, in the very next commit.

## Confidence / unresolved gaps

High confidence on scope and date (migration filenames directly embed "kernel35"). No original spec or contemporary reportback survives — this record is reconstructed entirely from commit/migration/code evidence, not from a lost planning document.
