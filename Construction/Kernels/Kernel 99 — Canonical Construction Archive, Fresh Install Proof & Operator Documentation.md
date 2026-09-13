# Kernel 99 — Canonical Construction Archive, Fresh Install Proof & Operator Documentation

**Status:** CLOSED — PASS (2026-09-12). See `Construction/OperatorLogs/kernel-99-construction-archive-fresh-install-docs-ledger.md`, `Construction/History/` (full archive), and `Docs/` (operator/product/developer documentation suite).  
**Type:** Historical reconciliation + documentation recovery + fresh-install proof + operator onboarding  
**Sequence position:** May be executed after Kernel 100 if desired; should precede final release candidate work  
**Primary proof:** Victory’s full construction history from Kernel 1 through Kernel 100 is safely accounted for, honestly reconstructed where necessary, and a new Operator can install, understand, recover, and operate Victory without tribal knowledge  
**Core doctrine:** If Victory depends on something we once knew but failed to write down, Kernel 99 makes that knowledge durable.

---

## 0. Kernel mode

Kernel 99 has two equal responsibilities:

1. **Preserve Victory’s construction history.**
2. **Prove and document how a stranger operates a fresh Victory installation.**

This is not merely “write a README.”

This is the kernel that makes the preceding hundred kernels survivable.

Every numbered kernel from **1 through 100** must be accounted for.

That does **not** mean inventing specifications that never existed.

It means finding the best surviving evidence, identifying what was actually built, distinguishing original documents from reconstruction, and preserving a canonical historical record.

---

# PART I — THE ONE-HUNDRED-KERNEL ARCHIVE

## 1. Every kernel must have a record

At completion, Victory must have an indexed record for:

- Kernel 1
- Kernel 2
- Kernel 3
- ...
- Kernel 98
- Kernel 99
- Kernel 100

No missing numbers.

If a kernel number was:

- skipped;
- renamed;
- merged;
- superseded;
- split;
- drafted but never executed;
- executed without a surviving spec;
- represented only by comments/commits/chat/operator notes;

that fact must be documented rather than hidden.

---

## 2. Do not rewrite history

The archive must distinguish historical evidence from later reconstruction.

Never create a reconstructed kernel document and present it as though it were the original artifact.

Every kernel record gets a provenance label.

Use one of:

### ORIGINAL
The original kernel/spec survives substantially intact.

### ORIGINAL + AMENDED
Original survives, with later canonical amendments clearly separated.

### RECONSTRUCTED — HIGH CONFIDENCE
Original spec is missing or incomplete, but commits, code, reports, operator logs, comments, and adjacent kernels provide strong evidence.

### RECONSTRUCTED — PARTIAL
Only some of the kernel can be recovered confidently.

### NUMBER RESERVED / NOT EXECUTED
The number existed in planning but no implementation occurred.

### MERGED / SUPERSEDED
Work was absorbed into another kernel and should not be falsely represented as an independent implementation.

### UNKNOWN
Evidence is insufficient. Preserve the gap honestly.

Do not use UNKNOWN until repository/history searches have been exhausted.

---

## 3. The first `//` comment matters

Trace Victory history as far back as the repository allows.

The archive should begin with the earliest identifiable construction intent — including the first meaningful `//` comment or other artifact that functioned as Kernel 1 before formal kernel documents existed.

Do not assume early kernels used the later document format.

Preserve the evolution of the process itself.

---

## 4. Evidence hierarchy

Reconstruct history using the strongest evidence available.

Suggested priority:

1. original kernel files;
2. kernel reportbacks;
3. Git commits/diffs;
4. migrations;
5. source-code comments;
6. operator logs;
7. operator notes;
8. construction contracts/workflow docs;
9. issue/branch names if preserved;
10. adjacent kernel references;
11. tests proving historical behavior;
12. current code only when historical intent cannot otherwise be established.

Current behavior does not automatically prove historical intent.

---

