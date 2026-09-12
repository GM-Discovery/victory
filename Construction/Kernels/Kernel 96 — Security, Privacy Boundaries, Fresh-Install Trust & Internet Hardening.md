# Kernel 96 — Security, Privacy Boundaries, Fresh-Install Trust & Internet Hardening

**Status:** PASS (2026-09-12) — trust/default audit and full adversarial sweep complete, real findings fixed and verified (production redeployed clean). See `Construction/OperatorLogs/kernel-96-security-hardening-ledger.md` for the full reportback, fix list, and explicitly-named deferrals (CSP/HSTS, release cleanup, git history scrub).  
**Type:** Security architecture audit + deployment hardening + fresh-install trust boundary + adversarial proof  
**Sequence position:** After Kernel 95  
**Primary proof:** A fresh Victory installation can be safely exposed to the public internet without inheriting Grant’s identity, family lot, Discord application, or production secrets, while preserving strict character/privacy boundaries and an SSH-only break-glass recovery path  
**Core doctrine:** Victory stores people’s characters and relationships. Treat that data as precious even when each installation serves only a small group.

---

## 0. Kernel mode

Kernel 96 is not a ceremonial security checklist.

It is a bounded adversarial hardening kernel.

Work in this order:

1. define the actual trust model;
2. audit repository assumptions;
3. identify inherited single-install / Grant-specific assumptions;
4. identify externally reachable attack surfaces;
5. identify authorization/privacy boundaries;
6. attack those boundaries deliberately;
7. fix launch-relevant defects;
8. rerun targeted proof;
9. document what Victory can and cannot promise.

Do not disappear into an open-ended penetration-testing project.

Do not call PASS because tests are green if obvious trust assumptions remain embedded in the product.

---

## 1. Threat model

Victory is intended to be deployed as many small installations rather than one giant centralized service.

A typical installation may hold a small social group — perhaps roughly 3–10 people — but there may eventually be very many such installations.

That distribution reduces centralized blast radius.

It does **not** make the data unimportant.

Treat the following as high-value private data:

- Character records;
- Character mechanics/state;
- private Character notes;
- relationships / “My People” information;
- messages;
- private production information;
- backstage notes;
- Show/Scene state not intended for a user;
- identity/profile information;
- uploaded assets tied to private Shows or Characters;
- any other user-authored material whose exposure could reveal personal, social, or creative information.

Assume attackers may be:

- random internet attackers;
- malicious invited users;
- curious players attempting to cross role boundaries;
- compromised ordinary accounts;
- malicious or compromised browser clients;
- automated credential attackers;
- users attempting malformed uploads or injection;
- users attempting to exploit stale sessions or WebSockets.

The infrastructure/server Operator is a distinct trust category and is addressed separately below.

---

## 2. Operator trust model

A person who controls the Victory host, database, container runtime, backups, and filesystem must be treated as a privileged infrastructure administrator.

Victory should **not** promise users that their data is cryptographically hidden from the server Operator unless Victory later adopts an architecture that actually provides that property.

For launch:

> **Victory protects users from unauthorized users and internet attackers. The Operator of a self-hosted Victory server may have technical access to the server’s database, files, backups, and administrative recovery tools.**

Document this clearly and calmly.

Do not imply that “self-hosted” means “private from the person operating the server.”

Do not describe Operator access as ordinary product access. It is infrastructure-level access.

---

## 3. Privacy promise must be truthful

Victory may say that a properly maintained installation is designed to keep private data inaccessible to unauthorized users.

Victory must not claim:

- the Operator cannot inspect the database;
- the host owner cannot access storage;
- backups are unknowable to the infrastructure administrator;
- private Character data is end-to-end encrypted unless it actually is.

If future application-level or user-held-key encryption is desired, that is a separate architectural project.

K96 should establish a truthful operator/privacy notice suitable for installation documentation, privacy documentation, and Operator-facing setup.

---

## 4. No Grant-specific bootstrap assumptions

A fresh Victory installation must never silently inherit:

- `straturli`;
- Grant’s user ID;
- `amurray-family`;
- Grant’s family lot;
- production-only identifiers;
- Murray-specific seed authority;
- a hard-coded existing Operator;
- a hard-coded existing Location intended only for the original deployment.

