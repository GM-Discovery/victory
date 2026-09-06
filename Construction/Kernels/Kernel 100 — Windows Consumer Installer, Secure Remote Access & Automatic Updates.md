# Kernel 100 — Windows Consumer Installer, Secure Remote Access & Automatic Updates

**Status:** DRAFT — ready for implementation  
**Type:** Consumer distribution + Windows installer + first-run bootstrap + remote connectivity + updater  
**Sequence position:** May be executed immediately after Kernel 95; does not require Kernels 96–99 first  
**Primary proof:** A normal Windows user can download Victory, double-click one installer, make only the few decisions Victory genuinely needs, and finish with a working self-hosted Victory lot that remote friends can reach securely  
**Core doctrine:** Installing Victory should feel like installing software, not administering a web stack.

---

## 0. Kernel mode

Kernel 100 turns Victory from developer-deployed software into an installable Windows product.

The user should not need to understand Go, Postgres, Docker, container networking, `.env`, migrations, TLS certificates, reverse proxies, ports, OAuth, database initialization, secret generation, service registration, or shell commands.

The installer owns those details. Human input is requested only when Victory needs a real product decision.

---

## 1. Primary Windows experience

Target flow:

> **Download → double-click → installer takes over when decisions are needed → Victory works.**

The installer may take clear foreground focus for required setup decisions, but should not force the user to babysit routine operations.

Preferred sequence:

1. launch installer;
2. dependency/system check;
3. automatic runtime installation/configuration;
4. Victory installation;
5. data/runtime initialization;
6. minimal Operator setup;
7. secure remote-access setup;
8. launch Victory;
9. show the Operator working local and remote access;
10. enable/configure automatic updates.

No terminal. No manual `.env`. No manual database commands. No separate prerequisite installation before starting Victory’s installer.

---

## 2. Consumer-grade prerequisite rule

Victory may use Docker/container technology internally.

The user should not have to install it first.

If Victory requires a container runtime:

- detect whether an acceptable runtime already exists;
- install/configure one when absent;
- handle required Windows components such as WSL2/virtualization where feasible;
- explain only genuinely blocking host requirements;
- resume installation after reboot where practical.

The user may see that a runtime is being installed. They should not have to understand or operate it.

---

## 3. Docker is an implementation choice, not the product

Do not hard-wire the consumer contract to Docker Desktop before checking:

- redistribution rights;
- license acceptance requirements;
- commercial-use restrictions;
- silent-install rules;
- update obligations;
- third-party notices;
- whether another open-source Windows-compatible runtime better fits Victory distribution.

If Docker Desktop is used, Victory must respect Docker’s licensing and user-acceptance requirements.

If bundling Docker Desktop would create undesirable licensing, account, or user-interruption requirements, prefer a legally distributable runtime architecture instead of requiring users to install Docker manually.

The product requirement is:

> **Victory manages its runtime.**

Not:

> **Victory must use Docker Desktop.**

---

## 4. Third-party software acknowledgements

Victory must maintain clear third-party notices for distributed dependencies.

Audit and update the existing vendor acknowledgements / third-party notices.

For every bundled redistributable component, record as applicable:

- component;
- version;
- license;
- copyright notice;
- source/license link;
- redistribution obligations;
- source-offer/source-distribution obligation if any;
- whether the user must separately accept vendor terms.

The installer should provide access to applicable notices without turning installation into a wall of legal text.

Do not silently accept another vendor’s EULA on the user’s behalf.

---

## 5. Windows support target

Target current supported consumer Windows, with primary focus on Windows 11 and Windows 10 only if current dependencies and support realities make it responsible.

Detect unsupported configurations clearly.

Check where relevant:

- 64-bit architecture;
- virtualization capability;
- WSL2/container prerequisites;
- sufficient disk;
- sufficient RAM;
- required Windows features;
- firewall capability;
- available ports.

Do not bury an unsupported-host failure inside container logs.

---

## 6. Installation locations

Use ordinary Windows conventions.

Separate:

### Application files
Replaceable by upgrade/reinstall.

### User/Victory data
Persistent and protected from routine upgrades/uninstall.