## 5. Git history is primary evidence

Audit the complete Git history.

Use:

- commit timestamps;
- commit subjects;
- changed files;
- migration numbers;
- deleted/renamed files;
- branch history if available;
- tags/releases;
- blame/history on foundational files.

Look for kernel references in:

- commit messages;
- code comments;
- migration comments;
- filenames;
- docs;
- tests.

Do not rely only on the current tree.

---

## 6. Recover deleted documentation where possible

Use Git history to recover:

- deleted kernel specs;
- old README sections;
- earlier operator notes;
- historical architecture docs;
- removed workflow files;
- old deployment instructions;
- historical comments that explain major transitions.

Place recovered historical documents in a clearly marked archival location when preserving them is useful.

Do not reintroduce obsolete instructions into active operator documentation.

---

## 7. Canonical archive structure

Create a durable structure similar to:

```text
Construction/
  Kernels/
    Kernel 001 — ...
    Kernel 002 — ...
    ...
    Kernel 100 — ...
  KernelReports/
    ...
  History/
    Kernel Index.md
    Kernel Lineage.md
    Reconstructed Kernels/
    Archived Original Docs/
    Architecture Milestones.md
```

Adapt to the repository’s existing organization rather than duplicating an established equivalent.

Use zero-padding in the index if useful for sorting, but do not gratuitously rename all existing historical files if links would break.

---

## 8. Kernel Index

Create:

`Construction/History/Kernel Index.md`

or canonical equivalent.

For all 100 numbers include:

| Kernel | Name | Provenance | Status | Primary change | Evidence | Successor/debt |
|---|---|---|---|---|---|---|

This should become the fastest way to understand Victory’s history.

---

## 9. Kernel lineage

Create a narrative lineage showing major eras.

Do not merely summarize 100 rows.

Identify architectural phases such as:

- earliest prototype;
- live table/session foundation;
- identity;
- Character model;
- Greenroom/Trailers;
- venue model;
- production/show architecture;
- documents/eWrite;
- Storyboards/Timeline;
- leadership/turn primitive;
- Socio sustained play;
- dice;
- cartography;
- guided play;
- Director prepared play;
- canonical visibility;
- campus guidance;
- Showtime/Showing;
- Audience experience;
- Vue/beautification;
- security;
- authority reconciliation;
- unknown-unknown hunt;
- fresh-install/archive;
- Windows consumer distribution.

Use actual repository evidence to define the early eras rather than forcing this later vocabulary backward.

---

## 10. Each kernel record

Where an original spec exists, preserve it.

Where reconstruction is necessary, create a record containing at minimum:

```markdown
# Kernel XX — [Recovered Name]

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** [if recoverable]
**Implementation status:** PASS / PARTIAL / UNKNOWN / MERGED
**Evidence sources:** [commits/docs/migrations/tests]

## Reconstructed purpose

## What evidence proves was built

## Files/systems affected

## Known deviations / later corrections

## What this kernel handed to the next kernel

## Confidence / unresolved gaps
```

Do not manufacture dialogue, exact acceptance language, or decisions not supported by evidence.

---

## 11. Original artifacts remain original

Do not “improve” historical kernel specs by silently editing them to match current architecture.

If an old document says The Cave was canonical and later kernels replaced that model, preserve the old document as history.

Add a note or lineage link explaining supersession.

History should reveal evolution.

---

## 12. Reconstructed records must cite evidence

Every reconstructed kernel should identify concrete evidence.

Examples:

- commit hashes;
- migration filenames;
- reportback filename;
- operator-log date;
- source files;
- tests;
- adjacent kernel references.

No evidence means the reconstruction should be marked uncertain.

---

## 13. Status reconciliation

Historical PASS/PARTIAL/FAIL should come from surviving reportbacks where possible.

If no original status exists, do not casually assign PASS.

Use:

- IMPLEMENTED — status not recorded;
- PARTIALLY RECOVERED;
- NOT EXECUTED;
- UNKNOWN;