These are development-history artifacts, not Victory defaults.

Audit aggressively for literal strings and implicit assumptions.

Tests may retain fixtures where clearly isolated.

Production/bootstrap behavior may not.

---

## 5. Fresh-install bootstrap

Every fresh installation establishes its own local authority.

Required:

- no pre-existing human identity is assumed;
- no Murray-family Location is assumed;
- installation creates or prompts for a new local production lot / Location identity;
- installation creates or explicitly bootstraps the first Operator;
- Operator bootstrap is scoped to that installation;
- bootstrap cannot be rerun casually to create arbitrary new Operators;
- bootstrap leaves an auditable receipt/log entry;
- bootstrap secrets are not committed to the repository.

Prefer a generated installation identity plus an Operator-selected human-facing lot name/slug where product flow supports it.

Do not invent a global Victory cloud identity.

---

## 6. Break-glass recovery

Operator recovery may remain an SSH/terminal-only operation.

Required principles:

- break-glass is not exposed as an ordinary public HTTP route;
- recovery requires host-level access;
- recovery commands are explicit and auditable;
- recovery can restore or establish Operator authority without depending on Grant’s identity;
- recovery does not silently destroy user data;
- recovery documentation warns the human what authority is being granted;
- recovery receipts identify when it happened and what changed.

The product may later offer paid/support-assisted recovery similar to a locksmith model.

K96 does not build billing/support commerce around that idea.

---

## 7. Public internet is a normal deployment condition

Victory must assume a user may run it on a home Windows/Linux machine or small server while friends elsewhere connect over the public internet.

LAN-only assumptions are not acceptable as the launch security model.

Audit:

- externally reachable routes;
- reverse-proxy assumptions;
- TLS expectations;
- secure cookies;
- host/origin behavior;
- WebSocket security;
- rate limiting;
- auth endpoints;
- uploads;
- CORS;
- CSRF where relevant;
- session fixation/reuse;
- password reset/recovery routes;
- default ports and bindings;
- container exposure;
- debug/development routes;
- stack traces;
- error leakage.

Do not require users to share a VPN merely to play Victory.

---

## 8. Discord is optional integration, not boot authority

Victory must remain usable when Discord/OAuth is not configured.

A fresh installation must not require the Operator to create a Discord developer application, know OAuth terminology, provide Discord secrets, configure callback URLs, configure a bot, or configure gateway intents.

Discord can be added later.

K96 should ensure:

- no-Discord first boot is legitimate;
- local account establishment is secure;
- Discord provider availability is accurately surfaced;
- missing Discord configuration does not break unrelated auth/UI;
- optional Discord configuration can be added after install.

---

## 9. Discord setup documentation

Create or update a clear Operator-facing document explaining how to add Discord integration later.

Document:

- what Discord adds;
- what is optional;
- required Discord developer-portal steps;
- redirect URL requirements;
- bot requirements if relevant;
- exact environment variables/configuration;
- restart/reload behavior;
- how to verify it works;
- how to disable/remove it safely.

If the Producer’s Office already has or is intended to have integration-management affordances, document the seam.

Do not build a whole new integration UI unless a small existing Producer-office integration surface can safely expose configuration status.

---

## 10. Authentication without Discord

Audit current password/local auth.

Required:

- no open anonymous signup accidentally grants sensitive Location access;
- fresh-install account creation is bounded and intentional;
- password hashing is appropriate;
- login rate limiting is effective;
- session creation is secure;
- session cookies are Secure in public HTTPS deployment;
- HttpOnly/SameSite behavior is appropriate;
- logout revokes sessions;
- password reset does not leak account existence unnecessarily where practical;
- account enumeration is minimized;
- default credentials do not exist.

Local-development escape hatches must not activate in production accidentally.

---

## 11. Character privacy is a first-class boundary

“People are their Character” is a product-level privacy principle for K96.

A user must not gain unrelated Character data merely because they:

- joined the same Location;
- entered the Third Place;
- know a Character ID;
- know a user ID;
- know a Show ID;
- know a URL;
- altered a request body;
- changed a query parameter;
- connected a WebSocket;
- were once in the same Show;
- have a different Character in the same installation.