### Generated configuration/secrets
Persistent and appropriately permissioned.

### Logs
Bounded and discoverable.

Do not store irreplaceable user data under an application directory that uninstall may remove automatically.

---

## 7. Victory data must survive upgrades

An installer update must never casually delete Characters, Shows, Showings, Scenes, Storyboards, messages, relationships, assets, configuration, Operator identity, or lot identity.

Before migration-bearing upgrades:

- create or verify an appropriate recovery/backup point where feasible;
- run migrations;
- report failure clearly;
- avoid launching half-migrated services as though healthy.

---

## 8. Uninstall policy

Routine uninstall preserves Victory user data by default.

If the user wants total deletion, provide a separate unmistakable option such as:

> **Permanently delete this Victory lot and all local Victory data**

This should not be preselected. Require explicit confirmation.

---

## 9. First-run human questions

Keep first-run questions few.

Required product questions should be approximately:

1. **What should this Victory lot be called?**
2. **What should your Operator name/handle be?**
3. **Create your Operator password.**
4. **Launch Victory now?** if that is not already implicit.

Do not ask users for database names, ports, secret keys, cookie keys, `.env` values, container names, TLS certificate paths, or Discord credentials.

---

## 10. Generated keys and secrets

Victory should generate its own machine/application secrets using secure randomness.

The installer should generate and persist everything necessary for session/auth secrets, database credentials, internal service credentials, installation identity, and recovery identifiers where appropriate.

Do not ask users to invent cryptographic keys. Do not use fixed sample secrets. Do not commit generated secrets.

---

## 11. `.env` ownership

If Victory continues to use `.env` internally, the installer owns it.

Required:

- generate it;
- validate it;
- store it in the correct protected location;
- preserve it through updates;
- change only values that need changing;
- never expose secrets in ordinary installer logs.

The user should not need to know `.env` exists.

---

## 12. Database bootstrap

The installer owns database runtime startup, database creation, credentials, schema migrations, initial seed data, health verification, and fresh-install identity.

No Grant-specific data may be required.

Even if Kernel 96 has not yet run, K100 should surface any dependency on `straturli`, `amurray-family`, Grant’s user ID, Grant’s production database, or fixed production URLs.

Record unresolved inherited assumptions as blockers for K96 if they cannot be safely corrected within K100.

---

## 13. Fresh Operator bootstrap

The first-run process establishes the first Operator for **this installation**.

Required:

- no pre-existing Operator required;
- no fixed handle;
- no Grant-specific fallback;
- password securely stored/hashed;
- bootstrap path cannot remain open indefinitely after success;
- successful bootstrap leaves a useful local audit receipt.

If bootstrap requires a one-time internal token/key, generate and consume it automatically.

Do not make the user copy secrets between terminal windows.

---

## 14. Fresh lot / Location bootstrap

A new machine gets a new Victory lot.

The original Murray-family lot is historical data belonging to the original installation, not a product seed.

Ask the Operator for a human-facing lot name. Generate safe internal IDs/slugs as needed.

---

## 15. First-time Operator configuration guide

K100 creates the **seam and launch point** for a first-time Operator guide.

K99 will own the full guide/documentation pass.

After installation and first login, Victory should clearly offer:

> **Set up your Victory lot**

or equivalent.

The guide should eventually cover identity/profile, basic lot orientation, optional Discord integration, first Producer/production, invitations, remote-access/share link, backups/update preferences, and where Operator recovery information lives.

K100 must not require K99 to exist before the installer works.

---

## 16. Discord is not part of first install

Do not ask for Discord configuration during installation.

Victory must boot and function without Discord.

Discord setup is optional later.

---

## 17. Remote friends are a normal use case

A Windows Operator should be able to host Victory at home and invite friends elsewhere over the internet.

The product target is not LAN-only.

The user should not need to understand router NAT simply to begin playing if Victory can reasonably avoid it.

---

## 18. Preferred remote-access UX

Preferred final experience:

> **Victory gives the Operator a secure shareable URL.**

The Operator copies it and sends it to players. The player opens it in a browser.

Avoid requiring players to install VPN software, install Victory, configure their router, know the host’s IP address, or understand ports.