as appropriate.

The modern archive should not falsify old confidence.

---

## 14. Retroactive reportbacks

Not every historical kernel needs a fake contemporary reportback.

Where useful, create:

> **Retrospective Reportback — reconstructed in Kernel 99**

This must be visibly different from an original reportback.

It may summarize:

- what exists now;
- what historical evidence shows;
- what later kernels repaired.

Do not write “commands run” as though they were run historically.

---

## 15. Architecture milestones

Create:

`Construction/History/Architecture Milestones.md`

Capture major canonical changes and when they occurred.

Examples include:

- when user identity separated from Character/persona;
- when Show replaced Session as product center;
- when selected Character became canonical for Show participation;
- when presence was explicitly noncanonical;
- when visibility became object-state projection;
- when Storyboards generalized into Timeline primitives;
- when Audience admissions became Showing-scoped;
- when Vue entered selected frontend surfaces;
- when Windows consumer distribution architecture was established.

This file should explain *why old code/docs may use different vocabulary*.

---

## 16. Superseded doctrine map

Create a concise “do not resurrect” map for retired assumptions.

Examples from current doctrine include:

- raw Cave/session as product center;
- `current_session_personas` as canonical Show authority;
- sitewide active Character as Show authority;
- access grants/memberships as primary Show authority;
- presence as durable participation;
- Session as canonical Show owner;
- DOM-era stage math where superseded;
- universal Index Card abstraction.

Verify each against repository history before documenting.

This is intended to prevent future agents from rebuilding dead architecture.

---

## 17. Migration chronology

Map database migrations to kernel history where recoverable.

Create an index:

| Migration | Kernel / Era | Purpose | Current status |
|---|---|---|---|

Identify:

- migrations whose kernel attribution is unknown;
- duplicated bootstrap/Ensure surfaces;
- historical compatibility migrations;
- migrations required for fresh install.

Do not renumber historical migrations.

---

## 18. Kernel-to-commit map

Where feasible, map kernels to Git commits.

A kernel may have:

- one commit;
- many commits;
- shared commits;
- no uniquely attributable commit.

Record honestly.

Do not rewrite Git history solely to create clean kernel boundaries.

---

## 19. Kernel-to-feature map

Create a current product map from features back to the kernels that introduced/hardened them.

Examples:

- identity;
- Characters;
- Trailers;
- Greenroom;
- Show;
- Showing;
- Scene;
- Storyboards;
- Timeline;
- dice;
- cartography;
- Audience;
- fanmail;
- tours;
- Showtime;
- visibility;
- trays/live shell;
- installer/updater.

This helps future agents find historical context before changing a feature.

---

## 20. Kernel debt ledger

Many kernels intentionally handed debt forward.

Consolidate unresolved debt.

For each item:

- originating kernel;
- description;
- current status;
- fixed by kernel X;
- still open;
- launch blocker / K101 / K102+.

Do not preserve already-fixed debt as though it remains active.

---

## 21. Construction contracts

Audit and update current construction-process documents, including the equivalents of:

- Construction Contracts;
- Kernel Maker Field Guide;
- Dev Workflow;
- Reportback Template;
- Operator Notes.

The existing Kernel Maker Field Guide contains historical assumptions and vocabulary that may now be obsolete.

Update active guidance to current canonical architecture.

Preserve the old version in history if it is historically important.

---

## 22. Kernel-maker instructions

The final field guide should tell a new coding agent:

- what Victory is;
- current architecture;
- current canonical domain vocabulary;
- where authority lives;
- current runtime modes;
- database/migration rules;
- frontend/Vue conventions;
- Pixi/stage conventions;
- Git workflow;
- how to test;
- how to report back;
- what historical assumptions must not be resurrected;
- where to read relevant prior kernels.

Do not make future agents rediscover 100 kernels.

---

# PART II — FRESH INSTALL PROOF