Audit authorization at the data-return layer.

Do not rely on hidden UI or CSS.

---

## 12. Third Place boundary

The Third Place must not become a privacy side channel.

Explicit adversarial question:

> “I got access to Third Place. What can I now learn about people that I should not know?”

Test for:

- profile overexposure;
- private Character fields;
- relationship fields;
- contact information;
- email;
- account metadata;
- backstage memberships;
- private Show participation;
- private messages;
- personal notes;
- identifiers that enable follow-on scraping.

Return only what the Third Place product genuinely requires.

---

## 13. Object-level authorization

Audit endpoints for insecure direct object reference / broken object-level authorization.

For every sensitive entity class, ask:

> “If I know another record’s ID, can I fetch, mutate, delete, move, export, or attach it?”

Prioritize:

- Characters;
- Shows;
- Showings;
- Scenes;
- stage objects;
- messages;
- relationships;
- audience admissions;
- Director prep;
- Storyboards;
- eWrite documents;
- assets;
- uploads;
- tickets;
- profiles;
- exports;
- backups/status surfaces.

Use a malicious low-authority account in tests.

---

## 14. Role escalation

Attempt to escalate:

- Audience → Cast;
- Cast → Crew;
- Crew → Director;
- Director → Producer;
- Producer → Operator;
- ordinary account → Operator;
- nonparticipant → participant;
- participant in Show A → authority in Show B.

Try altered request JSON, changed URL IDs, stale membership, duplicated invitations, replayed tokens, forged client-side state, unauthorized WebSocket messages, and direct API calls with hidden controls bypassed.

Server authorization must remain canonical.

---

## 15. Session and WebSocket security

Audit:

- session expiration;
- revoked sessions;
- simultaneous sessions;
- logout;
- password reset invalidation where intended;
- account disable/delete effects;
- WebSocket authentication;
- WebSocket revalidation;
- stale connections;
- cross-Show subscriptions;
- cross-venue broadcasts;
- reconnect behavior.

A user whose authority is revoked should not continue receiving private live data indefinitely through an already-open socket.

---

## 16. Upload scope

Launch-relevant user uploads are intentionally narrow.

Allowed user-facing categories should be limited to product needs such as:

- token assets;
- map assets;
- Director-authorized element assets.

Do not create generic arbitrary-file storage.

Do not allow users to upload code for Victory to execute.

Do not allow uploaded content to become executable server-side or browser-side merely because it has a misleading extension or MIME type.

---

## 17. Upload validation

Required:

- explicit allowlist of accepted media types;
- bounded file sizes;
- bounded dimensions where appropriate;
- content sniffing / validation rather than extension trust alone;
- generated/randomized storage names rather than trusting filenames;
- path traversal prevention;
- filenames treated as display metadata only;
- no shell interpolation;
- no code execution;
- no template evaluation;
- no SQL construction from metadata;
- no HTML/script injection from filenames, captions, alt text, or metadata;
- authorization checked before upload and before retrieval;
- private assets are not guessable/public merely through direct storage URLs.

Strongly consider excluding active-content formats such as unsanitized SVG at launch unless an existing sanitization path is demonstrably safe.

---

## 18. Asset serving

Uploaded assets should be served with safe headers and correct content types.

Audit:

- `Content-Type`;
- `Content-Disposition` where appropriate;
- sniffing protections;
- cache behavior for private assets;
- access control before file delivery;
- filename encoding;
- range requests if supported;
- deleted/revoked asset behavior.

Do not host user-provided HTML/JS as same-origin executable content.

---

## 19. Injection audit

Search all externally influenced inputs for:

- SQL injection;
- command injection;
- path traversal;
- HTML injection;
- stored XSS;
- reflected XSS;
- unsafe template interpolation;
- unsafe URL handling;
- header injection;
- log injection;
- JSON/script-context injection.

Parameterized SQL remains mandatory.

No user-controlled string should become a shell command.

---

## 20. HTML and rich-text safety

Audit rendering paths for chat, eWrite, Storyboards, Character fields, card text, messages, venue labels, Show names, nicknames, asset names, announcements, Director prep, and profile text.

Default to text rendering.

