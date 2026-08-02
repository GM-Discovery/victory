# Kernel 77 — Private-Client Readiness: Deletion, Export, Recovery, and Off-Host Restore

**Status:** READY FOR IMPLEMENTATION
**Type:** Bounded stabilization and private-client readiness kernel
**Primary tracks:** Operational Integrity, Concierge and Distribution
**Secondary tracks:** Victory Core, Player Identity
**Planning authority:** `Victory_Canonical_Roadmap_v2.md`
**Immediate evidence authority:** Kernel 76 reportback and Kernel 76 security artifacts
**Required readiness target:** Level 4 — Private paying clients
**Stretch target:** None. Do not broaden this kernel into public signup.

**Supersedes:** `Construction/Kernels/Kernel 77 — Proposed Hosted-User Stabilization Scope.md`
(a PROPOSAL document, not a final spec). Per Grant's decision on 2026-08-01, this document is
canonical Kernel 77, and every item from the proposal's "Strongly recommended" and "Worth
doing, not blocking" sections (K77-05 through K77-12) is folded in as in-scope, non-optional
work under §25 below. Where the two documents differ, this document wins.

---

## 0. Kernel contract

Kernel 77 closes the specific gaps that prevented Kernel 76 from classifying Victory as ready for private paying clients.

Kernel 76 reached:

```text
Level 3 — Invited strangers
```

It withheld Level 4 because Victory lacked:

1. account deletion;
2. self-service user data export;
3. scheduled encrypted off-host backup with tested restore.

Kernel 76 also identified two related but bounded needs:

4. self-service password recovery with real email delivery;
5. two tutorial tests that fail against a clean test database because they depend on undeclared seed data.

Kernel 77 addresses those five areas, plus the merged proposal items in §25, and no general security wishlist.

The builder must not reopen the entire Kernel 76 audit, redesign authentication, build SaaS, create public signup, replace Discord, or create a general data-management platform.

---

# 1. Locked product decisions

## 1.1 Account deletion

When a user deletes an account:

### Delete private and account-bound data

Delete or irreversibly detach:

- password credentials;
- Discord OAuth/account-link records;
- password-reset and recovery tokens;
- active sessions;
- private journals;
- private relationship records and notes;
- direct or private messages where deletion does not destroy another user's independent record;
- private drafts;
- exclusively owned private uploads;
- unpublished private profile material;
- notification and recovery metadata;
- other records that exist only to serve that user privately.

### Preserve shared history by anonymization

Preserve shared canonical history when deleting it would damage other participants' records, including where appropriate:

- Actions;
- shared Show history;
- Scene contributions;
- Cue history;
- shared cards;
- shared messages or records that another participant legitimately retains;
- collaborative production history.

Preserved history must no longer identify or link back to the deleted account.

Use a neutral display identity such as:

```text
Deleted User
```

Do not preserve:

- email;
- Discord ID;
- handle;
- profile link;
- avatar;
- private fields;
- recovery path.

### Transfer or resolve sole ownership

A user must not delete an account while solely owning a shared structure that must continue to exist.

Examples may include:

- Production;
- Show;
- Scene Library;
- private customer Lot or Location;
- shared asset collection;
- other durable collaborative container.

The deletion flow must require one of:

- transfer ownership to an eligible user;
- delete the owned structure;
- archive it under an approved policy;
- operator-assisted disposition where no safe self-service resolution exists.

The application must explain the blocker. It must not silently fail or leave ownerless records.

## 1.2 User export

A user receives one downloadable archive containing their own data in open, understandable formats.

The archive must include, as applicable:

- profile;
- account metadata suitable for disclosure;
- Characters;
- journals;
- private relationship records;
- messages they authored or own, subject to other users' privacy;
- memberships;
- authored Actions and collaborative-history references;
- owned uploads;
- owned Documents or draft content if present;
- relevant structured metadata;
- a manifest describing each exported component.

Preferred formats:

```text
JSON      structured records
Markdown  human-readable writing and long text
Original  uploaded files in their stored or safely normalized format
README    manifest and explanation
```

Do not export:

- password hashes;
- session tokens;
- raw recovery tokens;
- OAuth tokens;
- server secrets;
- private information belonging solely to another user;
- internal security metadata that would create avoidable risk.

## 1.3 Backup destination

Encrypted off-host backups will be stored in a dedicated folder in Grant's personal Google Drive.

The intended transport is `rclone` using:

- one Google Drive remote;
- one `crypt` remote layered over the Drive remote;
- encrypted file contents;
- encrypted filenames;
- credentials and crypt passwords stored outside Git;
- a dedicated Victory backup path;
- integrity verification;
- tested download and restore.

Google Drive is storage only. It is not the source of application truth.

**Implementation status as of 2026-08-01:** a dedicated Google account (`victoryvtt.ops@gmail.com`)
was created for this purpose. `rclone` is installed on the host. The Drive remote
(`victoryvtt-drive`) and layered crypt remote (`victoryvtt-crypt`, pointed at
`victoryvtt-drive:VictoryBackups`) are configured in `/root/.config/rclone/rclone.conf` and have
been round-trip tested (plaintext filename/content via the crypt remote, encrypted filename/content
via the raw Drive remote). The crypt password and password2 were generated with `openssl rand`,
shown to Grant once, and saved by Grant in Bitwarden. They are not recorded anywhere in this
repository.

## 1.4 Restore control

Infrastructure restoration remains a trusted server/operator operation.

Victory must not expose:

- restore-all;
- database import;
- backup deletion;
- remote backup credentials;
- encryption keys

through an ordinary web page or application role.

A server-side runbook and scripts are required.

## 1.5 Recovery model

Build self-service password recovery with real email delivery.

The intended model is:

- Discord remains the normal account-establishment and login path;
- password login remains available as the independent fallback;
- a user may request a password-recovery email;
- Victory sends a one-time expiring link;
- reset tokens are stored hashed;
- successful reset revokes active sessions;
- request behavior does not materially reveal whether an email exists;
- request and confirmation endpoints are rate-limited;
- operator-issued break-glass recovery remains available.

