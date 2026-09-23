# Victory

Victory is a live virtual tabletop theater: a real-time stage for running tabletop RPG shows, cast as a theater rather than a generic VTT — Directors and Producers run live Shows, Cast play Characters, Crew handle bounded backstage tasks, and an Audience watches and participates from their own seat. Go backend, no-build-step vanilla JS/Vue frontend, Postgres, optional Discord integration for chat/voice/OAuth.

Live and in active production at [victory.amurray.family](https://victory.amurray.family) — this isn't a tech demo, it's the actual house running actual shows.

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

## Licensing

Victory is **source-available**, not open source in the OSI sense — you can read it, run it, and build on it, but the terms are Victory's own, not a stock OSS license.

- **Free** for noncommercial use (personal, hobby, community games, classroom/academic use), and for **Small Commercial Use** — running paid shows or a paid service on Victory — up to US $2,600/year in Victory-related revenue, with no signup required.
- **Paid, published tiers** kick in past that threshold ($49–$999/year depending on revenue band, or a custom agreement above $250k/year) — administered by Discovery Games Interactive LLC on behalf of the owner. No negotiation needed for a standard tier; you just pay it.
- Modify and distribute freely under copyleft terms (share-alike, source stays available) — see the license for the exact conditions.
- The **Victory**, **Victory Theater**, and **Victory Theater VTT** names/marks are separately trademark-policed; forks are welcome but rebrand if they drop required packages.

Full terms live in [`licensing/`](licensing/): [`VICTORY_COMMUNITY_LICENSE.md`](licensing/VICTORY_COMMUNITY_LICENSE.md) (the license itself), [`COMMERCIAL_TERMS.md`](licensing/COMMERCIAL_TERMS.md) (paid tiers), [`TRADEMARK_AND_OFFICIAL_BUILD_POLICY.md`](licensing/TRADEMARK_AND_OFFICIAL_BUILD_POLICY.md) (branding/forks), [`CONTRIBUTOR_TERMS.md`](licensing/CONTRIBUTOR_TERMS.md) (if you're submitting a PR), and [`COMPONENT_LICENSES.md`](licensing/COMPONENT_LICENSES.md) (third-party and bundled-content licenses, including Socio and Niava).

**Paying for a tier:** [Buy Me a Coffee](https://buymeacoffee.com/grantamurray) — a placeholder payment channel until a dedicated licensing checkout exists. Include your production/organization name and the tier from [`COMMERCIAL_TERMS.md`](licensing/COMMERCIAL_TERMS.md) in a note.

Licensing questions: discord @gm_discovery github @gm-discovery