Any intentional rich-text/Markdown path must have a clearly bounded sanitizer or safe renderer.

Do not rely on “our users are friends.”

---

## 21. Secrets and configuration

Audit repository and deployment config for committed passwords, OAuth secrets, bot tokens, SMTP credentials, database credentials, session secrets if any, production URLs, recovery tokens, backup credentials, and test secrets accidentally reused.

Required:

- `.env`/secret files are gitignored;
- fresh install generates or requests secrets;
- defaults are not usable production secrets;
- logs do not print secrets;
- error messages do not echo secrets;
- documentation uses obvious placeholders.

---

## 22. Container and network boundaries

Audit Docker/container exposure.

Required:

- Postgres is not publicly exposed by default;
- backend/database internal networking is intentional;
- only necessary public ports are exposed;
- storage mounts are intentional;
- backup/export mounts are intentional;
- containers do not unnecessarily run privileged;
- no Docker socket mount into Victory services;
- development services are not exposed in production;
- health endpoints reveal only bounded information.

Do not assume Docker itself is a security boundary against the host Operator.

---

## 23. Reverse proxy and TLS

Victory deployed to the internet should have a documented HTTPS path.

K96 should verify and document:

- TLS termination expectation;
- secure cookie requirement;
- trusted proxy assumptions;
- forwarded-header handling;
- canonical public base URL;
- WebSocket upgrade path;
- HTTP→HTTPS redirect strategy where relevant.

Do not hard-code one hosting provider.

Windows consumer packaging may automate this later in Kernel 100.

---

## 24. CSRF / origin-sensitive actions

Audit cookie-authenticated state-changing endpoints.

Where browser cookie authentication is used, verify appropriate defenses against cross-site request forgery.

At minimum assess:

- SameSite cookie policy;
- Origin/Referer validation where appropriate;
- unsafe GET mutations;
- form endpoints;
- OAuth state handling;
- WebSocket origin policy.

---

## 25. CORS

Prefer same-origin frontend/backend deployment.

Do not use wildcard credentialed CORS.

If development CORS exists, ensure it cannot silently become production policy.

---

## 26. Rate limits and abuse controls

Audit externally reachable high-cost or credential-sensitive endpoints.

Prioritize login, signup/invite acceptance, password reset, OAuth start/callback abuse, uploads, exports, account deletion/recovery, message spam, expensive search/document endpoints, and repeated malformed requests.

Do not rate-limit ordinary live-play interactions so aggressively that the product becomes unusable.

Use differentiated limits.

---

## 27. Error leakage

Public errors should not reveal SQL, filesystem paths, secrets, stack traces, private identifiers unnecessarily, internal topology, or raw database errors.

Server logs may preserve enough technical detail for diagnosis while avoiding secrets.

---

## 28. Backup and export security

Audit:

- who can request exports;
- where export archives are stored;
- expiry;
- download authorization;
- predictable URLs;
- stale archives;
- backup status leakage;
- backup file permissions;
- restore authority.

A private export should not become a public static file.

---

## 29. Fresh database proof

Run security acceptance against a truly fresh database.

Fresh install must show:

- no `straturli`;
- no `amurray-family`;
- no inherited production identities;
- no inherited Discord credentials;
- no privileged seeded human account;
- a newly established local Operator;
- a newly established local Location/lot;
- sane default roles;
- no unexpected public data.

---

## 30. Existing database compatibility

Hardening must not silently destroy the original deployment.

Where Grant-specific legacy state exists in an already-running database:

- preserve it as data;
- stop treating it as product defaults;
- migrate only when necessary;
- do not rename/delete legitimate existing Location/user records merely because they are no longer defaults.

The original lot may remain the original lot.

New installs must not inherit it.

---

## 31. Operator setup record

Create/update an Operator-facing security/setup document describing:

- fresh Operator bootstrap;
- fresh lot/Location creation;
- break-glass recovery;
- public internet/TLS expectations;
- Discord optional integration;
- backup responsibilities;
- operator database visibility;
- asset/upload limits;
- security update expectations.

Use plain language.

---

## 32. Adversarial proof account set

Create/use test identities representing at least:

- anonymous visitor;
- Audience;
- Cast;
- Crew;
- Director;
- Producer;
- Operator;
- user from unrelated Show;
- user from unrelated Character/relationship context.

Attempt boundary violations deliberately.

All adversarial proof stays inside the Victory test environment.

---

## 33. Required attack scenarios

At minimum attempt:

1. fetch another user’s private Character by ID;
2. mutate another user’s Character;
3. read Show A private data as Show B participant;
4. read Director-only data as Cast;
5. become Director by altering request payload;
6. become Operator through bootstrap/recovery HTTP paths;
7. replay stale/revoked session;
8. keep receiving WebSocket data after authority revocation;
9. fetch a private asset directly by guessed URL;
10. upload a disallowed file type;
11. upload content disguised with a safe extension;
12. path traversal through filename/path fields;
13. stored script/HTML injection through common text fields;
14. retrieve private Third Place/profile data;
15. retrieve “My People” relationship data without authority;
16. abuse exports/backups;
17. enumerate accounts through auth/recovery behavior;
18. abuse no-Discord fresh-install auth;
19. use an Audience Showing admission outside its Showing;
20. cross Location/lot boundaries where multiple Locations exist.

Record results.

---

## 34. Severity classification

### CRITICAL
Remote unauthenticated compromise, arbitrary code execution, secret theft, Operator takeover, broad database exposure.

### HIGH
Cross-user private data exposure, role escalation to Director/Producer/Operator, arbitrary private asset access, authentication bypass.

### MEDIUM
Limited information leak, abuse-prone endpoint, stale-session issue, missing rate limit, constrained stored XSS, unsafe default requiring unusual conditions.

### LOW
Hardening gap with small realistic impact, metadata leak, minor header/config issue.

### INFORMATIONAL
Defense-in-depth or future improvement.

K96 must resolve CRITICAL and HIGH launch-relevant findings before PASS.

MEDIUM findings require fix or explicit justified deferral.

---

## 35. Dependency / supply-chain review

Perform a bounded dependency review of Go modules, frontend packages, container images, pinned image digests, obviously abandoned/high-risk packages, and known vulnerable versions if repository tooling can check them.

Update launch-relevant vulnerable dependencies when reasonably bounded.

Do not churn every dependency merely because a newer version exists.

---

## 36. Security headers

Assess appropriate headers such as:

- Content-Security-Policy;
- X-Content-Type-Options;
- Referrer-Policy;
- frame embedding policy;
- HSTS when HTTPS deployment is actually established;
- cache controls for sensitive pages/files.

Implement deliberately.

Do not copy a generic header bundle that breaks Pixi, WebSockets, OAuth, or asset loading.

---

## 37. Logging and auditability

Security-relevant actions should leave useful bounded receipts where appropriate.

Candidates:

- Operator bootstrap;
- break-glass Operator recovery;
- role/authority changes;
- sensitive exports;
- account recovery;
- Discord integration configuration state changes if product-managed;
- destructive administrative operations.

Do not log private message/body content unnecessarily.

Do not log passwords/tokens/secrets.

---

## 38. What may remain Operator-visible

Document explicitly that infrastructure Operators may technically access database contents, stored assets, backups, logs, and server configuration.

This is not an invitation for casual snooping.

It is a truthful statement of the self-hosted trust model.

Victory’s application permissions protect users from other ordinary users.

They do not supersede operating-system/root/database-administrator authority.

---

## 39. No central Grant dependency

A third party must be able to install Victory without contacting Grant for routine setup.

Grant/support-assisted break-glass recovery may exist as a service option.

But routine install must not require Grant’s account, Grant’s Location, Grant’s OAuth app, Grant’s database, Grant’s server, or Grant’s secret keys.

---

## 40. Kernel boundaries

K96 does not own:

- full consumer installer UX — Kernel 100;
- complete role-model reconciliation — Kernel 97;
- broad unknown-unknown product audit — Kernel 98;
- release-wide bug burn — Kernel 101;
- end-to-end encryption redesign;
- enterprise key management;
- billing/support commerce;
- new theater capabilities.

If K96 discovers launch-blocking security flaws in adjacent areas, fix the smallest canonical defect and record broader debt.

---

## 41. Human checkpoints

