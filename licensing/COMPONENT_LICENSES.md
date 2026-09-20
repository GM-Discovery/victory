# Victory Component and Content Licenses

This document separates the licenses governing the application, game rules, settings, art, and third-party software. It must be completed from the actual release repository before publication.

## First-party and bundled components

| Component | Owner | Proposed or existing license | Approval/status |
| --- | --- | --- | --- |
| Original Victory application code | Grant A. Murray | Victory Community License 1.0 or applicable commercial terms | Owner confirmed |
| Victory names, logos, and product identity | Grant A. Murray | Victory Trademark and Official Build Policy 1.0 | Owner confirmed; confirm mark and logo inventory |
| Socio Core rules package | Grant A. Murray | CC BY 4.0 | Owner and existing license confirmed; verify attribution files |
| Niava setting material | Grant A. Murray | CC BY-ND 4.0 | Owner and existing license confirmed; keep separate from adaptable core rules |
| Catharsis Theater package | Grant A. Murray | Victory Community License 1.0 | Owner and proposed license confirmed; required in branded builds |
| Victory documentation | Grant A. Murray | Victory Community License 1.0 | Owner and proposed license confirmed |
| Original art, venue icons, maps, audio, video, and fonts | Various | Per-asset licenses | Complete asset manifest required |

## Required official packages versus license ownership

The requirement that Victory-branded builds include Socio and Catharsis Theater does not merge those packages into the software license. Each package retains its stated content license.

Standard commercial Victory permission covers operation of the bundled product only to the extent Grant A. Murray owns or may license each component and has authorized Discovery Games Interactive LLC to sublicense it. It does not override a third party’s terms.

## Currently documented third-party software

The historical vendor acknowledgement identifies the following components. Exact versions and current licenses must be verified against the public-release repository:

| Component | Expected license | Verify from |
| --- | --- | --- |
| Go toolchain/runtime | BSD-style Go license | Current toolchain distribution |
| PostgreSQL Alpine image | PostgreSQL license plus image-package notices | Current container image and packages |
| Docker and Docker Compose | Applicable vendor and component terms | Actual distribution method |
| `github.com/jackc/pgx/v5` | MIT | Current `go.mod`, `go.sum`, and upstream license |
| `golang.org/x/crypto` | BSD-style | Current `go.mod`, `go.sum`, and upstream license |
| `golang.org/x/sys` | BSD-style | Current `go.mod`, `go.sum`, and upstream license |
| PostgreSQL `pgcrypto` | PostgreSQL distribution terms | Shipped PostgreSQL version |

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
