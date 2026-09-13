# Developer Setup Guide

This is the orientation doc for a new developer. It tells you where things live and where to go next; it does not duplicate the exact commands — those live in `Construction/workflow/dev-workflow.md`, which is the maintained source of truth for day-to-day commands and gets updated as they change.

## What you need before starting

- Go (backend)
- Docker (Postgres, and the full install-mode stack)
- A browser — no frontend build tooling is required; venues are served as static HTML/JS/CSS directly (see below).

No Node/npm install is required for normal development. Playwright (browser-proof scripts only) is the one exception — installed on demand into a scratch directory, not vendored in the repo.

## Repository layout

```
backend/            Go backend — internal/<package> holds domain logic (see below)
backend/migrations/ Sequential, kernel-tagged SQL migrations
frontend/venues/     One directory per venue, served directly (no bundler)
frontend/lib/        Shared frontend runtime (stage-runtime, socket handling, etc.)
Construction/        Kernel specs, operator logs, design docs, workflow docs
Docs/                 Product- and developer-facing documentation (this file's home)
packaging/podman/    Docker Compose stack for both dev-mode Postgres and full install mode
```

## Clone, run, test

1. Clone the repository.
2. Follow **Dev Mode** in `Construction/workflow/dev-workflow.md` to start Postgres and run the backend on the host.
3. Open a venue directly in your browser (e.g. `http://127.0.0.1:8081/venues/the-cave/`).
4. Run the test suite: set up the dedicated `victory_test` database once (see the same doc's "Dedicated test database" section), then `go test ./...` from `backend/`.

For validating the deployable path (not day-to-day dev work), use **Install Mode** in the same document — the full Docker Compose stack, the same path production uses.

## Backend structure

Domain logic lives under `backend/internal/<package>/` — one package per concern (`access`, `participation`, `showruns`, `scenes`, `storyboards`, `ewrite`, and so on). Before adding a new permission check anywhere, read the "Domain Authority Helpers" section of `dev-workflow.md` and `Construction/Identity/Canonical Role and Authority Resolution.md` — most authority questions already have a canonical answer.

## Frontend structure

See `dev-workflow.md`'s "Frontend Conventions" section for the current Vue/Pixi/shared-shell guidance. In short: Vue for newer interaction-heavy surfaces (Storyboards is the reference implementation), PixiJS as a renderer-only layer for stage composition, and a shared shell primitive (`frontend/venues/shared/`, `frontend/lib/stage-runtime/`) that Catharsis and First Theater both build on.

## Migrations

See `dev-workflow.md`'s "Creating a new migration" section. In brief: sequential numbering, `NNN_kernelXX_description.sql`, must be safe to replay against an already-migrated database.

## Deployment / packaging architecture

Three runtime modes exist — dev, Docker install (server/Linux), and the Windows consumer installer (Kernel 100, Velopack-based with bundled Postgres/Caddy/cloudflared). See `Docs/Operator/` for the operator-facing side of install/update/backup, and `Construction/Kernels/Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates.md` plus its ledger for the actual packaging implementation.

## Where to find history

`Construction/History/Kernel Index.md` maps every feature back to the kernel(s) that introduced or hardened it — check it before making an assumption about *why* something is built the way it is.
