# Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates Ledger

**Kernel spec:** `Construction/Kernels/Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates.md`
**Status:** PARTIAL. Core loop (install → run → update → invite) is real and working on real hardware across multiple physical machines. Several spec-required items (code signing, third-party notices for the Windows-bundled binaries, host prerequisite checks) are not yet built. Formal Checkpoint reports (spec §56) and this ledger did not exist until now — everything below is reconstructed from the actual codebase and this project's own commit/session history, not re-derived from memory.
**Branches:** `windows-native` (launcher/installer work) merges from `main` (core Victory) rather than the reverse, so core fixes land once and flow into the Windows build; `main` never carries launcher-only code.

---

## 1. Architecture actually in place (spec Checkpoint A)

- **Installer framework:** Velopack. `packaging/windows/VictoryLauncher` is a .NET 8 WinForms tray app (`VictoryLauncher.exe`), self-contained `win-x64` publish, packed with `vpk` into a Setup.exe and an auto-update feed. Releases are hosted in a separate repo (`GM-Discovery/victory-releases`), not this repo — `.github/workflows/windows-installer.yml` builds on every push to `main`/`windows-native` touching `packaging/windows/**`, `backend/**`, or `frontend/**`, and publishes via `vpk upload github`.
- **Runtime strategy: no Docker/container runtime on Windows at all.** `RuntimeManager.cs` spawns Postgres, the Victory backend, Caddy, and (when remote access is on) `cloudflared` as plain native Windows child processes it manages directly — `docker`/`podman` only appear in this codebase for the separate Linux/dev deployment path (`packaging/podman/compose.yml`), never in the Windows one. This resolves spec §2/§3 cleanly: no Docker Desktop dependency, no licensing question to chase.
- **Versioning:** placeholder `0.0.<CI run number>`, monotonically increasing, good enough for Velopack's own update-detection but explicitly not a real semantic version yet (noted in-workflow as a thing to revisit once signing exists).
- **Install/data separation (spec §6):** application files live entirely inside Velopack's own managed directory (replaced wholesale on update, removed on uninstall). Everything persistent — Postgres data, generated `.env`/secrets, storage, backups, exports, logs — lives under `%LocalAppData%\Victory`, which Velopack's uninstaller never touches. This was a deliberate windows-native decision to move off the earlier Podman-era `%ProgramData%\Victory` choice specifically so nothing in this app ever needs to run elevated.

---

## 2. What's built and proven on real hardware