---

## 19. Connectivity architecture research

Before committing to the implementation, compare at minimum:

### A. Direct public hosting
UPnP/NAT traversal + public IP/domain + automated TLS where possible.

### B. Connection broker
A lightweight Victory service helps peers locate/connect to a self-hosted lot.

### C. Secure relay/tunnel
The local Victory server creates an authenticated outbound connection to a public relay/tunnel endpoint, avoiding inbound router configuration.

### D. Hybrid
Attempt direct connectivity; fall back to relay/tunnel.

Evaluate:

- setup complexity;
- user friction;
- CGNAT compatibility;
- IPv6;
- TLS;
- operating cost;
- bandwidth cost;
- privacy;
- reliability;
- dependency on Discovery Games infrastructure;
- abuse potential;
- scalability;
- open-source/self-hosting compatibility;
- whether the architecture can later support hosted Victory.

Do not choose based only on easiest developer demo.

---

## 20. Alpha remote-access requirement

For alpha, a secure shareable URL is sufficient even if the final networking architecture later becomes more elegant.

K100 must still avoid making the alpha Operator manually configure TLS certificates.

If the final broker/relay architecture is too large for this kernel, implement a bounded secure path that Victory can launch automatically, produces a stable-enough share link, uses HTTPS, does not expose database/internal service ports, and can be replaced later without rewriting application identity.

Document the dependency.

---

## 21. HTTPS/TLS must be automatic

Users should not manually buy a certificate, run Certbot, edit reverse-proxy config, copy PEM files, or understand certificate renewal.

The remote URL must be HTTPS.

Whatever remote-access architecture is selected must own certificate issuance/termination, renewal, hostname, WebSocket security, and redirect behavior.

---

## 22. Share link

Victory should expose an Operator-facing **Copy Invite/Share Link** affordance.

The link should:

- use HTTPS;
- be easy to copy;
- be human-shareable;
- avoid raw IP/port combinations where possible;
- point to the correct public arrival/login flow;
- not contain reusable privileged secrets.

A short hostname/slug is preferable to third-party URL-shortener dependency.

Do not use public TinyURL-style services for normal operation.

---

## 23. QR code

QR is optional.

It may be useful for moving a link from desktop to phone/tablet, but it is not the primary desktop sharing mechanism.

Do not spend significant time on QR before copy-link works perfectly.

---

## 24. Firewall configuration

The installer should configure Windows Firewall only as needed.

Required:

- explain what network access Victory needs;
- use the narrowest rule practical;
- remove/update owned rules on uninstall/update where appropriate;
- avoid exposing Postgres;
- avoid opening arbitrary broad inbound ranges.

If a relay/tunnel model avoids inbound ports, prefer that reduced exposure.

---

## 25. Router configuration

The preferred path should not require manual router configuration.

If direct-host fallback needs port forwarding:

- Victory may attempt safe automatic configuration such as UPnP when appropriate;
- clearly detect failure;
- fall back to the supported secure remote path;
- do not tell a novice to “open ports” as the default experience.

Manual port forwarding may remain an advanced option.

---

## 26. Consumer status surface

The Operator needs a simple status view.

Show human-readable states such as:

- Victory running;
- database healthy;
- remote access connected;
- public URL;
- update status;
- backup status if available;
- Discord: not configured / connected.

Do not expose container internals as the primary UI.

---

## 27. Start/stop behavior

Victory should behave like installed software.

Implement a coherent approach for start Victory, stop Victory, start with Windows if configured, run in background, reopen Operator UI, and restart after update.

The user should not operate Compose manually.

---

## 28. Desktop/start menu presence

Install appropriate Start Menu entry, desktop shortcut if product convention supports it, uninstall entry, and version information.

Clicking Victory should either start services then open the local Operator interface, or focus/open the already-running instance.

Do not spawn duplicate server stacks.

---

## 29. Automatic updater is required before testers

K100 must include a real update mechanism.

The Operator should not need to download ZIPs, replace source trees, use Git, rebuild containers, or run migrations manually.

Required modes:

- automatic install;
- notify then install;
- manual-only check.