Grant should not be required to manually repair ordinary lost-password cases.

**Implementation status as of 2026-08-01:** email delivery is configured via Brevo SMTP relay
(`smtp-relay.brevo.com`), sending from `recovery@ops.amurray.family`, a domain verified in Brevo
(SPF/DKIM confirmed green). Credentials live in `backend/.env` (`RECOVERY_EMAIL_*`, `SMTP_*`),
which is git-ignored.

## 1.6 Existing legal and customer-facing documents

Victory already has a Privacy Policy and service terms in the frontend.

Kernel 77 must:

- locate them;
- verify they remain reachable;
- compare their factual claims with current behavior;
- correct technical inaccuracies caused by Kernel 76 or 77;
- ensure relevant account, privacy, deletion, export, recovery, and backup disclosures are present or flag exact operator edits required.

This is not a mandate to draft an entirely new legal regime.

Grant has confirmed these documents exist and were last worked on approximately 2026-05-19;
he was unable to point to proof of their current reachability at kernel-authoring time, hence
this requirement stands as written — locate, verify, and record.

## 1.7 Age boundary

Victory does not offer ordinary service to anyone under 17.

A 17-year-old may participate only under an appropriate supervised license or arrangement.

Victory's future LMS is for adult learners.

Kernel 77 must not create:

- child accounts;
- parental-consent flows;
- school-minor infrastructure;
- COPPA-oriented product behavior.

The existing public policy and signup/account language should not contradict this boundary.

---

# 2. Source findings from Kernel 76

Kernel 77 must begin by reading the full Kernel 76 artifacts and reportback.

At minimum, preserve these facts unless current repository evidence disproves them:

- every Critical and High Kernel 76 finding was repaired;
- open password signup is closed;
- Discord is the initial account path;
- password login remains as recovery fallback;
- password-reset request was closed because no safe delivery existed;
- `victory-recover` is the proven break-glass tool;
- `straturli` resolves through Grant's Discord identity and retains operator/Producer access;
- private Venue, Production, profile, and WebSocket boundaries were tested;
- current production data began from a clean 81-migration rebuild;
- current backups do not leave the host;
- assets are not backed up;
- account deletion is structurally blocked;
- data export does not exist;
- two merchant tutorial tests depend on undeclared dialogue-topic fixtures.

Do not regress any repaired Kernel 76 behavior.

---

# 3. Goals

Kernel 77 has six goals.

## Goal A — Safe account deletion

Provide a user-facing deletion workflow with:

- authenticated confirmation;
- recent-authentication or equivalent sensitive-action proof;
- ownership/blocker discovery;
- explicit transfer/delete choices;
- irreversible confirmation;
- session revocation;
- private-data deletion;
- shared-history anonymization;
- completion receipt;
- negative authorization tests.

## Goal B — Self-service user export

Provide a user-facing export workflow that:

- exports only the requesting user's permissible data;
- produces open formats;
- includes uploads;
- avoids leaking other users' private data;
- supports meaningful size limits and background-safe execution without requiring an asynchronous product system;
- provides a bounded download lifetime;
- is auditable;
- can be reproduced in tests.

## Goal C — Encrypted off-host backup and tested restore

Create a repeatable backup system for:

- PostgreSQL;
- uploaded/user assets;
- required deployment metadata that is safe and necessary to restore;
- migration/version manifest;
- checksums.

Store encrypted copies in Google Drive through `rclone crypt`.

Prove a restore into an isolated environment.

## Goal D — Self-service recovery email

Reopen password-reset request only after real email delivery, rate limiting, privacy-safe behavior, and end-to-end proof exist.

## Goal E — Fresh-database test repair

Make the Kernel 74 and Kernel 75 tutorial tests create their own required dialogue-topic fixtures.

A clean test database must not rely on accidental historical rows.

## Goal F — Reassess Level 4

At completion, run focused Kernel 76 regression checks and classify whether Victory now meets:

```text
Level 4 — Private paying clients
```

Do not claim public-signup readiness.

(See also §25, Goal G — the merged proposal hardening items, required for this same Level 4
classification.)

---

# 4. Anti-lockout and continuity requirements

Kernel 76's anti-lockout contract remains binding.

## 4.1 Preserve Grant access

Before changing recovery or credentials:

- prove current Discord login;
- prove password fallback or break-glass recovery;
- prove `straturli` operator and Producer access;
- preserve `victory-recover`;
- record the current commit and deployment state.

Do not remove a recovery path before its replacement is proven from a fresh browser.

## 4.2 Backup setup must not endanger development

Do not:

- make `/opt/victory` immutable;
- lock Git;
- change repository ownership broadly;
- expose `.env`;
- place Drive credentials in the repository;
- require Google Drive availability to start Victory;
- make ordinary local development depend on the backup remote;
- use `rclone sync` in a way that can delete remote backups unexpectedly.

Backup failure must not crash the application. It must alert or record failure clearly.

## 4.3 Deletion cannot target operator account accidentally

Account deletion for `straturli` must require an explicit exceptional confirmation and must refuse when it would remove the last operator or make the installation unrecoverable.

The break-glass tool must remain able to restore operator access.

---

# 5. Required preflight

## 5.1 Read

Read:

- `Victory_Canonical_Roadmap_v2.md`;
- Kernel 76 spec;
- Kernel 76 reportback;
- all Kernel 76 security artifacts (`Construction/Security/kernel-76-findings.md`,
  `Construction/Security/kernel-76-data-classification.md`);
- current `Security-notes.md`;
- account-recovery runbook;
- backup/restore assessment (`Construction/Operations/victory-backup-restore-assessment.md`);
- current Privacy Policy and service terms;
- identity/auth code;
- session code;
- profile/Character/My People/message/journal code;
- Action and ownership schemas;
- upload/asset code;
- Docker Compose and environment handling;
- migrations;
- test database helpers;
- merchant tutorial fixtures.

## 5.2 Confirm numbering and clean branch

Confirm Kernel 77 is unused.

