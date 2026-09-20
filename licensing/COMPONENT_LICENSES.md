# Victory Component and Content Licenses

This document separates the licenses governing the application, game rules, settings, art, and third-party software. It must be completed from the actual release repository before publication.

## First-party and bundled components

| Component | Owner | Proposed or existing license | Approval/status |
| --- | --- | --- | --- |
| Original Victory application code | Grant A. Murray | Victory Community License 1.0 or applicable commercial terms | Owner confirmed |
| Niava setting material | Grant A. Murray | CC BY-ND 4.0 | Owner and existing license confirmed; keep separate from adaptable core rules |
| Catharsis Theater package | Grant A. Murray | Victory Community License 1.0 | Owner and proposed license confirmed; required in branded builds |
| Victory documentation | Grant A. Murray | Victory Community License 1.0 | Owner and proposed license confirmed |

## Required official packages versus license ownership

The requirement that Victory-branded builds include Socio and Catharsis Theater does not merge those packages into the software license. Each package retains its stated content license.

Standard commercial Victory permission covers operation of the bundled product only to the extent Grant A. Murray owns or may license each component and has authorized Discovery Games Interactive LLC to sublicense it. It does not override a third party’s terms.

## Currently documented third-party software

Verified 2026-09-20 directly against this repository: `backend/go.mod`/`go.sum`, the actual compiled dependency graph of the shipped binaries (`go list -deps ./cmd/victory/... ./cmd/victory-bootstrap/...`), the pinned Postgres image, `backend/Dockerfile`, and `packaging/podman/compose.yml`. Only modules actually compiled into a shipped binary are listed — `backend/go.mod`'s module graph also resolves a handful of test-only modules (`stretchr/testify` and its own transitive deps) that are not imported anywhere in this codebase and are not part of any shipped binary; they're intentionally omitted here.

| Component | Version (as pinned in this repo) | License | Verified from |
| --- | --- | --- | --- |
| Go toolchain/runtime (statically embedded in the compiled binaries) | go1.26 | BSD-3-Clause ("Go" license) | `backend/go.mod` |
| `github.com/gorilla/websocket` | v1.5.3 | BSD-2-Clause | module cache `LICENSE` |
| `github.com/jackc/pgx/v5` (incl. `pgconn`, `pgproto3`, `pgtype`, `pgxpool`) | v5.9.2 | MIT | module cache `LICENSE` |
| `github.com/jackc/pgpassfile` | v1.0.0 | MIT | module cache `LICENSE` |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523-5a60cdf6a761 | MIT | module cache `LICENSE` |
| `github.com/jackc/puddle/v2` | v2.2.2 | MIT | module cache `LICENSE` |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 | BSD-3-Clause | module cache `LICENSE.md` |
| `github.com/aymerick/douceur` (indirect, via bluemonday) | v0.2.0 | MIT | module cache `LICENSE` |
| `github.com/gorilla/css` (indirect, via bluemonday) | v1.0.1 | BSD-3-Clause | module cache `LICENSE` |
| `github.com/yuin/goldmark` | v1.8.5 | MIT | module cache `LICENSE` |
| `golang.org/x/crypto` | v0.57.0 | BSD-3-Clause | module cache `LICENSE` |
| `golang.org/x/image` | v0.46.0 | BSD-3-Clause | module cache `LICENSE` |
| `golang.org/x/net` | v0.59.0 | BSD-3-Clause | module cache `LICENSE` |
| `golang.org/x/sync` | v0.23.0 | BSD-3-Clause | module cache `LICENSE` |
| `golang.org/x/sys` | v0.48.0 | BSD-3-Clause | module cache `LICENSE` |
| `golang.org/x/text` | v0.42.0 | BSD-3-Clause | module cache `LICENSE` |
| PostgreSQL server container image (`postgres@sha256:57c72fd2a128...`, pinned in `packaging/podman/compose.yml`) | PostgreSQL 16.14 on Alpine Linux 3.24.1 | PostgreSQL License (permissive), plus Alpine's own base-image component licenses | `docker inspect` / running the pinned image directly |
| Victory backend image's own base (`alpine:3.20`, `backend/Dockerfile`), plus `postgresql16-client` (apk, used for the Kernel 72 pre-migration `pg_dump` backup) | Alpine 3.20 | Alpine's own component licenses (mostly MIT/BSD; `busybox` is GPL-2.0) plus the PostgreSQL License for the client tools | `backend/Dockerfile` |
| Container runtime (host-provided, not pinned by this repo — `packaging/podman/compose.yml` targets either) | Docker Engine or Podman | Apache License 2.0 (both projects) | `packaging/podman/compose.yml` |

## Release inventory required

Before publication, inventory:

- Go, JavaScript, and other language manifests and lockfiles;
- container images, Dockerfiles, installers, and update-system components;
- fonts, icons, images, audio, video, textures, and maps;
- copied code, templates, snippets, and migrations;
- rulebooks, settings, venues, tutorials, demonstrations, and sample data; and
- all human and organizational contributors.

## Boundary rule

The Victory Community License applies only to material owned or relicensable by Grant A. Murray. A separately licensed file remains under its stated license. Nothing in the Victory license expands or restricts rights independently granted by another owner.