Provide a reasonable default for alpha.

---

## 30. Update source and trust

Updates must be authenticated.

The updater should verify that an update actually comes from the Victory release channel.

Use an appropriate signing/checksum model.

Do not execute arbitrary downloaded artifacts based only on HTTPS.

Record current version, target version, release channel, verification result, and migration result.

---

## 31. Update channels

At minimum architect for stable and alpha/beta/test channel if needed.

Do not overbuild channel management.

Alpha testers may default to the designated test channel.

---

## 32. Update behavior

A normal update should:

1. detect update;
2. download in background where appropriate;
3. verify artifact/signature;
4. prepare backup/recovery point;
5. stop services safely;
6. install application/runtime changes;
7. run migrations;
8. start services;
9. health-check;
10. report success;
11. roll back or present clear recovery instructions if startup fails.

Do not leave users staring at a broken localhost page with no explanation.

---

## 33. Update timing

Victory should not auto-restart in the middle of a live Showing.

If an update arrives during active use:

- download may occur;
- installation waits;
- Operator is informed;
- apply after the Show or at chosen time.

Do not infer that silence means permission to interrupt a production.

---

## 34. Updater and runtime dependencies

If bundled runtime components need updates, define ownership.

Prefer Victory to manage versions it depends upon.

Do not allow an unrelated runtime auto-update to silently break Victory without detection.

---

## 35. Recovery from failed update

The Operator must have a path back.

At minimum:

- preserve user data;
- preserve prior configuration;
- retain enough prior application/runtime state for rollback or repair;
- log the failure;
- provide a support bundle/diagnostic path.

Terminal recovery may exist as break glass. Routine updates should not require it.

---

## 36. Installer updates

The installer itself should support future upgrade installs.

Running a newer Victory installer over an existing installation should detect existing data/config and offer an upgrade path rather than creating a second lot accidentally.

---

## 37. Installer UI tone

The installer is part of Victory.

Use Victory’s product voice and visual identity where feasible.

Prefer human statements:

- “Preparing your Victory lot”
- “Setting up private storage”
- “Creating your Operator account”
- “Making Victory reachable to your players”
- “Victory is ready”

Avoid exposing jargon unless troubleshooting requires it.

---

## 38. Progress reporting

Long operations need truthful progress.

Examples:

- enabling Windows features;
- downloading runtime;
- pulling/installing Victory components;
- initializing database;
- applying migrations;
- establishing secure remote access.

Do not fake a percentage if the installer cannot know it.

Step-based progress is acceptable.

---

## 39. Reboot handling

If Windows features/runtime require reboot:

- tell the user why;
- persist installer state;
- offer restart;
- resume automatically after login/reboot where practical.

Do not make the user redownload or remember where they left off.

---

## 40. Offline considerations

A first installation may require internet access for runtime components, Victory package, remote connectivity, and update verification.

Be explicit.

After successful installation, local Victory operation should not become unnecessarily dependent on a central service except where remote connection architecture requires it.

If the relay/broker is unavailable, local access should still work.

---

## 41. Victory central-service dependency

If K100 introduces a Discovery Games-operated broker/relay/update service, document exactly what it knows.

Minimize central metadata.

Prefer that a broker know only what is necessary to connect instances.

Do not centralize Characters, Shows, messages, Storyboards, or private stage state unless a later hosted architecture explicitly chooses to.

---

## 42. Remote-service outage behavior

If the secure remote-access service is unavailable:

- local Victory remains usable;
- user data remains local;
- Operator receives a clear status;
- retry is automatic where appropriate;
- no data corruption occurs.

If direct/LAN access is available, show it as a fallback.

---

## 43. Installer logging

Keep a bounded installation log useful for support.

It may include Windows version, installer version, steps completed, dependency versions, migration IDs, health results, and non-secret error details.

It must not include plaintext passwords, generated session secrets, database passwords, private Character data, or OAuth secrets.

---

## 44. Support bundle

Provide or establish a seam for **Create Support Bundle**.

A support bundle should collect safe operational diagnostics without private game content where possible.

Useful for alpha.

Do not require Grant to remote-desktop into every tester’s PC.