**Update, 2026-08-01:** at preflight this was found NOT clean — a proposal-status document
already existed under the Kernel 77 number. Resolved per Grant's decision; see the supersession
note at the top of this document. Record kept below unmodified as originally specified for
process continuity:

Record:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

## 5.3 Prove current access

From a fresh browser:

- Discord login as Grant;
- `/api/session/me`;
- Cabin visibility;
- Producer/operator surface;
- logout;
- break-glass `whoami`.

## 5.4 Inventory data ownership

Before designing deletion/export, produce a schema-level map of all tables that:

- reference user/account IDs;
- reference actor IDs;
- own uploaded assets;
- own collaborative containers;
- contain private text;
- contain shared history;
- contain external identities;
- contain sessions/tokens.

Do not implement deletion from an incomplete table list.

---

# 6. Account deletion design

## 6.1 User-facing workflow

Provide a dedicated account-data surface reachable from authenticated account settings.

Minimum flow:

1. Show what deletion means.
2. Identify owned structures that block deletion.
3. Offer valid transfer/delete resolution.
4. Require recent authentication or a one-time email confirmation.
5. Require a deliberate final confirmation phrase or equivalent.
6. Execute deletion transactionally where feasible.
7. Revoke all sessions.
8. Return a final receipt without retaining unnecessary personal data.

Do not use a single casual button.

## 6.2 Deletion service

Create one server-authoritative domain operation.

Do not scatter deletion SQL across handlers.

Suggested conceptual result:

```text
DeletionPlan
- deletable_private_records
- anonymized_shared_records
- transferable_resources
- blocking_resources
- asset_actions
- warnings
```

Then:

```text
ExecuteDeletion(plan, confirmation)
```

The service must:

- re-check the plan at execution;
- verify requester identity;
- verify ownership transfers;
- operate in a transaction where possible;
- fail closed;
- avoid partial account deletion;
- revoke sessions even when later cleanup must be retried safely;
- emit an audit receipt with non-sensitive metadata.

## 6.3 Shared Actions

Resolve `actions.actor_id` deliberately.

Acceptable implementation patterns include:

- nullable actor with preserved immutable display snapshot;
- dedicated deleted-user tombstone account that contains no personal identity;
- anonymous actor record designed specifically for historical retention.

Do not:

- preserve a link to the original account;
- assign deleted history to Grant;
- delete shared history casually;
- use one reusable "deleted user" account if it creates cross-user inference or ownership bugs without a stable anonymized snapshot.

The chosen model must be documented in migration notes.

## 6.4 Private messages

Define behavior carefully:

- delete mailbox/private copy owned solely by the deleting user;
- preserve another user's legitimate copy when it is independently theirs;
- anonymize author identity where shared message history remains;
- do not expose deleted private content through export or audit tooling.

## 6.5 Uploads

For each upload:

- delete when exclusively private and owned only by the user;
- retain/anonymize when embedded in shared collaborative content and another authorized party requires it;
- transfer when ownership of the containing structure transfers;
- remove inaccessible orphaned files;
- clean filesystem/object records consistently.

## 6.6 Owned structures

At minimum test:

- no owned structures;
- one transferrable Production;
- one shared Scene;
- one sole-owned asset library;
- last Producer/owner;
- account with shared history;
- account with private data;
- account with an active session;
- `straturli` as last operator.

## 6.7 Deletion receipt

Keep only what is needed to prove that a request occurred:

- receipt ID;
- timestamp;
- anonymized/internal irreversible account reference if legally/operationally necessary;
- counts by deleted/anonymized/transferred category;
- status.

Do not keep the deleted email or Discord identity merely for convenience.

---

# 7. User export design

## 7.1 Request workflow

Provide an authenticated account-data page with:

- request export;
- current export status;
- download when ready;
- expiration date;
- delete expired export;
- clear statement of included and excluded data.

A first implementation may generate synchronously only when safely small. For realistic accounts, use a server-side export job record and worker/command that can complete without holding one HTTP request open.

## 7.2 Export package layout

Recommended structure:

```text
victory-export-YYYYMMDD-HHMMSS/
├── README.md
├── manifest.json
├── account/
│   ├── profile.json
│   ├── memberships.json
│   └── account-metadata.json
├── characters/
│   ├── index.json
│   └── <character-slug>/
│       ├── character.json
│       ├── face.md
│       ├── mechanics.json
│       └── journal/
├── relationships/
│   ├── relationships.json
│   └── notes/
├── messages/
│   └── messages.json
├── activity/
│   └── authored-actions.json
├── uploads/
│   ├── manifest.json
│   └── files/
└── documents/
```

Only include folders that apply.

## 7.3 Export authorization

The export service must resolve the current user server-side.

It must not accept arbitrary:

- `user_id`;
- `account_id`;
- `email`;
- `handle`;
- `character_id`

as authority to export another user.

Operator UI must not provide a casual "download user export" function.

## 7.4 Other-user privacy

For shared records:

- include the requester's contribution;
- include only the minimum shared context necessary to understand it;
- omit private fields belonging to others;
- avoid exporting another user's email, Discord ID, relationship notes, journals, or unpublished content;
- document unavoidable shared display names.

## 7.5 Archive security

Exports contain sensitive data.

Require:

- unguessable download token or authenticated download;
- short expiration;
- no public static directory;
- restrictive filesystem permissions;
- cleanup of expired archives;
- no archive path traversal;
- bounded archive size;
- safe filenames;
- audit metadata without content logging.

## 7.6 Export proof

Create a fixture account containing:

- profile;
- Character;
- journal;
- relationship note;
- private message;
- shared Action;
- upload;
- membership.

Generate the export and verify:

- expected files present;
- private data included correctly;
- another user's private data absent;
- secrets absent;
- archive reopens;
- manifest counts match;
- expired download fails;
- second user cannot download it.

---

# 8. Self-service email recovery

## 8.1 Delivery abstraction

Inspect current email capability.

Implement a small delivery interface rather than hard-coding one vendor throughout auth code.

For example:

```go
type RecoveryMailer interface {
    SendPasswordReset(ctx context.Context, recipient, resetURL string) error
}
```

