# Kernel 96 — Security, Privacy Boundaries, Fresh-Install Trust & Internet Hardening Ledger

**Kernel spec:** `Construction/Kernels/Kernel 96 — Security, Privacy Boundaries, Fresh-Install Trust & Internet Hardening.md`
**Status:** PASS. Every substantive section of the spec (§4, §7-28, §35-36) has been audited with real evidence, not a checklist recitation. The great majority of Victory's existing authorization/session/upload/injection surface was already genuinely secure — server-side checks, not UI hiding — confirmed by reading the actual handler code for each. Real findings were fixed, not just logged; the ones intentionally deferred (CSP, HSTS, a full git-history scrub) are named explicitly with the reason, per §34's "fix or explicit justified deferral" rule for MEDIUM findings.
**Method:** Ten bounded, evidence-first adversarial audits (parallel forks, each reporting file:line citations and a SECURE/VULNERABLE/UNCLEAR verdict — never a generic security-checklist paragraph), covering the trust/default audit and every numbered attack category in the spec.

---

## 1. Checkpoint A — Trust/default audit (spec §4, §29, §39)

The architectural finding: `backend/migrations/002_seed_world.sql` created a location literally named/slugged `amurray-family` unconditionally, on every database, fresh installs included — and 16 further migrations (Producer's Office, Third Place, Storyboards, Courtyard, Writers Room, Socio equipment catalog, tutorial content, capacity guardrails, etc.) attached all of Victory's actual canonical content to that exact slug, and only that slug, regardless of what an Operator actually named their own lot at first-run. A second migration (`025_kernel42_neutral_install_location.sql`) additionally created a second, equally unconditional phantom `victory-theater` location. Root cause: these migrations ran before `access.EnsureDefaultLocation` (which creates the install's real location, under whatever slug its own Operator chose) ever executed, so they had no way to know what that location would be and fell back to names written when this was still single-tenant software.

**Fix:** `locations` gained an `is_default` column (at most one `true`, enforced by a partial unique index). Boot now runs in three phases — schema migrations (000-001) only, then `EnsureDefaultLocation` (marks the real default location `is_default = true`), then every remaining migration, rewritten to attach via `WHERE is_default` instead of a hardcoded slug. Migration 025 is neutralized to a no-op; migration 002 no longer creates a location at all, only attaches to the one `EnsureDefaultLocation` already made. This edits already-applied migrations, which the runner normally refuses (checksum mismatch on boot) — deliberate and safe only because every existing installation of this software was wiped and reinstalled fresh alongside this change (production, both Windows machines, and this machine's own local dev stack).

**Other Grant-specific defaults found and fixed in the same pass:**
- `docker-compose.yml` (the repo-root file, since retired) and `packaging/podman/compose.yml`'s spiritual predecessor defaulted `OPERATOR_HANDLE` to `straturli` when unset — any plain self-hosted install that didn't override it silently granted Grant's own operator authority to a stranger's instance.
- `backend/cmd/victory-recover/main.go` (the break-glass CLI) defaulted `--operator-handle`/`--base-url` to `straturli`/`https://victory.amurray.family` when their env vars weren't set.
- A venue literally named "Grant's Cabin" (`backend/internal/access/kernel16_venue_bootstrap.go`) seeded with `contact: "grant@amurray.family"` baked in — the venue itself stays (Grant's explicit call, 2026-09-11: it's a permanent feature, not a mistake), but the contact now reads "GM-Discovery on GitHub, gm_discovery on Discord." A new migration updates the already-seeded row on existing installs, since the Go source fix alone only affects future fresh installs.
- Two dev-only fixture CLIs (`k89fixture`, `k90fixture`) reference `amurray-family` — accepted as isolated test/dev tooling, not a production path.

**Repo hygiene:** a fully stale, pre-`windows-native` `packaging/windows/VictoryLauncher` tree plus its frozen CI workflow were found still sitting on `main` (dead since 2026-09-06, before the real Podman/native-process rewrite) and removed at Grant's direction. The repo-root `docker-compose.yml` — an explicitly-labeled "Grant's own production deployment," kept deliberately separate from the real K100 Podman file — was retired entirely; production, self-hosters, and local dev now all use `packaging/podman/compose.yml`, with `packaging/podman/compose.production.yml` as a thin override for production's one genuine infrastructure difference (the shared `bread-caddy` container on the external `edge_net` network).

---

## 2. Adversarial sweep — findings by category

**Object-level authorization (§11, §13; Characters/Shows/Scenes/Messages) — SECURE.** No direct-fetch-by-ID route exists for Characters at all; mutations check real ownership server-side. Shows/Scenes resolve authorization against the *specific* show's own location, not a global role check — a Show-A participant supplying Show B's IDs gets checked against Show B's real location and fails. Director-only data (`director_notes`) sits behind a real server-side gate, not a client-side hide. Messages are scoped by `to_user_id` in the SQL itself.

**Role escalation (§14) — SECURE.** Invite creation resolves the inviter's role server-side, never from client input; the target-role enum has no `producer`/`operator` path. Operator authority is provably only `OPERATOR_HANDLE`/`OPERATOR_USER_ID` env match or a real Producer membership row — no alternate "become operator" path found. Cross-show authority (`CanManageShowRun`) is parameterized by the actual show's own location, not "does this user hold Director anywhere."

**Session/WebSocket security (§15) — one real gap, found and fixed.** Sessions are DB-backed with real server-side expiry/revocation checked on every request; logout actually revokes, not just clears the cookie. WebSocket connections authenticate and authorize *before* completing the upgrade handshake. The periodic revalidation sweep (`Hub.RevalidateSessions`, added at Kernel 77 for exactly this class of bug) only rechecked whether the session was valid *at all* — never whether the user's role for the *specific venue* their socket was connected to still held. A Director removed from a Show's roster, or demoted, kept receiving that venue's Director-level broadcasts indefinitely on an already-open connection. **Fixed:** `Client` now records the `VenueSlug` it was authorized for at connect time; the revalidation sweep separately rechecks `access.UserCanAccessVenueSlug` per (user, venue) pair, disconnecting with a distinct `venue_access_revoked` notice. Verified against a real database with a new test proving the disconnect actually happens.

**Upload/asset security (§16-18) — SECURE.** Real magic-byte content sniffing (`http.DetectContentType`), not trusted headers or extensions; bounded file size and image dimensions; storage paths built from server-generated IDs only, immune to path traversal by construction; asset serving requires a real DB-backed authorization check before any byte is served; no SVG/active-content path exists at all. One LOW note: no explicit `X-Content-Type-Options` on the asset-serving handler specifically (the global one added this pass now covers it).

**Injection (§19-20) — SECURE.** Parameterized queries throughout; the only dynamic-SQL builders use hardcoded column-name fragments, never user-supplied ones. No user input reaches `os/exec` anywhere in the backend. The one markdown-to-HTML path goes through `bluemonday` sanitization; chat renders via `textContent`, not `innerHTML`. (Not exhaustively checked: the remaining ~15 frontend files using `innerHTML` beyond the two spot-checked, both of which were already safe via an `escapeHtml()` helper.)

**Secrets/config (§21) — SECURE, one historical leak already dead.** `change_this_now` was a real production Postgres password once committed to git — already found and rotated at Kernel 76, long before this pass; still sitting in immutable git history, but the credential itself is dead. No other real secret value was ever committed. `.gitignore` correctly excludes `.env` everywhere; confirmed none are tracked.

**Container/network boundaries (§22) — one real gap, found and fixed.** No `privileged: true`, no Docker-socket mount, Postgres never bound to a non-loopback address in any current compose file. Gap: the backend `Dockerfile` had no `USER` directive, running as root by default. **Fixed:** runs as a fixed non-root UID now; `generate-env.sh`'s created directories are made group/other-writable so any container UID can write the host-bind-mounted storage/backup/export paths regardless of exact host ownership. Verified end-to-end on a real rebuilt container, not assumed — confirmed the effective UID and confirmed an actual write to all three directories.

**CSRF/CORS/cookies/reverse-proxy (§23-25) — one real gap, found and fixed.** Session cookie is `HttpOnly`, `SameSite=Lax`, `Secure` by default — real CSRF protection, not just convention. No CORS anywhere (same-origin deployment, nothing to misconfigure). OAuth `state` is real, single-use, server-verified. Gap: `ratelimit.ClientIP` trusted `X-Forwarded-For` unconditionally — correct only because of current network topology (backend never directly reachable), not because the code verified it; a caller reaching the backend directly could spoof the header and bypass every IP-based rate limit. **Fixed:** now only honors it when the immediate connection is itself a private-range address — correctly covers both the self-hosted loopback-published topology and production's container-bridge (`edge_net`) topology, since a genuine internet attacker's own address can never be private-range in either case.

**Rate limits/error leakage (§26-27) — two real gaps, found and fixed.** Login/signup/password-reset/export/account-deletion are all rate-limited; most of the mutation surface shares a second, higher-frequency bucket. Gaps: `/api/invites` and `/api/invites/accept` had zero rate limiting — invite-accept specifically creates a new account, the exact category the spec calls out — **fixed**, now sharing the credential-guessing bucket. `internal/identity/requests.go` leaked raw internal error strings to clients on DB failures in 9 places, unlike the rest of the codebase's consistent bounded-error convention — **fixed**. Also found and fixed: a login timing side-channel (a nonexistent handle short-circuited before the deliberately-slow Argon2id comparison a real handle's wrong-password path always pays, a measurable latency tell for handle enumeration even though the response body is identical) — now runs the same comparison against a fixed dummy hash regardless.

**Backup/export security (§28) — SECURE.** Export request/download both resolve the account exclusively from the session — no target user ID or token accepted from the client at all, stronger than even an unguessable-token scheme. Real 48-hour expiry with actual cleanup. Export/backup directories are never web-servable. No HTTP-reachable restore endpoint exists anywhere; restore is confirmed CLI/host-only.

**Dependency/supply-chain (§35) — one HIGH CVE, fixed.** `govulncheck` found `pgx/v5@v5.7.2` carrying a real, code-reachable HIGH-severity SQL injection vulnerability (GO-2026-5004, dollar-quoted string literal confusion), reachable via `characters.ListOwnedCards`. **Fixed:** bumped to v5.9.2. Also bumped `golang.org/x/image` (webp-decode DoS CVEs, reachable via the upload path), `golang.org/x/net`/`golang.org/x/text` (indirect, one reachable through Postgres connection-string normalization). Frontend dependencies (Vue, PixiJS) are vendored locally, never CDN-loaded — no supply-chain exposure there. Postgres image remains digest-pinned.

**Security headers (§36) — real gap, partially fixed.** Exactly one header was set anywhere in the stack, on one handler. **Fixed:** `X-Content-Type-Options`, `Referrer-Policy`, and `X-Frame-Options` now apply globally — confirmed safe on every response regardless of content type or deployment mode. **Deliberately not done this pass:** Content-Security-Policy (the kernel's own doctrine warns against a generic bundle breaking Pixi/WebSockets/OAuth/assets — needs a real policy authored and tested against this app specifically) and Strict-Transport-Security (only correct once a deployment is definitely HTTPS-only, and this stack is also legitimately reachable over plain HTTP for pure-local access).

---

## 3. Verification

- Full backend test suite run against multiple genuinely fresh `victory_test` databases (dropped and recreated from scratch, `-p 1` to rule out cross-package races over shared fixture rows): down from 68 failures on the first fresh run after the migration-architecture change to 3, all confirmed unrelated to this kernel's work and routed to Kernel 98 as carried-in findings rather than fixed out of scope.
- `scripts/test/lib-migrate-and-bootstrap.sh` (the actual documented way to set up the test database) had the same pre-Kernel-96 bug `scripts/smoke/fresh-install.sh` had — manually pre-applying every migration before the real binary's own phased boot ever ran — and got the same fix.
- Two full, from-scratch real server rebuilds on this machine's own local dev stack (not just disposable test databases), each independently verified in the live database: correct `is_default` location, zero `amurray-family`/`victory-theater` reference anywhere, correct Grant's Cabin contact.
- Production (`victory.amurray.family`) was wiped and reinstalled fresh by Grant during this pass, verified live: new lot "Hope" (slug `hope`), `is_default = true`, zero legacy artifacts, reachable through the real `bread-caddy`/`edge_net` path.
- The non-root container fix was verified with an actual write to all three bind-mounted directories as the new UID, not assumed.
- The WebSocket per-venue revalidation fix has a real test against a live database proving the disconnect fires.

---

## 4. Residual/deferred items (not blockers, explicitly named)

- **Public release cleanup.** `victory-releases` is public with no auth required; every release before `v0.0.68` has the pre-fix `amurray-family`/`grant@amurray.family` strings baked into the compiled binary (Go's `//go:embed`). Real-world exposure is effectively zero so far (download counts confirm only Grant's own machines have touched it). Grant's plan: finish this kernel, cut a real `v0.1.1`, then prune the ~30 old releases under an honest "early scrubbing while learning to build a launcher" framing.
- **Git history.** The `victory` source repo is currently private (confirmed), so exposure is already narrow. Grant intends to eventually make it public under an AGPL license (confirmed no `LICENSE` file exists yet despite believing one had been added). A `git filter-repo` history scrub to remove personal-identity strings was discussed as sensible to do *before* going public, while no outside forks/clones exist yet to break — not yet scheduled.
- **CSP and HSTS** — deliberately deferred per §36 above, need dedicated testing, not a copy-pasted default.
- **Innerhtml audit** — not exhaustive across all frontend files; two representative samples were checked and both safe.
- **Three Kernel 98 carried-in findings** — a pre-existing dice-roll broadcast nil-pool panic, an eWrite skill-directory seed test that can't reproduce a new catalogue entry, and two scene-capture tests failing on layer state. None are security-relevant; all confirmed unrelated to this kernel's own changes; none investigated further, per this kernel's own boundary doctrine (§40).

---

## 5. Decision memory

- The Operator/handle/lot-name-at-first-run flow is the one correct source of identity for a fresh install — no code path may hardcode a fallback identity, including test infrastructure (which now gets its own explicit, isolated `amurray-family` convention via `dbtest.OpenTestPool`, not an accidental one).
- "Trusted proxy" for rate-limiting purposes means "private-range address," not "loopback specifically" — production's container-bridge topology and the self-hosted published-port topology are both legitimate and both need to work.
- Grant's Cabin keeps its name permanently; only its contact info was ever the problem.
- CSP/HSTS are real, known gaps, deliberately not rushed.

---

## 6. What Grant should still confirm or decide

1. When to schedule the `victory-releases` pruning and the `v0.1.1` public cut.
2. Whether/when to do the `git filter-repo` history scrub before going public, and whether to add the AGPL `LICENSE` file at the same time.
3. Whether the three Kernel 98 carried-in findings warrant prioritizing ahead of other K98 work, given none are security-relevant.