---

## 45. Installer signing

Release artifacts should be code-signed using an appropriate Windows signing process before broad public distribution.

If signing infrastructure is not yet available for alpha, record current SmartScreen/user-warning behavior and the exact step needed before public release.

Do not treat persistent frightening OS malware warnings as acceptable consumer launch UX.

---

## 46. Licensing / acknowledgements review

Before release, verify:

- Victory’s own license;
- bundled third-party licenses;
- base container images;
- fonts;
- icons;
- JS libraries;
- Go dependencies where notice obligations apply;
- container runtime;
- installer framework;
- tunnel/relay client if bundled;
- update framework.

Update `vendor-acknowledgements.md` or canonical equivalent.

Do not assume “open source” means “no attribution obligations.”

---

## 47. Installer framework selection

Audit current viable Windows installer/updater frameworks rather than hand-writing every low-level installer feature.

Evaluate for code signing, elevation, prerequisites, reboot/resume, update support, rollback, Start Menu/uninstall, license compatibility, and maintainability.

Choose the simplest framework that meets Victory’s product requirements.

---

## 48. Architecture boundary with K96

K100 may reveal security problems.

Do not turn K100 into the full security kernel.

Fix packaging-specific security defects such as exposed database ports, unsafe generated secrets, insecure installer transport, unsigned update execution, insecure remote URL, and unsafe config-file permissions.

Route broader auth/privacy/authority hardening to K96/K97.

---

## 49. Architecture boundary with K99

K100 creates a product that can install itself.

K99 proves a fresh installation and writes/repairs the human documentation around it.

K100 should create structured hooks for K99:

- Operator first-time guide;
- advanced networking guide;
- Discord integration guide;
- backup/restore guide;
- update settings;
- troubleshooting;
- break-glass recovery reference.

Do not delay working installation until every guide is written.

---

## 50. Alpha tester acceptance

A Windows alpha tester should be able to:

1. receive a download link;
2. download one installer;
3. double-click it;
4. approve normal Windows elevation/security prompts;
5. answer only simple Victory questions;
6. wait while Victory handles dependencies;
7. log in as their new Operator;
8. receive a secure remote share link;
9. send the link to a remote friend;
10. have the friend reach Victory in a normal browser;
11. see that updates are configured;
12. close/reopen Victory without technical intervention.

Grant should not have to talk them through Docker, Postgres, `.env`, Caddy, or ports.

---

## 51. Remote-player proof

Do not prove remote access from the same LAN only.

Use a genuinely external network path.

Proof should include:

- host on Windows;
- remote browser on another internet connection;
- HTTPS;
- login/arrival;
- WebSocket/live updates;
- representative asset loading;
- disconnect/reconnect.

Do not expose internal database/backend ports to accomplish this.

---

## 52. Clean-machine proof

Test on a Windows machine/VM that does not already have Victory development prerequisites.

Do not count a developer workstation with Docker, Git, Go, Postgres, Node, certificates, and cached images as the clean-install proof.

---

## 53. Upgrade proof

Test:

- install Version A;
- create representative data;
- update to Version B through updater;
- verify data;
- verify remote access;
- verify Operator identity;
- verify migrations;
- verify uninstall still preserves data.

Use actual packaged artifacts where feasible.

---

## 54. Uninstall/reinstall proof

Test:

1. install;
2. create data;
3. uninstall preserving data;
4. reinstall;
5. detect/recover existing lot;
6. verify data intact.

Then separately test explicit full deletion.

---

## 55. Decision memory

Locked product decisions:

- Windows installation begins from **one downloaded installer** and **one double-click**.
- The installer should take clear foreground control when it genuinely needs user decisions.
- The installer bootstraps all necessary technical operations.
- The user should not manually set `.env`, database credentials, runtime configuration, migrations, TLS, or other developer infrastructure.
- Containerization is acceptable if Victory installs/manages it after the installer begins.
- A separate pre-install Docker step is not acceptable.
- Docker Desktop is not mandatory if licensing/distribution/user-friction argues for another runtime.
- Bundled third-party dependencies require appropriate acknowledgements/licensing compliance.
- Preferred remote play is a **secure shareable URL**.
- Manual TLS setup is not acceptable.
- QR may be optional, but copyable hyperlinks are primary.
- First-run asks only for lot identity and Operator identity/password unless repository evidence requires another genuine product decision.
- Discord setup is skipped during installation.
- K99 should own the fuller first-time Operator configuration guide, but K100 provides its launch seam.
- An automatic/configurable updater is **required before alpha testers**.
- Updates must preserve data and avoid interrupting live Shows.
- Uninstall preserves user data by default.
- K100 aims at the real consumer distribution architecture, not a throwaway alpha shortcut.

---

## 56. Agent checkpoints

### Checkpoint A — Packaging architecture
Before broad implementation, report:

- proposed installer framework;
- proposed runtime strategy;
- whether Docker Desktop is used and why;
- third-party licensing implications;
- application/data/config locations.

### Checkpoint B — Remote connectivity
Report:

- direct/broker/relay/hybrid comparison;
- chosen alpha path;
- ongoing infrastructure/cost implications;
- privacy implications;
- fallback behavior.

Do not make Discovery Games permanently dependent on a costly relay architecture without surfacing that choice.

### Checkpoint C — First packaged install
Produce a real installer and clean-machine result.

### Checkpoint D — Remote external proof
Show secure remote browser connectivity.

### Checkpoint E — Updater
Show A→B automatic update with preserved data.

Then hand to Grant only for consumer-experience judgment.

---

## 57. Pass criteria

K100 passes when:

- one Windows installer starts the entire process;
- no developer prerequisite must be manually installed beforehand;
- runtime dependencies are installed/managed by Victory;
- third-party licensing/acknowledgements are compliant and documented;
- application/data/config separation is sound;
- `.env`/secrets are generated automatically;
- fresh database is initialized automatically;
- fresh Operator is created for that machine;
- fresh lot is created without Murray-specific defaults;
- Victory runs without Discord configured;
- remote friends can connect through a secure HTTPS URL;
- remote connectivity does not require the player to configure networking;
- the Operator is not required to manually manage certificates;
- Windows Firewall/network configuration is handled appropriately;
- Victory can be started/reopened like ordinary software;
- automatic/configurable updates work;
- update artifacts are verified;
- live Shows are not interrupted by surprise auto-update;
- upgrade preserves all representative user data;
- routine uninstall preserves data;
- reinstall can rediscover/recover preserved lot data;
- clean-machine proof passes;
- genuine external-network proof passes;
- installer logs contain no secrets;
- unresolved Windows-signing requirement is explicitly blocked before public release if necessary.

---

## 58. Reportback

Report:

1. installer framework;
2. runtime/container strategy;
3. Docker/Desktop licensing decision if applicable;
4. third-party notice changes;
5. Windows prerequisites handled;
6. install locations;
7. generated configuration/secrets;
8. database bootstrap;
9. Operator bootstrap;
10. lot bootstrap;
11. no-Discord proof;
12. remote-access architecture comparison;
13. selected remote-access implementation;
14. secure URL behavior;
15. TLS handling;
16. firewall/router behavior;
17. local fallback;
18. central-service metadata, if any;
19. remote-service outage behavior;
20. desktop/start behavior;
21. updater architecture;
22. update trust/signing;
23. update timing around Shows;
24. A→B upgrade proof;
25. uninstall/reinstall proof;
26. clean-machine proof;
27. external-network proof;
28. installer/code-signing state;
29. support-bundle/logging behavior;
30. items routed to K96;
31. items routed to K99;
32. items routed to K101;
33. Grant consumer-experience acceptance.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

---

## 59. Completion condition

Kernel 100 passes when an ordinary Windows user can install and operate a self-hosted Victory lot without learning how Victory is built.

They should know:

- what their lot is called;
- who their Operator is;
- where to open Victory;
- what link to send their friends;
- whether Victory is healthy;
- whether an update is available.

They should not need to know which containers are running, which migrations executed, which `.env` keys exist, where TLS terminates, how Postgres was initialized, or which reverse proxy is involved.

The infrastructure remains real.

Victory simply takes responsibility for it.