A practical first implementation may use:

- SMTP;
- an existing transactional-email provider;
- another already-configured delivery mechanism.

Do not add a large marketing-email system.

## 8.2 Configuration

Configuration must come from environment or protected deployment config.

Likely variables:

```text
RECOVERY_EMAIL_ENABLED
RECOVERY_EMAIL_FROM
RECOVERY_EMAIL_FROM_NAME
RECOVERY_EMAIL_BASE_URL
SMTP_HOST
SMTP_PORT
SMTP_USERNAME
SMTP_PASSWORD
SMTP_TLS_MODE
```

Names may differ to fit the repository.

Do not log secrets.

Fail closed in production when recovery is advertised but mail is not configured.

**Implementation status as of 2026-08-01:** all of the above are set in `backend/.env` using a
Brevo SMTP relay account, sending from a verified `ops.amurray.family` subdomain address.

## 8.3 Request behavior

`POST /api/auth/password-reset/request` may be reopened only when:

- request is rate-limited;
- response does not reveal account existence;
- email lookup is normalized safely;
- token is high entropy;
- only token hash is stored;
- token expires;
- older outstanding tokens are invalidated or safely bounded;
- email contains the correct public base URL;
- no token appears in logs;
- delivery failures do not expose account existence.

Recommended public response:

```json
{
  "ok": true,
  "message": "If that address can receive a Victory recovery message, one has been sent."
}
```

## 8.4 Confirmation

On successful password reset:

- validate one-time token;
- reject expired/replayed token;
- set password using current secure hashing;
- revoke all existing sessions;
- optionally preserve the current reset session only through a new login;
- mark token consumed;
- record non-sensitive audit metadata.

## 8.5 Email-address quality

Grant does not want to manually repair ordinary bad email addresses.

Therefore:

- require users to verify recovery email before relying on it;
- show whether recovery email is verified;
- allow authenticated users to update and reverify it;
- do not claim recovery availability for an unverified address;
- preserve Discord as normal login;
- preserve break-glass recovery for exceptional operator-controlled cases.

Kernel 77 need not guarantee delivery to every provider or fix a user's mistyped address after they lose access.

## 8.6 Recovery proof

Use a controlled test inbox or mail-capture environment to prove:

- request accepted;
- real message generated and delivered;
- link opens;
- token works once;
- replay fails;
- sessions revoked;
- unknown email gets indistinguishable response;
- rate limit works;
- disabled/unverified email behavior is clear.

Do not publish real recovery links or email credentials in the reportback.

---

# 9. Encrypted Google Drive backup

## 9.1 Architecture

Use this separation:

```text
Victory application
    |
    | produces consistent local backup set
    v
Local staging directory
    |
    | checksum + manifest
    v
rclone crypt remote
    |
    | encrypted contents and filenames
    v
Grant's Google Drive / dedicated Victory backup folder
```

The application does not need Google Drive credentials.

Backup scripts and timers run from the trusted host.

## 9.2 Required backup contents

At minimum:

### Database

- consistent PostgreSQL dump;
- enough metadata to identify database and application version;
- migration version;
- dump format and restore command.

### Assets

- uploaded image/file storage;
- any user-created assets not reproducible from Git/migrations;
- filenames or mapping metadata needed to restore them.

### Configuration manifest

Include only safe metadata:

- commit hash;
- timestamp;
- migration count/version;
- backup format version;
- expected archive members;
- checksums;
- source host label;
- retention class.

Do not include raw `.env` in ordinary backup archives.

### Secret-recovery instructions

Document separately what secrets must be preserved outside the host:

- PostgreSQL password;
- rclone configuration or reauthorization procedure;
- rclone crypt password and salt/password2;
- recovery email credentials;
- Discord client secret;
- any application secret needed to resume sessions or OAuth safely.

Do not place the only copy of those secrets in the encrypted backup if the key needed to decrypt it would be lost with the host.

## 9.3 Rclone configuration

Create:

1. a Google Drive remote;
2. a crypt remote layered over a dedicated Drive folder.

The builder must not commit `rclone.conf`.

Restrict permissions.

Use Grant's interactive authorization once when needed.

Prefer a deployment path such as:

```text
/root/.config/rclone/rclone.conf
```

or a dedicated service account user's protected config, depending on the host model.

Record the exact chosen path without recording tokens.

**Implementation status as of 2026-08-01:** done. Config path is
`/root/.config/rclone/rclone.conf`. Remotes: `victoryvtt-drive` (Drive, dedicated
`victoryvtt.ops@gmail.com` account, authorized via `rclone authorize "drive"` run on Grant's
Windows machine and the resulting token applied server-side) and `victoryvtt-crypt` (crypt,
`remote=victoryvtt-drive:VictoryBackups`, standard filename encryption, directory name
encryption on). Round-trip tested successfully.

## 9.4 Copy semantics

Use non-destructive upload semantics for backup archives.

Do not make ordinary backup upload depend on `rclone sync` deleting destination files.

Preferred:

```text
rclone copy
```

or `copyto` for immutable timestamped archives.

Retention deletion should be a separate explicit command with:

- dry-run proof;
- path guard;
- minimum-age rules;
- logs;
- tests.

## 9.5 Encryption

The crypt remote must encrypt:

- file contents;
- filenames;
- directory names where supported/configured.

The crypt password and secondary salt/password must not be:

- committed;
- logged;
- stored in shell history casually;
- kept only on the Victory host.

Grant must receive a clear operator instruction to preserve them in a separate trusted password/secret store.

**Implementation status as of 2026-08-01:** done. Both values generated via `openssl rand`,
shown to Grant once in-session, saved by Grant to Bitwarden under a named login entry. Not
retained in shell history beyond the single generating command; not logged; not committed.

## 9.6 Schedule and recovery objectives

Target:

### Frequent database recovery point

- database backup at least every 15 minutes, or an equivalent mechanism that demonstrates a maximum 15-minute recovery-point objective.

