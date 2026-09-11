# Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates Ledger

**Kernel spec:** `Construction/Kernels/Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates.md`
**Status:** PASS for alpha. Core loop (install → run → update → invite → delete) is real, working on real hardware across multiple physical machines, and every spec-required item this kernel can actually finish without Grant's own separate action has been built. Three items remain, and none of them are code: obtaining a code-signing certificate, and two live proofs (genuinely-external-network remote access, A→B upgrade with data verified intact) that need Grant's hands, not another engineering pass. See §7.
**Branches:** `windows-native` (launcher/installer work) merges from `main` (core Victory) rather than the reverse, so core fixes land once and flow into the Windows build; `main` never carries launcher-only code.
**Correction note (2026-09-11):** this ledger's first draft wrongly listed host-prerequisite checks and third-party notices as missing. Both already existed (`HostPrerequisites.cs`, `THIRD_PARTY_NOTICES.md`) — found on a second, closer pass before closing the kernel out. Corrected below rather than left standing.

---

## 1. Architecture actually in place (spec Checkpoint A)

- **Installer framework:** Velopack. `packaging/windows/VictoryLauncher` is a .NET 8 WinForms tray app (`VictoryLauncher.exe`), self-contained `win-x64` publish, packed with `vpk` into a Setup.exe and an auto-update feed. Releases are hosted in a separate repo (`GM-Discovery/victory-releases`), not this repo — `.github/workflows/windows-installer.yml` builds on every push to `main`/`windows-native` touching `packaging/windows/**`, `backend/**`, or `frontend/**`, and publishes via `vpk upload github`.
- **Runtime strategy: no Docker/container runtime on Windows at all.** `RuntimeManager.cs` spawns Postgres, the Victory backend, Caddy, and (when remote access is on) `cloudflared` as plain native Windows child processes it manages directly — `docker`/`podman` only appear in this codebase for the separate Linux/dev deployment path (`packaging/podman/compose.yml`), never in the Windows one. This resolves spec §2/§3 cleanly: no Docker Desktop dependency, no licensing question to chase.
- **Versioning:** placeholder `0.0.<CI run number>`, monotonically increasing, good enough for Velopack's own update-detection but explicitly not a real semantic version yet (noted in-workflow as a thing to revisit once signing exists).
- **Install/data separation (spec §6):** application files live entirely inside Velopack's own managed directory (replaced wholesale on update, removed on uninstall). Everything persistent — Postgres data, generated `.env`/secrets, storage, backups, exports, logs — lives under `%LocalAppData%\Victory`, which Velopack's uninstaller never touches. This was a deliberate windows-native decision to move off the earlier Podman-era `%ProgramData%\Victory` choice specifically so nothing in this app ever needs to run elevated.

---

## 2. What's built and proven on real hardware