## 23. Fresh install means truly fresh

K99 must prove Victory from a clean state.

Do not use the long-lived original server/database as proof.

Use an environment without:

- existing Victory database;
- existing `.env`;
- existing Operator;
- existing lot;
- cached private configuration;
- Grant-specific credentials.

If K100 has already created a Windows installer, include its fresh-install path.

Also prove the canonical source/install path intended for developers/operators where relevant.

---

## 24. Fresh Git clone proof

From a clean environment:

1. clone the public/private release repository as intended;
2. follow only documented instructions;
3. initialize required runtime;
4. create configuration through supported tooling;
5. migrate/bootstrap;
6. launch;
7. create/establish Operator;
8. establish lot;
9. reach Victory in browser;
10. perform representative health check.

No undocumented intervention.

Every undocumented intervention found becomes a documentation or installation defect.

---

## 25. No tribal commands

The clean-install proof fails if success requires knowledge such as:

- “Grant usually runs this SQL manually”;
- “use this secret DATABASE_URL”;
- “create this Location row yourself”;
- “copy this old `.env`”;
- “log in as straturli first”;
- “Discord must already be configured”;
- “restart Caddy in this special way”;
- “visit this hidden route before setup works.”

Fix or document the supported mechanism.

---

## 26. Fresh database expectations

A new database should contain only intentional product/bootstrap data.

Audit fresh data for:

- seeded venues;
- system definitions;
- required defaults;
- no Murray-family personal data;
- no Grant account;
- no production messages;
- no test Characters;
- no production Show history.

Document what intentional seed content appears and why.

---

## 27. Linux/operator install path

Even if Windows becomes the primary consumer path, preserve a documented Linux/server installation path where Victory supports it.

Document:

- prerequisites;
- install command/process;
- data directories;
- service lifecycle;
- reverse proxy/TLS expectations;
- update path;
- backup path;
- break-glass path.

Do not make Linux operators reverse-engineer the Windows installer.

---

## 28. Windows install documentation

If K100 executes before K99, document the actual installer rather than the intended installer.

Include:

- supported Windows versions;
- download;
- code-signing/SmartScreen expectations;
- first-run questions;
- data location;
- startup behavior;
- remote URL;
- update settings;
- uninstall/data preservation;
- support bundle;
- recovery.

Use screenshots only if they materially help and can be kept maintainably current.

---

# PART III — FIRST-TIME OPERATOR GUIDE

## 29. Operator onboarding

Create a first-time Operator guide.

Suggested title:

`Docs/Operator/First Time Operator Guide.md`

or canonical equivalent.

The guide should be understandable by someone who does not know:

- Go;
- Docker;
- Postgres;
- OAuth;
- Victory’s construction history.

It should explain the product, not the stack.

---

## 30. Operator onboarding sequence

The guide should cover approximately:

1. Welcome / what an Operator is.
2. Name/orient the lot.
3. Confirm Operator account/recovery.
4. Understand where data lives.
5. Understand remote/share URL.
6. Invite or establish first people.
7. Create/identify first Producer or production path.
8. Optional Discord integration.
9. Update preferences.
10. Backup/recovery basics.
11. Where to get diagnostics/support.
12. What the Operator can technically access.

Keep advanced technical material linked rather than embedded in every step.

---

## 31. Operator versus Producer

Explain clearly:

- Operator = installation/infrastructure authority;
- Producer = production authority inside Victory.

A user may be both.

Do not make ordinary production management require infrastructure knowledge.

---

## 32. Privacy disclosure

Include the truthful self-hosted privacy boundary established for K96/K100 architecture:

> The server Operator may have technical access to the installation’s database, files, backups, and administrative recovery tools.

Explain that application permissions protect users from unauthorized ordinary users, not from the root/database administrator of the machine hosting the software.

Do not overclaim privacy.

If K96 has not yet executed, label this as current architectural truth subject to K96 hardening, not as completed security certification.