- **Automatic updates work.** This took the majority of tonight's session and roughly 60 failed real-hardware attempts before the actual root cause was found (by Grant, from evidence surfaced in Velopack's own log, not guessed by the agent): Postgres and the Cloudflare tunnel were never stopped before `ApplyUpdatesAndRestart`, and because neither process had an explicit `WorkingDirectory` set, both inherited the launcher's own `current\` directory as their working directory — holding an OS-level lock that silently blocked Velopack's rename-and-swap every time. Fixed in `UpdateChecker.ApplyPendingUpdateAsync` (stops Postgres via `pg_ctl -w` and the tunnel before applying). Confirmed working starting v0.0.60, reconfirmed through tonight's v0.0.66.
- **Update timing respects live Shows (spec §33).** A pending update is held, not applied, while any session has `status != 'closed'`; the Operator is told plainly (after a corrected round of guidance) to end the show via `/showtime <code> end` rather than a wrong first guess pointing at Director's Chair's "Close Showing" (a different mechanism entirely — verified in `backend/internal/showtime/showtime.go` that only the `/showtime end` path sets `sessions.status = 'closed'`). Every check outcome — including a silent block — now writes to `launcher.log` unconditionally, not just via balloon notification, since Windows notification settings (Focus Assist etc.) can suppress the balloon with zero other record.
- **Remote access is a secure shareable URL, no manual TLS/ports/router config (spec §17–§22).** Cloudflare Quick Tunnel by default (no account needed, fresh `*.trycloudflare.com` address each start) or a Named Tunnel if the Operator already has their own Cloudflare setup; both terminate HTTPS before traffic ever reaches Victory. The backend binds loopback-only (`127.0.0.1`), which also means Windows Firewall never prompts for it — Caddy is the only thing that talks to it, and Caddy itself is loopback-scoped too. Confirmed on real hardware.
- **Invite/share links resolve correctly across machines (spec §22), after two real bugs found and fixed this session:**
  1. Links were built from `window.location.origin`, always `localhost` for the Operator generating them — useless to a remote recipient. Fixed via `GET /api/system/public-url`, which the backend answers from a `PUBLIC_URL_FILE` the launcher keeps current (both tunnel modes write to it now; only Quick Tunnel did before).
  2. Accepting a valid, correctly-resolved invite could still 400 with an opaque `user_create_failed` if the email/handle already existed on that install (e.g. inviting yourself into a lot where you already have an Operator account). Backend now distinguishes `email_taken`/`handle_taken`; the invite page offers a "Sign in" link that round-trips back to redeem the invite via `return_to` (which also required fixing `login/index.html`'s password-login path — it ignored `return_to` entirely before tonight, only the Discord-login path honored it).
- **Support bundle exists** (`SupportBundle.cs`) — one button, one zip to the Desktop, built specifically because manually walking a tester through `%LocalAppData%\Victory\logs` file-by-file during the update-retry investigation was real, confirmed pain.
- **Status surface (spec §26)** shows Victory running / remote access state / public URL / update status in human terms, not container internals.
- **Backup-before-migrate has a real fix in place** (untested — see §4): the backend shells out to a bare `pg_dump` before running a schema-changing migration, which would have failed on any populated database because `pg_dump.exe` only exists inside the bundled Postgres install, never on system PATH. `RuntimeManager.StartBackendAsync` now prepends `AppPaths.PostgresBinDir` to the backend's own `PATH` env var so that resolves.

---

## 3. Real gaps — not built yet

- **Code signing (spec §45).** Confirmed unsigned — `.github/workflows/windows-installer.yml` says so directly ("Not yet signing anything — that's still pending a real certificate"). Acceptable for alpha per spec, but it's the explicit blocker before public release, and every install/update currently shows a SmartScreen warning as a result. Needs Grant to actually obtain a certificate; nothing to build in code until then.
- **Third-party notices are stale for the Windows-specific bundle (spec §4/§46).** `Construction/vendor-acknowledgements.md` hasn't been touched since May 2026 (Kernel 33 era) and does not mention `cloudflared` or Velopack — the two binaries this kernel newly bundles. Needs a real audit pass: component, version, license, copyright, redistribution obligation, whether a vendor EULA needs separate user acceptance.
- **No host prerequisite checks (spec §5).** Nothing in `packaging/windows/VictoryLauncher` checks 64-bit architecture, available RAM, available disk, or Windows version before proceeding. An unsupported machine currently just fails somewhere downstream instead of getting a clear message up front.
- **No explicit "permanently delete this Victory lot and all local data" uninstall option (spec §8).** Routine uninstall correctly preserves data by construction (Velopack's uninstaller only touches its own app-files directory, never `%LocalAppData%\Victory`) — that half of §8 is satisfied. The separate, unmistakable opt-in full-deletion path is not built. Discussed with Grant 2026-09-11: not required for any correctness/legal reason (the data is already under the user's own OS-level access, nothing is held remotely), real value would only be doing a full wipe *safely* (stopping Postgres/backend/tunnel first, so a running process doesn't leave a partial/corrupted delete behind) rather than raw convenience. Deferred — Grant's call whether to schedule it.

---

## 4. Needs verification, not more code

- **Backup-before-migrate**, described in §2 above, has never actually been exercised by a real schema-changing migration against a populated database this session — the fix addresses a documented failure mode but hasn't been proven end-to-end.
- **Uninstall → reinstall → recover existing lot (spec §54)** has not been tested.
- **A→B upgrade proof with representative data verified intact (spec §53)** — updates have been proven to *apply* successfully and repeatedly on real hardware, but no pass this session specifically verified that Characters/Shows/Storyboards/etc. created before an update survive it, as opposed to just confirming the app relaunches on the new version.
- **Genuinely external network proof (spec §51).** Tonight's cross-machine invite testing (Test lot ↔ Hope lot) exercised two different physical machines and a real Cloudflare tunnel, but both are presumably on Grant's own household network. The spec wants a remote friend on a separate internet connection entirely — not yet done.
- **Invite accept-flow fix itself** (the `email_taken`/`handle_taken` + sign-in round-trip shipped this session) has not yet been explicitly re-confirmed working end-to-end by Grant since it shipped.

---

## 5. Decisions recorded (not to be re-asked)

- No Docker/container runtime on Windows — native child processes only, managed directly by the launcher.
- Data root is `%LocalAppData%\Victory`, not `%ProgramData%\Victory` — nothing in this app runs elevated.
- Remote access defaults to Cloudflare Quick Tunnel (no account required); a Named Tunnel is supported for an Operator who already has one, using the same `PUBLIC_URL_FILE` mechanism either way.
- Backend binds loopback-only on Windows — deliberate, avoids a Windows Firewall prompt entirely rather than requesting and narrowing a rule.
- Version scheme is a placeholder (`0.0.<CI run number>`) pending real signing/release infrastructure.
- An explicit full-data-deletion uninstall option is not required and is deferred (see §3).

---

## 6. What Grant should still confirm or decide

1. Whether to pursue a real code-signing certificate now, or continue accepting SmartScreen warnings through the rest of alpha.
2. Whether the deferred "permanently delete this lot" uninstall option should be scheduled, or left out.
3. Whether an actual genuinely-external (different household/ISP) remote-friend test is worth arranging before calling §51 proven.
4. Confirm the invite accept-flow fix (email/handle-taken messaging + sign-in round-trip) works as intended on a real repeat-invite attempt.