- **Automatic updates work.** This took the majority of one very long session and roughly 60 failed real-hardware attempts before the actual root cause was found (by Grant, from evidence surfaced in Velopack's own log, not guessed by the agent): Postgres and the Cloudflare tunnel were never stopped before `ApplyUpdatesAndRestart`, and because neither process had an explicit `WorkingDirectory` set, both inherited the launcher's own `current\` directory as their working directory — holding an OS-level lock that silently blocked Velopack's rename-and-swap every time. Fixed in `UpdateChecker.ApplyPendingUpdateAsync` (stops Postgres via `pg_ctl -w` and the tunnel before applying). Confirmed working starting v0.0.60, reconfirmed through v0.0.66.
- **Update timing respects live Shows (spec §33).** A pending update is held, not applied, while any session has `status != 'closed'`; the Operator is told plainly (after a corrected round of guidance) to end the show via `/showtime <code> end` rather than a wrong first guess pointing at Director's Chair's "Close Showing" (a different mechanism entirely — verified in `backend/internal/showtime/showtime.go` that only the `/showtime end` path sets `sessions.status = 'closed'`). Every check outcome — including a silent block — now writes to `launcher.log` unconditionally, not just via balloon notification, since Windows notification settings (Focus Assist etc.) can suppress the balloon with zero other record.
- **Remote access is a secure shareable URL, no manual TLS/ports/router config (spec §17–§22).** Cloudflare Quick Tunnel by default (no account needed, fresh `*.trycloudflare.com` address each start) or a Named Tunnel if the Operator already has their own Cloudflare setup; both terminate HTTPS before traffic ever reaches Victory. The backend binds loopback-only (`127.0.0.1`), which also means Windows Firewall never prompts for it — Caddy is the only thing that talks to it, and Caddy itself is loopback-scoped too. Confirmed on real hardware.
- **Invite/share links resolve correctly across machines (spec §22), after two real bugs found and fixed:**
  1. Links were built from `window.location.origin`, always `localhost` for the Operator generating them — useless to a remote recipient. Fixed via `GET /api/system/public-url`, which the backend answers from a `PUBLIC_URL_FILE` the launcher keeps current (both tunnel modes write to it now; only Quick Tunnel did before).
  2. Accepting a valid, correctly-resolved invite could still 400 with an opaque `user_create_failed` if the email/handle already existed on that install (e.g. inviting yourself into a lot where you already have an Operator account). Backend now distinguishes `email_taken`/`handle_taken`; the invite page offers a "Sign in" link that round-trips back to redeem the invite via `return_to` (which also required fixing `login/index.html`'s password-login path — it ignored `return_to` entirely before this, only the Discord-login path honored it).
- **Support bundle exists** (`SupportBundle.cs`) — one button, one zip to the Desktop, built specifically because manually walking a tester through `%LocalAppData%\Victory\logs` file-by-file during the update-retry investigation was real, confirmed pain.
- **Status surface (spec §26)** shows Victory running / remote access state / public URL / update status in human terms, not container internals.
- **Backup-before-migrate has a real fix in place** (unexercised — see §5): the backend shells out to a bare `pg_dump` before running a schema-changing migration, which would have failed on any populated database because `pg_dump.exe` only exists inside the bundled Postgres install, never on system PATH. `RuntimeManager.StartBackendAsync` now prepends `AppPaths.PostgresBinDir` to the backend's own `PATH` env var so that resolves.
- **Host prerequisite checks exist (spec §5).** `HostPrerequisites.Check()` runs before the first-run wizard ever opens: rejects anything below Windows 10 1809 (build 17763 — .NET 8's own self-contained win-x64 support floor) and anything with under 2 GB free disk on the data drive, both with a plain message instead of failing obscurely mid-Postgres-download. A 64-bit check is redundant on top of this — a self-contained win-x64 publish simply cannot launch at all on a 32-bit OS, so there's no later point where that would need catching. Not checked: available RAM — realistically never the limiting factor for Victory's footprint on anything meeting the Windows-version floor, left out rather than added for its own sake.
- **Third-party notices are real and already shipped (spec §4/§46).** `packaging/windows/VictoryLauncher/THIRD_PARTY_NOTICES.md` — hand-maintained, version-pinned (PostgreSQL 16.15 EDB Windows binaries, Caddy 2.11.4, cloudflared 2026.8.3, Velopack, plus every .NET/NuGet and Go dependency) — is wired into `VictoryLauncher.csproj` via `CopyToOutputDirectory`, so it ships alongside `VictoryLauncher.exe` in every real install, not just sitting in the repo. `Construction/vendor-acknowledgements.md` (the repo-level, pre-K100 file) now points at it as the authoritative copy for the Windows bundle instead of duplicating it.
- **Explicit full-data-deletion option now exists (spec §8).** Routine uninstall already preserved data by construction (Velopack's uninstaller only ever touches its own app-files directory, never `%LocalAppData%\Victory`) — that half of §8 was already satisfied. The separate, not-preselected "permanently delete this lot" path didn't exist; now does, in Status ("Delete Victory Data..."). Two confirmations — a Yes/No warning defaulting to No, then typing the lot's own name back — before it stops the runtime and deletes `AppPaths.DataRoot`. Discussed with Grant 2026-09-11 first: not required for any correctness/legal reason (the data was always under the user's own OS-level access, nothing is held remotely), real value is doing a full wipe *safely* rather than a manual delete racing a process that still holds a file lock. Built since it was cheap; not yet tested by Grant, and doesn't need to be before this kernel closes.

---

## 3. Repo hygiene found and fixed along the way

- **`main` carried a fully stale, pre-`windows-native` `packaging/windows/VictoryLauncher` tree** (missing `PostgresBinDir` and everything else built since, still wired to an abandoned `podman/compose.yml`) plus a frozen CI workflow for it, both untouched since 2026-09-06 — dead weight left behind when `windows-native` took over as the real implementation. Confirmed inert (that workflow's path filters never matched anything pushed to `main` since, so it wasn't silently failing CI) before removing both, at Grant's direction.

---

## 4. Decisions recorded (not to be re-asked)

- No Docker/container runtime on Windows — native child processes only, managed directly by the launcher.
- Data root is `%LocalAppData%\Victory`, not `%ProgramData%\Victory` — nothing in this app runs elevated.
- Remote access defaults to Cloudflare Quick Tunnel (no account required); a Named Tunnel is supported for an Operator who already has one, using the same `PUBLIC_URL_FILE` mechanism either way.
- Backend binds loopback-only on Windows — deliberate, avoids a Windows Firewall prompt entirely rather than requesting and narrowing a rule.
- Version scheme is a placeholder (`0.0.<CI run number>`) pending real signing/release infrastructure.
- RAM is deliberately not part of `HostPrerequisites.Check()` — not a realistic limiting factor for this workload.

---

## 5. Needs verification, not more code

- **Backup-before-migrate** (§2) has never actually been exercised by a real schema-changing migration against a populated database — the fix addresses a documented failure mode but hasn't been proven end-to-end. Will be proven the next time a real migration ships in an update; nothing further to build now.
- **Uninstall → reinstall → recover existing lot (spec §54)** has not been tested.
- **Invite accept-flow fix** (the `email_taken`/`handle_taken` + sign-in round-trip) hasn't been explicitly re-confirmed end-to-end by Grant since it shipped.
- **Delete Victory Data** (§2) is new and untested — built correctly per code review, but no real click-through yet.

None of these block closing the kernel; they'll surface naturally the next time each path is actually used.

---

## 6. Kernel 100 Reportback (spec §58)

1. **Installer framework:** Velopack (`vpk`), packing a self-contained .NET 8 win-x64 publish into a Setup.exe + update feed hosted on `GM-Discovery/victory-releases`.
2. **Runtime/container strategy:** none — Postgres, backend, Caddy, cloudflared run as native Windows child processes the launcher supervises directly. Docker/Podman exist only on the separate Linux dev-deployment path, never touched on Windows.
3. **Docker/Desktop licensing decision:** moot — Docker Desktop was never adopted for this distribution path, so no licensing question applies.
4. **Third-party notice changes:** `packaging/windows/VictoryLauncher/THIRD_PARTY_NOTICES.md` (ships with the app) now covers every Windows-bundled component with pinned versions; `Construction/vendor-acknowledgements.md` updated to point at it as canonical rather than duplicate it.
5. **Windows prerequisites handled:** Windows 10 1809+ and 2 GB free disk, checked before first-run setup begins (`HostPrerequisites.cs`). RAM not checked (see §4).
6. **Install locations:** app files under Velopack's own managed directory (replaced on update, removed on uninstall); all persistent state under `%LocalAppData%\Victory` (data, backups, exports, logs, generated `.env`/secrets), untouched by both.
7. **Generated configuration/secrets:** `.env` generated and owned entirely by the installer (`EnvGenerator.cs`), never hand-edited, never exposed in ordinary logs.
8. **Database bootstrap:** installer downloads/initializes Postgres on first run, creates the database, runs migrations, verifies health — no Grant-specific data required for a fresh lot.
9. **Operator bootstrap:** first-run wizard creates a real Operator (handle/password) for that machine specifically, no fixed handle, no Grant-specific fallback.
10. **Lot bootstrap:** first-run wizard asks for a human lot name; the original Murray-family lot is not a seed for new installs.
11. **No-Discord proof:** Victory boots and runs fully without Discord configured; Discord setup is never part of first-run.
12. **Remote-access architecture comparison:** not re-litigated this pass — Cloudflare Tunnel (relay/tunnel model, spec §19.C) was already the standing decision going into this work; nothing surfaced that calls it into question.
13. **Selected remote-access implementation:** Cloudflare Quick Tunnel by default (no account), Named Tunnel supported for an Operator with their own Cloudflare setup.
14. **Secure URL behavior:** fresh `https://*.trycloudflare.com` per Quick Tunnel start, or the Operator's own stable hostname for a Named Tunnel; both written to a single file (`PUBLIC_URL_FILE`) the backend reads fresh per-request so invite links always resolve to the real reachable address, not `localhost`.
15. **TLS handling:** terminated by Cloudflare before traffic reaches Victory; the Operator never touches a certificate.
16. **Firewall/router behavior:** no router configuration needed (tunnel is outbound-only); no Windows Firewall prompt at all, since the backend and Caddy both bind loopback-only.
17. **Local fallback:** local access (`http://localhost:<port>`) always works regardless of tunnel state.
18. **Central-service metadata:** none beyond what Cloudflare's own tunnel service necessarily sees to route traffic; no Victory-operated broker exists.
19. **Remote-service outage behavior:** if the tunnel is down, local Victory keeps working; Status reflects tunnel state plainly rather than masking it.
20. **Desktop/start behavior:** tray icon with Open/Status/Check for Updates/Start with Windows/Quit; double-click opens the Operator UI; never spawns a duplicate stack.
21. **Updater architecture:** Velopack, checked automatically (4-hour cadence, 15-minute cadence once a backend-changing update is downloaded and waiting for the quiet window) and manually via "Check for Updates."
22. **Update trust/signing:** updates are fetched over HTTPS from Victory's own GitHub Releases feed via Velopack's `GithubSource`; artifact code-signing itself is not yet in place (see §7).
23. **Update timing around Shows:** backend-changing updates wait for either a quiet 2:30am-local window or all sessions being closed; frontend-only updates apply immediately since they cause no outage.
24. **A→B upgrade proof:** updates apply successfully and repeatedly on real hardware (v0.0.42→43 through v0.0.66); representative-data survival across an update specifically has not been separately verified (see §7).
25. **Uninstall/reinstall proof:** not yet performed (see §5).
26. **Clean-machine proof:** performed — a third physical machine with no prior Victory install reached a working state via the installer alone, confirmed by Grant directly.
27. **External-network proof:** partially performed — two separate physical machines, a real Cloudflare tunnel, genuine cross-machine invite flow all confirmed working; not yet confirmed over a truly separate household/ISP connection (see §7).
28. **Installer/code-signing state:** unsigned. Recorded plainly in `.github/workflows/windows-installer.yml`'s own header. SmartScreen will warn on every install/update until a certificate exists.
29. **Support-bundle/logging behavior:** one-click support bundle to the Desktop (`SupportBundle.cs`); `launcher.log` records every update-check outcome unconditionally, not just via balloon notification.
30. **Items routed to K96:** none newly surfaced this pass beyond what K100 already handles itself (loopback-only binding, generated secrets, no exposed database port).
31. **Items routed to K99:** the first-time Operator guide seam (spec §15/§49) still just points at "Set up your Victory lot" without the fuller guide content — unchanged this pass, remains K99's to build.
32. **Items routed to K101 (or later):** RAM prerequisite check if it's ever actually needed; a real semantic version scheme once signing exists; PNG/broader export-adjacent work is unrelated (K94, not K100).
33. **Grant consumer-experience acceptance:** given directly, 2026-09-11 — "I have tested the new install to my sufficient liking," accepting the install/update/invite loop as-is, and explicitly asked to close this kernel out.

**Final status: PASS for alpha.** Three items remain outside what more agent work can resolve — obtaining a code-signing certificate, a genuinely-external-network remote-access proof, and an explicit A→B data-integrity upgrade proof — all requiring Grant's own action or hands-on testing rather than more code. Recorded as open follow-ups, not blockers to closing this kernel.

---

## 7. What Grant should still confirm or decide

1. Whether to pursue a real code-signing certificate now, or continue accepting SmartScreen warnings for the rest of alpha.
2. Whether a genuinely-external (different household/ISP) remote-friend test is worth arranging at some point — not required to close this kernel, just not yet proven.
3. Whether to specifically verify data survives an A→B update at some point (create a Character/Show, update, confirm it's still there) — the update mechanism itself is proven solid; this is about the migration path specifically, and will get real-world proof the next time a schema-changing release actually ships.