A full `pg_dump` every 15 minutes may be acceptable at current scale if measured and low-cost. If not, use a more appropriate PostgreSQL mechanism.

Do not implement complex WAL archiving merely for prestige if frequent dumps safely meet current scale.

### Daily complete backup

At least once daily:

- database;
- assets;
- manifest;
- checksums;
- encrypted off-host upload.

### Retention

Default target:

- frequent database restore points: 48 hours;
- daily complete backups: 30 days;
- monthly archive: 12 months.

The builder may adjust exact counts based on measured size and Google Drive capacity, but must document any deviation.

## 9.7 Backup script behavior

A backup run must:

1. acquire a safe backup/staging lock;
2. record start;
3. produce a consistent database dump;
4. stage assets without following unsafe symlinks;
5. write manifest;
6. write checksums;
7. package or organize the set;
8. upload through crypt remote;
9. verify remote integrity;
10. record success/failure;
11. avoid deleting the last known good local copy immediately;
12. release lock;
13. return nonzero on failure.

Do not log private filenames in excessive detail when encrypted naming is intended.

## 9.8 Integrity verification

Use an appropriate rclone integrity mechanism for encrypted remotes and/or restore verification.

At minimum:

- compare source backup material against encrypted remote with `rclone cryptcheck` when supported;
- verify local archive checksums;
- download one backup;
- verify checksums after download;
- restore it.

An upload status of zero errors is not enough by itself.

## 9.9 Timers

Use one documented host scheduling mechanism:

- systemd timers preferred when consistent with the host;
- cron acceptable if the project already standardizes it.

Provide:

- service file;
- timer file;
- environment/config path;
- logs;
- manual-run command;
- enable/disable command;
- next-run inspection command.

The schedule must survive reboot.

## 9.10 Failure behavior

Backup failure must:

- not stop Victory;
- produce a visible operator error;
- retain local failed/staged backup long enough for diagnosis;
- return nonzero;
- avoid deleting remote history;
- avoid printing secrets;
- be detectable by a simple health/status command.

A sophisticated alerting service is not required, but silent failure is unacceptable.

## 9.11 Restore proof

Restore into an isolated environment, not over the live database.

Required proof:

1. select one encrypted remote backup;
2. download/decrypt it;
3. verify checksum and manifest;
4. create isolated PostgreSQL database;
5. restore database;
6. restore assets to isolated path;
7. launch Victory against isolated restored state;
8. run health check;
9. authenticate a test account through an appropriate test mechanism;
10. verify representative records and assets;
11. record elapsed restore time;
12. record demonstrated data-loss window.

Do not claim Level 4 without this proof.

## 9.12 Backup documentation

Create:

```text
Construction/Operations/victory-backup-runbook.md
Construction/Operations/victory-restore-runbook.md
Construction/Operations/victory-backup-secret-recovery.md
```

The secret-recovery document must describe what Grant must preserve, without containing the secrets in Git.

---

# 10. Privacy Policy and terms verification

## 10.1 Locate and inspect

Find the existing frontend Privacy Policy and service terms.

Record:

- route/path;
- source file;
- footer/account links;
- effective date if present;
- whether unauthenticated users can reach them.

## 10.2 Technical claims to verify

Check whether the documents accurately describe:

- Discord authentication;
- password fallback/recovery email;
- cookies and sessions;
- public versus private profile data;
- private journals, relationships, and messages;
- account deletion;
- user export;
- uploaded files;
- backup storage in encrypted form through a third-party cloud provider;
- operator-access boundaries;
- retention;
- age eligibility;
- no in-app payment-card storage;
- contact method.

Correct simple factual mismatches.

Do not perform broad legal redrafting without explicit operator direction.

## 10.3 Age language

Ensure no public page suggests the service is intended for children.

Use product-consistent language indicating:

- ordinary users must be at least 17;
- users who are 17 participate only under an appropriate supervised arrangement;
- adult learning is the intended LMS market.

If exact legal wording is not already approved, place the required factual correction in an operator-action note rather than inventing elaborate legal language.

---

# 11. Fresh-database merchant test repair

Kernel 76 proved these tests fail on a freshly created test database:

- `TestKernel74TutorialTailEndToEnd`;
- `TestKernel75TutorialCompletionEndToEnd`.

The reported cause is reliance on `crown-bet` dialogue-topic seed data that existed in a long-lived test database but was not created by the tests.

Required repair:

- identify every required dialogue topic;
- create it through test fixtures/setup;
- ensure fixture is isolated and idempotent;
- do not rely on live migrations containing accidental test-only rows;
- run against newly recreated `victory_test`;
- preserve Kernel 64's hard requirement for `TEST_DATABASE_URL`;
- prove live database row counts remain unchanged.

This is a regression repair, not a tutorial feature kernel.

---

# 12. Required routes and interfaces

Exact route names may follow repository conventions.

At minimum provide equivalents for:

## Account deletion

```text
GET/POST  /api/account/deletion-plan
POST      /api/account/delete
```

or a similarly explicit structure.

## User export

```text
POST      /api/account/export
GET       /api/account/export/status
GET       /api/account/export/download
DELETE    /api/account/export
```

## Recovery

```text
POST      /api/auth/password-reset/request
POST      /api/auth/password-reset/confirm
```

## Operator backup status

Prefer shell-level tooling.

A read-only operator application endpoint may expose only non-sensitive metadata such as:

- last successful backup;
- last failed backup;
- next scheduled run;
- age of last off-host copy.

It must not expose remote tokens, filenames containing private data, or restore controls.

---

# 13. Required migrations

Migrations may be needed for:

- deletion/anonymization model;
- nullable/tombstone Action actor semantics;
- account deletion receipts;
- export jobs and download tokens;
- verified recovery email state;
- recovery request metadata/rate limit support;
- backup status records if application metadata is used;
- ownership-transfer constraints.

Requirements:

- additive where practical;
- safe against clean install;
- no dependency on disposed pre-Kernel-76 data;
- compatible with bootstrap;
- tested on scratch database;
- backed up before live migration;
- clearly documented.

Do not use a migration to store Google Drive or encryption secrets.

