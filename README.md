# Victory

Victory is a live virtual tabletop theater: a real-time stage for running tabletop RPG shows, cast as a theater rather than a generic VTT — Directors and Producers run live Shows, Cast play Characters, Crew handle bounded backstage tasks, and an Audience watches and participates from their own seat. Go backend, no-build-step vanilla JS/Vue frontend, Postgres, optional Discord integration for chat/voice/OAuth.

Live at [victory.amurray.family](https://victory.amurray.family).

## Stack

- **Backend:** Go (`backend/`), one package per domain concern under `backend/internal/`
- **Frontend:** static HTML/JS per venue (`frontend/venues/`), no bundler; shared runtime in `frontend/lib/stage-runtime/`; Vue for newer interaction-heavy surfaces
- **Database:** PostgreSQL, sequential kernel-tagged SQL migrations (`backend/migrations/`)
- **Deployment:** Docker/Podman Compose (`packaging/podman/`), plus a Windows consumer installer (Velopack-based)

## Getting started

See **[`Docs/Developer/Setup Guide.md`](Docs/Developer/Setup%20Guide.md)** for prerequisites and repository layout, and **[`Construction/Process/workflow/dev-workflow.md`](Construction/Process/workflow/dev-workflow.md)** for the actual dev-mode/install-mode commands and test setup — that doc is the maintained source of truth, not duplicated here.

## Documentation map

- **[`Docs/`](Docs/)** — product, developer, and operator-facing documentation (setup, Discord integration, backups, recovery, glossary, venue index)
- **[`Construction/`](Construction/)** — build history and living design record: every kernel spec, its proof of what was actually built, and current-state canon (start with `Construction/README.md`)

## License

No license has been chosen yet; all rights reserved by default.
