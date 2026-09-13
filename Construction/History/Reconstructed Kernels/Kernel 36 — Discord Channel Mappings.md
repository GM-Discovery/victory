# Kernel 36 — Discord Channel Mappings

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-05-31 to 2026-06-01
**Implementation status:** IMPLEMENTED — status not recorded (no surviving reportback; recovered from migrations/commit)
**Evidence sources:** commit `adce59d` (same commit as Kernel 35) plus follow-up commit `9fb08e4` ("enhance Discord channel mapping repair summary with detailed failure messages"); migration `backend/migrations/021_kernel36_discord_channel_mappings.sql`; `backend/internal/identity/discord_channel_mapping.go`

## Reconstructed purpose

Map specific Victory venues/purposes to specific Discord channels on the server linked in Kernel 35, so later mic/chat-bridge kernels would know which Discord channel corresponds to which in-product location.

## What evidence proves was built

Migration `021_kernel36_discord_channel_mappings.sql` lands in the same commit as Kernel 35's migrations. The very next day's commit (`9fb08e4`) extends `discord_channel_mapping.go` with more detailed failure-message handling for a "channel mapping repair" flow — proving the mapping model was real and load-bearing enough to need a repair/self-heal path almost immediately.

## Files/systems affected

`backend/internal/identity/discord_channel_mapping.go`, `backend/migrations/021_kernel36_discord_channel_mappings.sql`.

## Known deviations / later corrections

None found specific to this kernel's own scope.

## What this kernel handed to the next kernel

The channel-mapping model Kernel 37 (mic/session threads) and Kernel 38 (chat bridges) both depend on to know which Discord channel a given Victory session/venue corresponds to.

## Confidence / unresolved gaps

High confidence on scope and date. No original spec or contemporary reportback survives.