---

## 33. Discord integration guide

Discord is optional.

Create/update a dedicated guide explaining:

- what Discord integration adds;
- that Victory works without it;
- how an Operator configures it;
- developer portal steps;
- callback URLs;
- required secrets;
- bot setup if applicable;
- how Victory stores configuration;
- how to test;
- how to disable.

Prefer surfacing this from the Operator/Producer flow after installation.

Do not put Discord setup in the required first-run installer.

---

## 34. Remote access guide

Explain the actual remote-access architecture established by K100.

Include:

- where to find the secure URL;
- how to copy/share it;
- what remote players do;
- expected HTTPS behavior;
- what happens if relay/broker connectivity is unavailable;
- local/LAN fallback if supported;
- advanced direct-host options only if actually supported.

Do not tell ordinary users to configure port forwarding unless it is genuinely required.

---

## 35. Backup guide

Document:

- what constitutes Victory data;
- automatic backups if implemented;
- manual backup mechanism;
- where backups live;
- how to verify;
- how to restore;
- what happens during updates;
- how uninstall interacts with data.

Do not claim backups exist if they do not.

If backup tooling is missing, classify it appropriately before release.

---

## 36. Break-glass recovery guide

Document the SSH/terminal recovery path for technically capable Operators/support.

This may remain advanced.

Include:

- when to use it;
- prerequisites;
- how to restore Operator authority;
- audit effects;
- how to avoid destroying data;
- how to gather support information first.

Keep dangerous commands clearly marked.

Do not expose break-glass through a public web endpoint merely for convenience.

---

## 37. Update guide

Document:

- update modes;
- default update behavior;
- how live Shows affect update timing;
- release channels;
- rollback/recovery behavior;
- how to check current version.

For Windows, normal users should use the updater rather than Git.

---

## 38. Troubleshooting guide

Create a decision-oriented troubleshooting document.

Start from symptoms:

- Victory will not start;
- cannot reach local page;
- remote URL unavailable;
- friend cannot connect;
- login fails;
- update fails;
- database unhealthy;
- assets missing;
- WebSocket/live play disconnected;
- Discord integration fails.

Provide:

- safe first steps;
- status checks;
- support bundle;
- when advanced terminal work is required.

Do not begin with a wall of infrastructure commands.

---

## 39. Support bundle documentation

If K100 provides a support bundle, explain:

- how to generate it;
- what it contains;
- what it intentionally excludes;
- whether users should inspect it before sharing;
- how to send it to support.

Never encourage users to email `.env` or database passwords.

---

# PART IV — USER & CREATOR DOCUMENTATION

## 40. Product vocabulary guide

Create one canonical glossary for:

- Location / lot;
- Venue;
- Show;
- Showing;
- Showtime;
- Scene;
- Character;
- cohort;
- Audience;
- Cast;
- Crew;
- Director;
- Producer;
- Operator;
- stage object;
- visibility;
- Storyboards;
- Timeline;
- eWrite;
- Guide/chat pod.

Retire obsolete public vocabulary from active docs.

Historical docs may retain it.

---

## 41. Role guides

Provide concise user-facing guides for:

- Audience;
- Cast;
- Crew;
- Director;
- Producer;
- Operator.

Do not produce six giant manuals.

Each should explain:

- what the role is;
- where its home/workspace is;
- common actions;
- what it does not control;
- where contextual Guide help appears.

K97 later/earlier remains canonical for actual authority.

---

## 42. Venue guide index

Create an index of major venues and their purpose.

Avoid explaining every UI control in prose if the product already teaches it contextually.

Focus on:

- why the venue exists;
- who uses it;
- what work belongs there;
- what related venue handles adjacent work.

---

## 43. Developer documentation

Document enough for a competent developer to:

- clone;
- run development mode;
- run tests;
- understand backend/frontend/database layout;
- create migrations;
- work with Vue/Pixi/shared shell;
- test WebSockets;
- find domain authority helpers;
- understand deployment/package architecture.

