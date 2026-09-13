# Kernel 99 — Fresh Install Proof

**Date:** 2026-09-12. Per spec Part II (§23-28). Proves the documented Linux/developer install path against a genuinely empty, fresh database using only the repository's own documented tooling (`scripts/smoke/fresh-install.sh --local`, per `Construction/Process/deployment/fresh-install.md`) — no manual SQL, no copied config, no tribal knowledge.

## Result: PASS, with one real defect found and fixed along the way

## Environment

A machine with an already-running, real, long-lived Victory dev stack (Podman-managed `victory-postgres`/`victory-backend`, 9 hours uptime at time of proof — left completely untouched throughout). The proof ran an entirely separate, isolated Docker-managed Postgres container on non-colliding ports (`POSTGRES_HOST_PORT=25432`, `BACKEND_PORT=28081`), created a uniquely-named throwaway database (`victory_fresh_<timestamp>_<random>`), and tore both down completely afterward — confirmed via `docker exec ... psql -l` (no leftover `victory_fresh_*` database) and `docker compose down` (container/network fully removed).

## What the proof covered

The documented smoke script exercises, from a truly empty database, in one run: schema migration bootstrap (`000`/`001` applied manually, all subsequent migrations applied by the real binary's own boot sequence — confirmed this is the actual documented mechanism, not a proof artifact); anonymous-auth rejection on every major API surface; local-auth signup; Operator bootstrap-producer CLI (`go run ./cmd/victory-bootstrap producer --handle ...`), confirmed idempotent; account authority reflecting the bootstrap grant; the full Trailer/Face/Workbook pipeline for two independent fresh accounts; private relationship/journal privacy boundaries; Third Place Headshot lifecycle including a real Kernel 68 face-readiness gate; the full Show Run → ticket-request/invite → roster → Character-selection pipeline for three more fresh accounts, including a real rejection of a Director's attempt to add a Player without a ticket; Audience self-join and Program curation; Show creation, short-code generation, PATCH round-trips, archiving; venue-visibility gating by Face-readiness and backstage authority; Production creation; and Scene reuse/staging/archiving across two independent Shows. All of it passed.

## Real defect found: stale smoke-test assertion (not a product bug)

The proof hard-failed at `/showtime <code>` with: *"expected /showtime's response to suggest /mic hot"*. Investigation (not a guess — traced via `git log -L` and `git blame`) found:

- The assertion was written in Kernel 71 (commit `a93d68a`, 2026-07-16), when `/showtime`'s response apparently always suggested the literal command `/mic hot`.
- Kernel 92 (commit `7bd1733`, "showtime scheduler, showing nicknames, chat bridge") reworked the message-building code (`backend/internal/showtime/http.go:82-97`) into the current, more accurate format: it now reports real Chat Bridge status contextually (`not ready` / `connected` / `not configured for this venue` / etc.) rather than always suggesting a command that might not even work yet.
- The smoke test was never updated to match. This is confirmed structurally, not just by inference: grepping the entire backend for the literal string `/mic hot` in non-test code returns zero hits — no code path has produced that string since Kernel 92 shipped.

**Fixed:** the assertion now checks for the presence of a `chat_bridge_message` field (always present per the current code) instead of the dead literal string. See `scripts/smoke/fresh-install.sh` for the fix and its inline comment citing this exact evidence. This is exactly the kind of "undocumented intervention" the spec asks to surface — a documented, supported proof path that silently didn't work — now closed so the next person to run this script gets a clean PASS on the first try.

## Documentation defect found and fixed

`Construction/Process/deployment/fresh-install.md` step 7 referenced `database/migrations/` — the real path is `backend/migrations/`, confirmed by reading the smoke script itself. Fixed in place.

## No tribal knowledge required

Every step taken was either directly from `Construction/Process/deployment/fresh-install.md` or from reading the smoke script's own `--help`/usage text (which documents the `POSTGRES_HOST_PORT`/`BACKEND_PORT` overrides used here to avoid colliding with the pre-existing real dev stack — a normal, documented override, not an undocumented workaround). No manual SQL, no hand-created rows, no copied `.env`, no knowledge that wasn't either in the repo's own docs or discoverable from `--help`.

## Fresh database contents audit (spec §26)

Confirmed via the proof run itself: a fresh database contains only the two structural migrations applied directly (schema only, no seed rows) plus whatever the real binary's boot sequence creates (the default Location, default venues) — no Murray-family personal data, no Grant account, no production messages, no test Characters, and no production Show history, because the database never existed until this proof created it and was destroyed immediately after.

## What this proof does NOT cover

- **Windows install path** — Kernel 100 already proved this on real hardware (`kernel-100-windows-installer-ledger.md`); not re-proven here, per spec §59's own guidance to document the real installer rather than duplicate its proof.
- **Discord-configured path** — this proof deliberately ran with blank Discord config (the documented "should still boot, report unavailable" case) to prove the no-Discord path is genuinely safe. The Discord-configured path is not separately re-proven here; Kernel 100's ledger and the Discord Integration Guide document that surface.
- **A true second machine / network-isolated proof** — this ran on the same machine as the existing dev stack (deliberately isolated by port/container name, not by physical/network separation). This satisfies "no existing Victory database, .env, Operator, or lot" (a genuinely fresh database was used throughout) but not "a machine that has never had Victory on it at all."
