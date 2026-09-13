# Kernel 38 — Discord Chat Bridges

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-06-02
**Implementation status:** IMPLEMENTED — status not recorded (no surviving reportback; recovered from migrations/commit)
**Evidence sources:** commit `dbae39d` ("feat: refactor chat UI for improved layout and responsiveness across venues"); migration `backend/migrations/023_kernel38_discord_chat_bridges.sql`; `discord_channel_mapping_test.go` update in the same commit

## Reconstructed purpose

Bridge in-product venue chat with the Discord channels mapped in Kernel 36, so messages could flow between a Victory venue and its linked Discord channel.

## What evidence proves was built

Migration `023_kernel38_discord_chat_bridges.sql` lands in a commit whose headline message is about chat UI refactoring — the commit bundles a genuine frontend chat-layout change together with the backend bridge migration and an update to `discord_channel_mapping_test.go`, indicating the chat-bridge feature and its UI consequences shipped together.

## Files/systems affected

`backend/migrations/023_kernel38_discord_chat_bridges.sql`, chat UI components across venues, `discord_channel_mapping_test.go`.

## Known deviations / later corrections

Later folded into the broader "Discord Mic and Chat Bridge Integration" refactor (commit `89af9ba`, Kernel 39/40 era), which renamed control-message terminology to "House Mic" for product consistency.

## What this kernel handed to the next kernel

A working chat-bridge surface that Kernel 39 (Discord Gateway) extended into full bidirectional message ingestion (edits, backfill of sparse payloads) rather than a one-way bridge.

## Confidence / unresolved gaps

High confidence on scope and date. No original spec or contemporary reportback survives.
