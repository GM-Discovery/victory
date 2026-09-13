# Kernel 39 — Discord Gateway Surface

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-06-06 to 2026-06-08
**Implementation status:** IMPLEMENTED — status not recorded (no surviving reportback; recovered from code/commits/later migration)
**Evidence sources:** commits `05c6894` ("Add Discord mic and session control functionality"), `1d9079d` ("Add Discord Gateway integration tests and database migration"), `89af9ba` ("Refactor Discord Mic and Chat Bridge Integration"); `backend/internal/identity/discord_gateway.go` (`EnsureKernel39DiscordGatewaySurface`); `backend/cmd/victory/main.go:190-191` (bootstrap call, still present verbatim today); `backend/migrations/054_kernel72_discord_gateway_surface_ddl.sql` (its own header states: "DDL extracted verbatim from identity.EnsureKernel39DiscordGatewaySurface")

## Reconstructed purpose

Establish real Discord Gateway (websocket) integration — ingesting live Discord events (message create/edit/delete, backfill of sparse payloads) into Victory's chat-bridge and mic-control model founded by Kernels 35-38, rather than relying on one-off REST calls.

## What evidence proves was built

Unlike Kernels 35-38, this kernel's schema was originally created via an idempotent Go bootstrap function (`EnsureKernel39DiscordGatewaySurface`) rather than a numbered migration file — the function name itself is the clearest surviving evidence of the kernel boundary, and it is still called unconditionally at server startup today (`main.go:190`, `log.Fatalf("kernel 39 discord gateway bootstrap failed...")`). Real test coverage exists across three files (`discord_gateway_test.go`, `discord_chat_bridge_test.go`, and the `identity` package's own `discord_gateway_test.go`) covering message edits and sparse-payload backfill specifically. Kernel 72 later extracted this same bootstrap's DDL verbatim into a proper numbered migration (054) — direct confirmation this was a real, still-load-bearing schema surface 30+ kernels later, not dead scaffolding.

## Files/systems affected

`backend/internal/identity/discord_gateway.go`, `backend/internal/network/discord_gateway_test.go`, `backend/internal/network/discord_chat_bridge_test.go`, `backend/cmd/victory/main.go`, later `backend/migrations/054_kernel72_discord_gateway_surface_ddl.sql`.

## Known deviations / later corrections

This is the one kernel in the 35-39 Discord-foundation arc whose schema was NOT captured in a numbered migration at the time — a genuine historical gap (bootstrap-function-only DDL) that Kernel 72, over a month later, treated as worth formally correcting by extracting it into `054_kernel72_discord_gateway_surface_ddl.sql`. Product terminology shifted from "Discord mic" to "House Mic" in this same window (commit `89af9ba`).

## What this kernel handed to the next kernel

A live Gateway ingestion path that Kernel 40 ("Runtime Hardening + House Mic Regression Harness") explicitly hardened and regression-tested, and that Kernel 72 later formalized into standard migration tooling.

## Confidence / unresolved gaps

High confidence on scope, moderate-high confidence on exact date boundary (the feature spans three commits across 2026-06-06 to 2026-06-08, plausibly one continuous kernel pass rather than several). No original spec or contemporary reportback survives.
