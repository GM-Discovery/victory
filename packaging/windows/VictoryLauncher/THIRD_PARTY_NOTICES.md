# Third-Party Notices

Victory bundles or downloads the following third-party software. This file
is shipped alongside VictoryLauncher.exe in every install.

Kept by hand, not generated -- re-check this list whenever a dependency in
`packaging/windows/VictoryLauncher/VictoryLauncher.csproj` or
`backend/go.mod` changes.

## Runtimes downloaded on first run (not compiled in)

| Component | Version pinned | License |
|---|---|---|
| PostgreSQL | 16.15 (EDB Windows binaries) | The PostgreSQL License (permissive, MIT-style) |
| Caddy | 2.11.4 | Apache License 2.0 |
| cloudflared | 2026.8.3 (Quick Tunnel / named tunnel mode only, opt-in) | Apache License 2.0 |

## .NET packages (VictoryLauncher.csproj)

| Package | License |
|---|---|
| Velopack | MIT License |
| Microsoft.Win32.Registry | MIT License |
| System.Security.AccessControl | MIT License |
| System.IO.FileSystem.AccessControl | MIT License |

## Go modules (backend/go.mod)

| Module | License |
|---|---|
| github.com/gorilla/websocket | BSD 2-Clause License |
| github.com/jackc/pgx/v5 | MIT License |
| github.com/microcosm-cc/bluemonday | BSD 3-Clause License |
| github.com/yuin/goldmark | MIT License |
| golang.org/x/crypto | BSD 3-Clause License |
| golang.org/x/image | BSD 3-Clause License |
| golang.org/x/sys | BSD 3-Clause License |
| github.com/aymerick/douceur (indirect) | MIT License |
| github.com/gorilla/css (indirect) | BSD 2-Clause License |
| github.com/jackc/pgpassfile (indirect) | MIT License |
| github.com/jackc/pgservicefile (indirect) | MIT License |
| github.com/jackc/puddle/v2 (indirect) | MIT License |
| golang.org/x/net (indirect) | BSD 3-Clause License |
| golang.org/x/sync (indirect) | BSD 3-Clause License |
| golang.org/x/text (indirect) | BSD 3-Clause License |

## Frontend (vendored under frontend/lib/, served as static files)

| Library | License |
|---|---|
| Vue.js (vue.esm-browser.prod.js) | MIT License |
| Pixi.js (pixi.min.js) | MIT License |

---

None of the above are modified from their upstream releases. Full license
texts are available from each project's own repository at the version
pinned above; this file records which license applies to which component
rather than reproducing every text in full.