---

# 14. Security requirements

## 14.1 Sensitive actions

Deletion, export download, email change, and password reset are sensitive.

Require:

- authenticated user where applicable;
- recent authentication or one-time confirmation;
- CSRF-safe request model;
- server-owned identity;
- rate limiting;
- audit metadata;
- no client-selected target user.

## 14.2 Export archives

- never public;
- no predictable URLs;
- expire;
- remove after expiration;
- restrictive permissions;
- no archive traversal;
- no secrets;
- no cross-user leakage.

## 14.3 Email recovery

- generic request response;
- hashed one-time token;
- expiration;
- replay rejection;
- session revocation;
- rate limit;
- verified recovery email;
- no raw token logging.

## 14.4 Backup

- encrypt before remote storage;
- protect rclone config;
- protect crypt password;
- separate operational logs from secret content;
- non-destructive upload;
- explicit retention deletion;
- isolated restore test;
- no application-level restore button.

## 14.5 Operator privacy

Backup and export tooling should operate on data without creating ordinary interfaces for Grant to browse private content.

Restore proof should use controlled fixtures, not inspection of real private user content.

---

# 15. Required tests

## 15.1 Account deletion tests

- user with only private data deletes successfully;
- credentials removed;
- sessions revoked;
- Discord link removed;
- private journal removed;
- private relationships removed;
- private uploads removed;
- shared Action preserved and anonymized;
- shared message behavior matches policy;
- owned Production blocks deletion;
- ownership transfer clears blocker;
- other user cannot delete account;
- stale deletion plan cannot bypass new ownership;
- last operator cannot self-delete without safe replacement;
- `straturli` remains recoverable.

## 15.2 Export tests

- requester receives own records;
- another user cannot request/download export;
- export excludes password/session/OAuth secrets;
- export excludes another user's private records;
- Markdown and JSON parse/open;
- upload included;
- manifest matches;
- expired token fails;
- archive cleaned up;
- size/path edge cases handled.

## 15.3 Recovery tests

- known email request returns generic response;
- unknown email returns same public response;
- email delivered in test environment;
- valid token resets password;
- replay fails;
- expired token fails;
- sessions revoked;
- rate limit works;
- unverified email not falsely advertised as recoverable;
- raw token absent from logs.

## 15.4 Backup tests

- manual backup succeeds;
- database dump valid;
- asset set included;
- manifest/checksums valid;
- encrypted remote upload succeeds;
- remote filenames/content unreadable without crypt config;
- integrity check passes;
- retention dry run correct;
- timer enabled;
- failure returns nonzero;
- restore from remote into isolated environment succeeds;
- restored app health succeeds;
- representative records/assets present.

## 15.5 Merchant tests

- clean `victory_test`;
- both Kernel 74/75 tests pass;
- no reliance on historic rows;
- live database untouched.

## 15.6 Kernel 76 regression tests

Repeat focused proof for:

- closed signup;
- private Third Place;
- restricted Production list;
- operator-only Discord gateway status;
- unauthorized WebSockets;
- request/frame limits;
- `straturli` login and access;
- break-glass recovery.

---

# 16. Required evidence

## 16.1 Commands

At minimum:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