Remove or update obsolete `/opt/victory`-only assumptions if current development now occurs elsewhere too.

Paths should be examples, not product law, unless genuinely required.

---

## 44. Current dev workflow

Update `dev-workflow.md` or canonical equivalent.

It must distinguish:

- local developer mode;
- packaged/install mode;
- server/Linux mode;
- Windows consumer mode where appropriate.

Document which backend/runtime owns which ports.

Avoid stale process confusion.

---

## 45. Operator logs

Define what belongs in Operator Logs.

They should preserve:

- unusual deployment choices;
- recovery events;
- migration incidents;
- infrastructure changes;
- release notes relevant to operation.

They should not become a dumping ground for every kernel narrative.

---

## 46. Reportback standard

Update the reportback template for the mature project.

Keep evidence mandatory.

Add where useful:

- kernel provenance;
- automated proof;
- browser/human proof;
- migrations;
- security implications;
- data implications;
- deferred debt;
- exact commit(s);
- deployment impact.

Do not make reportbacks so bureaucratic that agents avoid writing them.

---

# PART V — DOCUMENTATION SAFETY

## 47. No secrets in docs

Search active and historical docs for:

- real passwords;
- database passwords;
- Discord secrets;
- OAuth client secrets;
- API keys;
- cookies;
- session tokens;
- private recovery codes;
- personally sensitive user data.

Redact secrets from the repository.

If a secret was ever committed, documentation cleanup alone is insufficient: flag it for rotation.

Do not erase useful historical context unnecessarily.

---

## 48. Personal deployment details

Historical records may mention that the first installation was Grant’s/Murray-family environment where that fact is necessary to understand history.

Do not propagate personal identifiers into:

- default installation instructions;
- sample `.env`;
- bootstrap examples;
- consumer guides;
- seed data.

Prefer generic examples.

---

## 49. Public versus internal documentation

Classify docs:

### Public/user
Safe to publish with Victory.

### Operator
Safe for installation administrators.

### Developer
Technical architecture/build information.

### Internal historical
May include implementation archaeology not needed by consumers.

### Sensitive — do not commit
Secrets, live credentials, private database dumps, personal data.

Do not solve documentation by publishing everything indiscriminately.

---

## 50. Links and durability

Use relative repository links where practical.

Verify links.

Avoid links to temporary chat artifacts or `/mnt/data`.

Historical archive should be usable years later without this conversation.

---

## 51. Generated documentation

If documentation is generated from code/schema:

- document generation command;
- commit generated artifacts only where project practice supports it;
- do not let generated output overwrite hand-authored doctrine.

---

# PART VI — MACHINE-READABLE HISTORY

## 52. Kernel manifest

Create a machine-readable manifest, for example:

`Construction/History/kernels.json`

or YAML equivalent.

For each 1–100 include fields such as:

```json
{
  "number": 1,
  "name": "...",
  "provenance": "reconstructed_high_confidence",
  "status": "implemented",
  "spec_path": "...",
  "report_path": "...",
  "commits": ["..."],
  "migrations": ["..."],
  "superseded_by": [],
  "notes": "..."
}
```

This allows future tooling to navigate Victory’s history without scraping Markdown.

Do not put secrets/private personal data in the manifest.

---

## 53. Current architecture manifest

Create or update one concise machine/human-readable current architecture inventory.

Include:

- services;
- database;
- frontend technologies;
- Vue surfaces;
- Pixi runtime;
- WebSocket services;
- data directories;
- install/runtime modes;
- major domain packages;
- deployment components;
- updater/connectivity components from K100.

This is current state, not history.

---

# PART VII — VALIDATION

## 54. Documentation test

A fresh reader should be able to answer:

- What is Victory?
- What is a Show versus Showing?
- What is an Operator versus Producer?
- Where are Characters canonical?
- How do I install it?
- How do I invite remote users?
- How do I update it?
- How do I back it up?
- How do I recover Operator access?
- How do I add Discord?
- How do I run development mode?
- Where do I find the history of a feature?
- Which kernel introduced it?

If documentation cannot answer these, K99 is incomplete.

---

## 55. Fresh-operator human proof

Unlike K98, K99 benefits from one bounded human proof.

Use someone who did **not** build Victory if reasonably available.

Give them the documented installation/onboarding path.

Observe where they need undocumented help.

If no independent tester is available, simulate strictly:

- no shell history;
- no copied old config;
- no existing database;
- only documented instructions.

Do not turn Grant into a repetitive test harness.

---

## 56. Documentation drift checks

Where practical, add lightweight checks for:

- broken internal links;
- missing kernel numbers in manifest/index;
- duplicate kernel numbers;
- referenced files that do not exist;
- obviously stale commands;
- version placeholders.

Do not build a giant docs compiler.

---

## 57. All-one-hundred proof

Automate a simple assertion:

> Kernel numbers 1 through 100 are represented exactly once in the canonical manifest/index.

Merged/reserved/unknown kernels still count as represented.

No gaps.

No duplicate canonical identities.

---

## 58. Historical uncertainty report

At the end, provide a list of kernels whose original intent could not be fully recovered.

For each:

- what is known;
- what evidence is missing;
- confidence;
- whether the gap matters to current maintenance.

Do not hide uncertainty.

---

## 59. K100 relationship

If K100 has already run, K99 documents and fresh-installs the real packaged system.

If K100 has not run, K99 may prepare documentation structure but must not fabricate installer behavior.

After K100, rerun the relevant K99 fresh-install/documentation proof.

Given the current intended order, prefer:

> **K100 implementation → K99 canonical documentation/fresh proof**

---

## 60. K96/K97/K98 relationship

K99 may execute before those kernels.

If so:

- document current architecture truthfully;
- mark pending security/authority/resilience audits;
- do not state those audits have passed;
- create obvious seams for their reports/findings.

When 96–98 later execute, update the archive/index rather than rewriting K99 history.

---

## 61. K101 relationship

K99 should leave K101 with:

- documented install path;
- documented update path;
- known docs bugs;
- consolidated active debt;
- complete kernel manifest;
- clean operator/developer guidance.

K101 should not need to rediscover history.

---

# PART VIII — REQUIRED ARTIFACTS

## 62. Required deliverables

At minimum produce or update:

1. `Construction/History/Kernel Index.md`
2. `Construction/History/Kernel Lineage.md`
3. `Construction/History/Architecture Milestones.md`
4. `Construction/History/kernels.json` or equivalent
5. reconstructed kernel records for missing kernels
6. archival copies/links for recovered original docs as appropriate
7. current `Kernel Maker Field Guide`
8. current `Dev Workflow`
9. current `Kernel Reportback Template`
10. first-time Operator guide
11. Discord integration guide
12. remote-access guide
13. backup/restore guide
14. update guide
15. troubleshooting guide
16. break-glass recovery guide
17. role/product glossary
18. developer setup guide
19. documentation classification/index
20. unresolved historical uncertainty report

Adapt filenames to current repo conventions.

Do not create duplicate competing manuals when an existing canonical file should be updated.

---

## 63. Decision memory

Locked product decisions:

- Every kernel number **1 through 100** must be accounted for.
- Missing historical kernels should be recovered/reconstructed where evidence permits.
- Reconstruction must never be presented as an original artifact.
- The archive should trace Victory back to its earliest identifiable construction intent, including the first meaningful `//` comment where relevant.
- Git history, commits, migrations, reports, code comments, and operator records are legitimate reconstruction evidence.
- Historical evolution should be preserved rather than rewritten to match current doctrine.
- Obsolete active documentation should be updated, while historically meaningful old documentation may be archived.
- K99 owns the fuller first-time Operator configuration/onboarding guide.
- A stranger should be able to install and operate Victory without Grant’s tribal knowledge.
- Documentation must cover remote access, updates, backups, Discord, troubleshooting, and break-glass recovery.
- No secrets or personal deployment defaults should leak into consumer documentation.
- A machine-readable 1–100 kernel manifest should preserve the construction record for future tooling.
- The final archive should make it difficult for future agents to resurrect superseded architecture accidentally.

