# Kernel 37 — Discord Mic and Session Threads

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-06-01
**Implementation status:** IMPLEMENTED — status not recorded (no surviving reportback; recovered from migrations/commit)
**Evidence sources:** commit `4a09abd` ("Add Discord mic functionality with tests and database migration"); migration `backend/migrations/022_kernel37_discord_mic_threads.sql`; `discord_mic_test.go`; frontend asset-cache-busting tags `?v=kernel37-1` on `/lib/discord-mic.js`

## Reconstructed purpose

Introduce the "Discord mic" feature — letting a Victory session control/reflect voice-channel-style presence via Discord — backed by a new session-threads table (public key on server-link settings, plus a table for Discord session threads per the commit's own description).

## What evidence proves was built

The commit adds `discord_mic_test.go` (real test coverage) alongside the migration, and ships a frontend script `/lib/discord-mic.js` versioned `kernel37-1` in its own cache-busting query string — later commits bump this same tag to `kernel39-2`, directly evidencing the file's lineage across Kernels 37 through 39.

## Files/systems affected

`backend/migrations/022_kernel37_discord_mic_threads.sql`, `discord_mic_test.go`, `frontend` mic control script (`/lib/discord-mic.js`).

## Known deviations / later corrections

Kernel 40's own operator-log entry ("Runtime Hardening + House Mic Regression Harness") explicitly notes "No Discord voice/audio path was built for this kernel" while hardening the surrounding mic-control machinery — implying the mic feature founded here was control/signaling only, never full voice/audio, through at least Kernel 40. Later renamed in product-facing copy to "House Mic" terminology by commit `89af9ba` (Kernel 39/40 era).

## What this kernel handed to the next kernel

The mic/session-thread model Kernel 38 built chat-bridging on top of, and Kernel 39 (Discord Gateway) later consumed directly (`kernel39-2` asset tag reuses the same script this kernel introduced).

## Confidence / unresolved gaps

High confidence on scope and date. No original spec or contemporary reportback survives.