Stop and report to Grant after:

### Checkpoint A — Trust/default audit
Report every Grant/Murray-specific runtime/bootstrap assumption found.

### Checkpoint B — Threat findings
Report CRITICAL/HIGH findings before broad refactoring.

### Checkpoint C — Fresh-install proof
Show the new installation’s Operator/Location/bootstrap behavior and confirm Discord is absent but Victory remains usable.

### Checkpoint D — Final adversarial rerun
Summarize which attacks now fail and why.

Do not bury important security discoveries inside a final giant report.

---

## 42. Decision memory

Locked product decisions:

- Victory may eventually exist as very many small self-hosted containers rather than one giant centralized user database.
- Each small installation still contains precious Character/social/creative data.
- Users must be protected from unauthorized users and internet attackers.
- The server Operator is an infrastructure administrator and may technically access the database/files/backups; Victory must disclose that truthfully.
- `straturli` is Grant’s identity, not a Victory default.
- `amurray-family` is the original installation’s lot, not a Victory default.
- Every fresh installation establishes its own Operator and its own Location/lot.
- Discord/OAuth is optional and may be configured after first boot.
- Operator-facing documentation should explain how to add Discord later, with a Producer-office integration seam where appropriate.
- Public internet connectivity is a normal use case.
- Launch-relevant uploads are limited to token assets, map assets, and Director-authorized element assets.
- Uploaded content must never become executable code or an injection path.
- Break-glass Operator recovery may remain SSH/terminal-only.
- Protecting private Character and relationship data is a high-priority security boundary.
- Third Place access must never imply broad access to private personal/account/Character information.
- K96 should include a bounded adversarial pass, not just a code-review checklist.

---

## 43. Pass criteria

K96 passes when:

- no Grant-specific bootstrap authority remains in fresh-install behavior;
- no Murray-family lot is silently created for unrelated installs;
- a fresh Operator and lot/Location can be established securely;
- break-glass recovery works without Grant-specific identity assumptions;
- Victory operates without Discord configured;
- Discord can be added later through documented operator steps;
- public internet deployment assumptions are documented and hardened;
- Postgres/internal services are not unintentionally public;
- cookie/session/WebSocket behavior is appropriate for internet deployment;
- high-risk object-level authorization paths have adversarial proof;
- Third Place does not leak unrelated private data;
- Character privacy boundaries resist direct-ID/API attacks;
- role escalation attacks fail;
- private assets cannot be trivially guessed/fetched by unauthorized users;
- disallowed/active upload content is rejected or safely handled;
- common injection paths are bounded;
- CRITICAL/HIGH launch-relevant findings are resolved;
- Operator database visibility is truthfully documented;
- automated tests and targeted live attack proof support the result;
- Grant reviews the trust/default findings and final security report.

---

## 44. Reportback

Report:

1. threat model used;
2. trust model used;
3. every Grant/Murray-specific assumption found;
4. fresh-install bootstrap changes;
5. break-glass recovery path;
6. Discord/no-Discord behavior;
7. public-internet deployment findings;
8. auth/session/cookie findings;
9. WebSocket findings;
10. object-level authorization findings;
11. Third Place privacy findings;
12. Character privacy findings;
13. role-escalation findings;
14. upload/asset findings;
15. injection/XSS findings;
16. secrets/config findings;
17. container/network findings;
18. backup/export findings;
19. dependency findings;
20. CRITICAL/HIGH/MEDIUM/LOW ledger;
21. attacks attempted and results;
22. residual risks;
23. explicit Operator-visible-data disclosure;
24. documentation created/updated;
25. automated proof;
26. live/adversarial proof;
27. Grant acceptance result.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

Do not call PASS while unresolved CRITICAL/HIGH launch-relevant findings remain.

---

## 45. Completion condition

Kernel 96 passes when a stranger can install Victory for their own small group and the software no longer behaves as though Grant Murray, `straturli`, the Murray family lot, Grant’s Discord application, or Grant’s production server are part of Victory’s identity.

The installation should belong to its Operator.

The Characters should belong to their people.

The server should be safe to expose to the internet under the documented deployment model.

And Victory should make no privacy promise that its architecture cannot actually keep.