---

## 64. Human checkpoints

### Checkpoint A — Historical recovery map

Before writing dozens of reconstructed files, report:

- which kernels already have originals;
- which have reportbacks;
- which require reconstruction;
- which appear merged/reserved/unknown;
- evidence sources available.

Let Grant correct only genuine historical misunderstandings.

Do not ask him to retell 100 kernels from memory.

### Checkpoint B — Early-kernel reconstruction sample

Reconstruct a small representative set of missing early kernels.

Show the format and confidence labeling.

Do not reconstruct dozens until the method is accepted.

### Checkpoint C — Fresh-install documentation proof

Run the documented clean installation.

Report every undocumented intervention required.

### Checkpoint D — Final 1–100 manifest

Show the complete index with no gaps.

---

## 65. Pass criteria

K99 passes when:

- kernels 1–100 are represented exactly once in the canonical archive;
- surviving originals are preserved;
- missing kernels are reconstructed to the highest evidence-supported confidence;
- reconstructions are visibly labeled;
- no history is invented to create false completeness;
- Git/commit/migration evidence is linked where practical;
- architecture milestones explain major model changes;
- superseded doctrine is documented so future agents do not resurrect it;
- active field guides/workflow/reportback docs reflect current Victory;
- fresh install succeeds using documentation alone;
- no Grant-specific configuration is needed for a new installation;
- first-time Operator onboarding exists;
- Discord integration is documented as optional;
- remote-access/share-link behavior is documented;
- updater behavior is documented;
- backup/restore behavior is documented truthfully;
- break-glass recovery is documented;
- troubleshooting is symptom-oriented;
- no real secrets remain in committed docs;
- public/operator/developer/internal docs are clearly distinguished;
- machine-readable kernel manifest exists;
- uncertainty is preserved honestly;
- Grant is not required to reconstruct the entire history manually.

---

## 66. Reportback

Report:

1. total original kernel specs found;
2. original + amended count;
3. high-confidence reconstructions;
4. partial reconstructions;
5. merged/reserved kernels;
6. unknown kernels;
7. earliest construction evidence found;
8. recovered deleted docs;
9. kernel-to-commit coverage;
10. kernel-to-migration coverage;
11. architecture milestones created;
12. superseded-doctrine findings;
13. active debt reconciliation;
14. field-guide updates;
15. workflow updates;
16. reportback-template updates;
17. fresh Git install proof;
18. fresh database proof;
19. Windows installer documentation proof if K100 exists;
20. first-time Operator guide;
21. Discord guide;
22. remote access guide;
23. update guide;
24. backup/restore guide;
25. troubleshooting guide;
26. break-glass guide;
27. secret/documentation audit;
28. kernel manifest validation;
29. broken-link/doc checks;
30. unresolved historical uncertainties;
31. items routed to K101;
32. final human acceptance.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

Do not call PASS if any kernel number from 1 through 100 is unrepresented.

An honestly labeled UNKNOWN is representation.

A missing number is not.

---

## 67. Completion condition

Kernel 99 passes when Victory has a memory.

A future developer should be able to trace the software from its earliest surviving construction thought through one hundred kernels without depending on Grant remembering what happened.

A new Operator should be able to install and run Victory without knowing how Grant built the first server.

The historical record may contain uncertainty.

It may contain mistakes later corrected.

It may contain architecture Victory deliberately abandoned.

That is acceptable.

What it may not contain is fabricated certainty, missing construction history disguised as completeness, or critical operational knowledge that exists only in one person’s head.