## 16.2 Test suite

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test ./...
```

Use isolated `TEST_DATABASE_URL`.

## 16.3 Static checks

```bash
git diff --check
```

Run `node --check` for changed inline scripts.

## 16.4 Backup evidence

Record without secrets:

```text
local backup timestamp
database dump size
asset count/size
manifest version
remote upload result
crypt integrity result
download result
checksum result
isolated restore result
restore elapsed time
demonstrated RPO
demonstrated RTO
```

## 16.5 Email evidence

Record:

- provider/mechanism;
- successful controlled delivery;
- request status;
- confirmation status;
- replay status;
- session revocation;
- rate-limit result.

Redact addresses, tokens, and credentials.

## 16.6 Browser evidence

From separate browser contexts prove:

- account-data settings reachable;
- export requested/downloaded;
- deletion blocker displayed;
- deletion confirmation works on test account;
- recovery request gives generic result;
- reset link works;
- Grant still logs in with Discord;
- Cabin and Producer/operator access remain.

---

# 17. Required artifacts

Create or update:

## 17.1 Kernel specification

```text
Construction/Kernels/Kernel 77 — Private-Client Readiness.md
```

(this file)

## 17.2 Deletion policy and dependency map

```text
Construction/Privacy/account-deletion-policy.md
Construction/Privacy/account-deletion-dependency-map.md
```

## 17.3 Export specification

```text
Construction/Privacy/user-export-format.md
```

## 17.4 Recovery runbook update

Update:

```text
Construction/Operations/victory-account-recovery-runbook.md
```

Add self-service recovery and preserve break-glass instructions.

## 17.5 Backup runbooks

```text
Construction/Operations/victory-backup-runbook.md
Construction/Operations/victory-restore-runbook.md
Construction/Operations/victory-backup-secret-recovery.md
```

## 17.6 Backup status evidence

```text
Construction/Operations/kernel-77-backup-restore-proof.md
```

## 17.7 Legal-surface verification

```text
Construction/Privacy/kernel-77-policy-terms-verification.md
```

## 17.8 Security notes

Update current security notes with:

- Level 4 disposition;
- deletion behavior;
- export behavior;
- recovery email;
- backup RPO/RTO;
- remaining risks.

## 17.9 Reportback

Use the standard reportback template.

---

# 18. Level 4 acceptance criteria

Victory may be classified:

```text
Level 4 — Private paying clients
```

only when all of the following are proven:

## Deletion

- user can initiate deletion;
- private data is deleted;
- shared history is anonymized;
- owned structures block or transfer safely;
- sessions and login links are revoked;
- another user cannot delete the account;
- last-operator safety exists.

## Export

- user can request and download an export;
- export is readable and open;
- uploads are included;
- other users' private data is excluded;
- secrets are excluded;
- download expires and cleans up.

## Recovery

- recovery email is actually delivered;
- unknown-account response is privacy-safe;
- token is hashed, one-time, expiring;
- sessions revoke on reset;
- rate limiting works;
- Discord and break-glass access remain.

## Backup

- database and assets leave the host encrypted;
- Drive receives only encrypted backup names/content;
- backup runs on schedule;
- failures are observable;
- retention is controlled;
- crypt/integrity verification passes;
- restore from Drive succeeds in isolation;
- measured RPO and RTO are documented;
- required recovery secrets are preserved outside the host.

## Regression

- all Kernel 76 Critical/High repairs remain effective;
- full test suite passes, including repaired merchant tests;
- Grant remains able to log in and develop;
- no new Critical or High issue is introduced.

## Merged hardening (§25)

- all K77-05 through K77-12 items are resolved or explicitly deferred with Grant's sign-off.

---

# 19. Pass, partial, and fail

## PASS — Level 4 achieved

Use when all Level 4 criteria are proven.

## PASS — Kernel complete, Level 4 withheld

Permitted only when:

- all scoped work is implemented and evidenced;
- a new external blocker genuinely prevents classification;
- the blocker is not disguised unfinished kernel work;
- reportback is explicit.

## PARTIAL

Use when one or more major deliverables are incomplete:

- deletion incomplete;
- export incomplete;
- email not delivered;
- remote backup not scheduled;
- restore unproven;
- merchant tests still fail;
- policy verification incomplete.

## FAIL

Use when:

- private data is leaked by export;
- deletion corrupts shared history;
- account deletion leaves active login;
- backup is unencrypted remotely;
- restore damages live data;
- Drive retention deletes unintended files;
- Grant is locked out;
- secrets are committed or printed;
- Level 4 is claimed without restore proof.

---

# 20. Non-goals

Kernel 77 does not:

- open public signup;
- build billing;
- build subscriptions;
- create a SaaS tenant-control plane;
- redesign Discord auth;
- replace the Privacy Policy or terms wholesale;
- serve users under 17;
- build child or school-minor controls;
- build Victory Documents;
- redesign Catharsis;
- provide customer-facing server restore;
- create zero-knowledge or end-to-end encrypted application data;
- guarantee zero data loss;
- build unlimited retention;
- migrate to a different cloud provider without evidence;
- make Grant responsible for manually correcting ordinary broken recovery emails.

---

# 21. Builder guidance

## 21.1 Do not confuse deletion with erasure of shared reality

Victory is a shared performance and game-history system.

A user may erase their private data and identifying link without destroying every shared event in which they participated.

Anonymization is the correct tool for shared canonical history.

## 21.2 Do not confuse export with database dumping

A user export is a curated, understandable package of that user's permissible data.

It is not a PostgreSQL dump.

## 21.3 Do not confuse upload with backup

A backup is not proven until:

- encrypted off-host copy exists;
- integrity is checked;
- restore succeeds.

## 21.4 Do not make Google Drive a runtime dependency

Victory must continue operating during:

- Drive outage;
- expired Drive token;
- rclone failure;
- network interruption.

The failure must be visible, not fatal to the application.

## 21.5 Do not weaken Kernel 76 controls

No convenience added here may reopen:

- public password signup;
- raw token logging;
- authenticated-only-but-not-admitted access;
- operator data browsing;
- unsafe WebSocket subscription;
- default database credentials.

---

# 22. Required reportback additions

## 22.1 Achieved readiness

```text
Achieved readiness: Level X — [label]
Required target: Level 4 — Private paying clients
```

## 22.2 Deletion proof

Include:

- private rows deleted;
- shared rows anonymized;
- owned-resource blocker;
- transfer result;
- session revocation;
- last-operator protection.

## 22.3 Export proof

Include:

- archive layout;
- record counts;
- upload count;
- omitted secret classes;
- cross-user negative test;
- expiration/cleanup.

## 22.4 Recovery proof

Include:

- mail delivery mechanism;
- controlled delivery result;
- generic unknown-user response;
- token replay rejection;
- session revocation;
- rate-limit result.

## 22.5 Backup proof

Include:

```text
Backup destination: encrypted Google Drive crypt remote
Backup schedule:
Retention:
Last successful off-host backup:
Integrity check:
Restore source:
Restore target:
RPO:
RTO:
```

Do not include remote names if they reveal sensitive structure unnecessarily, and never include tokens or crypt passwords.

## 22.6 Legal-surface verification

List:

- Privacy Policy path;
- terms path;
- corrections made;
- operator wording decisions still required.

## 22.7 Remaining blockers

List only blockers that genuinely prevent private paying clients.

---

# 23. Acceptance checklist

## Preflight

- [x] Kernel number verified (conflict found and resolved — see supersession note).
- [ ] Kernel 76 artifacts read.
- [ ] Current Grant access proven.
- [ ] Break-glass recovery proven.
- [x] Current commit recorded (`c3a7a3a58c6f3a9e19216a7e0793c3d11821b7f1`, branch `main`, clean).
- [ ] Ownership/dependency map complete.

## Deletion

- [ ] Deletion plan endpoint/service exists.
- [ ] Sensitive confirmation exists.
- [ ] Private credentials removed.
- [ ] Sessions revoked.
- [ ] OAuth link removed.
- [ ] Private journals removed.
- [ ] Relationship notes removed.
- [ ] Private uploads handled.
- [ ] Shared history anonymized.
- [ ] Owned-resource blockers work.
- [ ] Transfer flow works.
- [ ] Last operator protected.
- [ ] Cross-user deletion denied.

## Export

- [ ] Export package documented.
- [ ] JSON records generated.
- [ ] Markdown generated where appropriate.
- [ ] Original files included.
- [ ] Manifest included.
- [ ] Secrets excluded.
- [ ] Other-user private data excluded.
- [ ] Download protected.
- [ ] Download expires.
- [ ] Expired archive cleaned.
- [ ] Cross-user download denied.

## Recovery

- [x] Email delivery configured (Brevo SMTP relay, verified sending domain).
- [ ] Recovery email verification exists.
- [ ] Request response is generic.
- [ ] Token stored hashed.
- [ ] Token expires.
- [ ] Token replay fails.
- [ ] Sessions revoke.
- [ ] Rate limit works.
- [ ] Raw token absent from logs.
- [ ] Discord login remains.
- [ ] Break-glass remains.

## Backup

- [x] Google Drive remote configured.
- [x] Crypt remote configured.
- [ ] Config permissions restricted (verify explicitly).
- [x] Crypt credentials preserved outside host (Bitwarden).
- [ ] Database included.
- [ ] Assets included.
- [ ] Manifest/checksums included.
- [ ] Upload is non-destructive.
- [ ] Integrity verification passes.
- [ ] Frequent schedule enabled.
- [ ] Daily complete schedule enabled.
- [ ] Retention implemented separately.
- [ ] Failure visible.
- [ ] Remote download proven.
- [ ] Isolated restore proven.
- [ ] Restored app health proven.
- [ ] RPO recorded.
- [ ] RTO recorded.

## Regression

- [ ] Merchant tests seed their own topics.
- [ ] Clean test database passes.
- [ ] Full Go suite passes.
- [ ] Live database unchanged by tests.
- [ ] Closed signup still closed.
- [ ] Third Place remains gated.
- [ ] Production list remains restricted.
- [ ] Gateway status remains operator-only.
- [ ] WebSocket negatives pass.
- [ ] `straturli` access remains.
- [ ] Dev Mode remains usable.
- [ ] Install Mode remains usable.

## Merged hardening (§25)

- [ ] K77-05 WebSocket session revocation.
- [ ] K77-06 broader rate limiting.
- [ ] K77-07 explicit CSRF posture.
- [ ] K77-08 WebSocket auth before upgrade.
- [ ] K77-09 pinned image digests.
- [ ] K77-10 `/health` scope decision.
- [ ] K77-11 stale worktree resolved.
- [ ] K77-12 duplicate Kernel 75 files reconciled.

## Documentation

- [ ] Deletion policy written.
- [ ] Deletion dependency map written.
- [ ] Export format written.
- [ ] Recovery runbook updated.
- [ ] Backup runbook written.
- [ ] Restore runbook written.
- [ ] Secret-recovery instructions written.
- [ ] Policy/terms verification written.
- [ ] Security notes updated.
- [ ] Reportback completed without secrets.

---

# 24. Immediate operator outcome

At completion, Grant must be able to answer:

1. Can a paying client delete their account without destroying everyone else's history?
2. Can they take a readable copy of their own data?
3. Can they recover a forgotten password without Grant's intervention?
4. Does Discord login still work?
5. Can Grant recover `straturli` from the shell?
6. Does an encrypted database-and-asset backup leave the host automatically?
7. Can that backup be restored from Google Drive?
8. What is the demonstrated maximum data-loss window?
9. How long did restoration take?
10. Are the Privacy Policy and terms still factually accurate?
11. Do clean-database tutorial tests pass?
12. Has Level 4 actually been proven?

Kernel 77 succeeds when Victory can accept a private paying client without relying on untested promises about deletion, portability, recovery, or disaster recovery.

---

# 25. Merged scope from the superseded proposal document

The following items are carried forward from
`Kernel 77 — Proposed Hosted-User Stabilization Scope.md` (K77-05 through K77-12) per Grant's
2026-08-01 decision to merge both documents, with this document winning on any conflict. These
are in-scope, required work for this kernel's Level 4 classification — not optional stretch
items.

## Goal G — Merged hardening items

### K77-05 — Session revocation reaches open WebSockets

Revoking a session today prevents new connections but does not tear down sockets already open
under it. Add a periodic re-validation or a revocation broadcast to the hub.

### K77-06 — Rate limiting beyond credential endpoints

Only `/api/auth/*` and `/api/account/email` are throttled as of Kernel 76. Extend to message
sending, note cards, uploads, command execute, WebSocket connections and messages, and the new
export endpoint.

Two constraints from Kernel 76 §5.18: derive the client IP correctly behind Caddy rather than
trusting arbitrary forwarded headers, and ensure no limiter can lock the operator out without
recovery.

### K77-07 — Explicit CSRF posture

Protection currently rests implicitly on `SameSite=Lax` plus the fact that no GET route mutates
state. That is adequate but undocumented and easy to break by accident. Either adopt a token
for state-changing routes or add a test asserting no GET handler mutates.

### K77-08 — Reject WebSocket upgrades before completing them

Authenticate before `upgrader.Upgrade` rather than after (Kernel 76 finding K76-L01).

### K77-09 — Pin base images to digests

Pin `postgres:16-alpine` and `caddy:2-alpine` to digests rather than floating tags.

### K77-10 — Decide `/health` exposure

Decide whether to expose a deliberately minimal public `/health` endpoint (Kernel 76 finding
K76-I03), and implement that decision. Note: Kernel 72's security notes record that `/health`
is not proxied by Caddy today — confirm current behavior before changing it.

### K77-11 — Resolve the stale agent worktree

`.claude/worktrees/agent-ad8c3df7c85e39ce3/` is a stale agent worktree. Its committed history
(`dc8c54f`) is confirmed fully merged into `main`. It also contains ~700 lines of uncommitted,
never-merged local changes to `frontend/lib/stage-runtime/` and
`frontend/venues/show-runs/show.html` whose value is unconfirmed as of 2026-08-01. Its
`docker-compose.yml` hardcodes the pre-Kernel-76 placeholder password
(`POSTGRES_PASSWORD: change_this_now`) rather than reading from `.env` — not a live credential,
but the insecure pre-fix pattern. Resolve by inspecting the uncommitted diff for anything worth
salvaging, then removing the worktree.

### K77-12 — Reconcile the two competing Kernel 75 spec files

`Construction/Kernels/kernel-75-tutorial-completion-aftercare-continuation-v0.1.md` and
`Construction/Kernels/kernel-75-tutorial-completion-story-so-far-aftercare-mvp-proof-v0.1.md`
both exist. Determine which reflects the actually-shipped Kernel 75 behavior and mark or remove
the other.
