# Victory VTT — Kernel Specification and Reportback Protocol

**Version:** 1.0  
**Purpose:** A copy-ready kernel template plus mandatory completion/reporting instructions.

---

# Part I — Kernel Maker Operating Protocol

## 1. Required reading before drafting

A kernel maker must inspect, in this order:

1. `Victory VTT — Master Actual Implementation Guide`;
2. `Victory VTT — Parallel Track Roadmaps`;
3. the most recent relevant kernel specifications and reportbacks;
4. `kernel-maker-field-guide.md`;
5. `operator-notes.md`;
6. the tail of `operator-log.md`;
7. `dev-workflow.md`;
8. the actual repository code, migrations, routes, tests, and current runtime behavior.

The roadmap is not proof that a feature exists. A previous reportback is not proof if its required evidence is missing. Audit actual state before writing dependencies or acceptance criteria.

---

## 2. File locations and naming

Use the repository’s established `Construction/` structure. If subdirectories do not exist, create and document them consistently.

Recommended paths:

```text
Construction/Kernels/kernel-XX-short-slug-v0.1.md
Construction/Reports/kernel-XX-short-slug-reportback.md
Construction/Roadmaps/victory-master-actual-implementation-guide.md
Construction/Roadmaps/victory-track-roadmaps.md
Construction/Templates/kernel-spec-reportback-template.md
Construction/operator-log.md
Construction/operator-notes.md
Construction/dev-workflow.md
Construction/Process/kernel-maker-field-guide.md
```

Do not create a kernel report only in chat. Save it in the repository.

---

## 3. Status meanings

Use only:

- **READY FOR BUILD** — specification complete enough to implement;
- **IN PROGRESS** — implementation underway;
- **PASS** — every required acceptance criterion has evidence;
- **PARTIAL** — useful work landed, but one or more required behaviors or proofs are missing;
- **FAIL** — the kernel did not produce a safe usable increment or was reverted.

A central browser behavior that was only code-reviewed is **PARTIAL**.  
A two-user behavior tested with one client is **PARTIAL**.  
A security claim without a negative rejection test is **PARTIAL**.  
Do not use “PASS with pending required verification.” Minor non-acceptance findings may be recorded under a full PASS.

---

## 4. Kernel size rule

A kernel should produce one coherent usable increment.

A good kernel normally contains:

- one primary domain capability;
- the minimum frontend and backend surface needed to use it;
- migration and compatibility work;
- evidence and operator-memory updates.

Do not combine unrelated roadmap items merely because they share a kernel number. Do combine a generic primitive with its immediate Socio consumer when that makes the increment testable and prevents abstraction churn.

---

## 5. No silent scope rewrite

If the builder discovers the full kernel cannot be completed:

- preserve completed work;
- mark the kernel PARTIAL;
- name each missing acceptance criterion;
- create or recommend a continuation kernel;
- do not rewrite comments or a retroactive report to pretend the original kernel was narrower.

---

# Part II — Copy-Ready Kernel Specification Template

```markdown
# Kernel XX — [Canonical Title]

**Revision:** 0.1  
**Status:** READY FOR BUILD  
**Primary track:** V / S / A / C / O — [milestone]  
**Complementary tracks:** [honest secondary payoffs]  
**Kernel type:** [domain, runtime, UI, migration, repair, verification]  
**Depends on:** [specific proven capabilities, not merely kernel numbers]  
**Supersedes/continues:** [prior spec or report, if applicable]  
**Expected next consumer:** [Socio behavior, generic system, anthology prototype, or concierge surface]

---

## 1. Objective

State one concrete outcome in plain language.

Example:

> Allow a Director to save the current supported stage arrangement as a versioned Scene and later activate that exact revision atomically for every connected client.

---

## 2. Why this kernel now

### Actual repository state

List what was verified in code, database, tests, and runtime.

### Problem being solved

Describe the user-visible or architectural failure.

### Track match

- **Victory primitive:** [what becomes reusable]
- **Immediate Socio use:** [what uses it]
- **Anthology payoff:** [if any]
- **Concierge payoff:** [if any]
- **Operational proof:** [what establishes trust]

---

## 3. Resolved owner decisions

List decisions the builder must not reinterpret.

Examples:

- Commands target the issuer's active character.
- Journal entries are private to the owner.
- A Scene belongs to a Production, not a Venue.
- Token Aura is a real swatch-driven token visual fact and defaults to unset.

---

## 4. Scope

### Included

- [concrete deliverable]
- [endpoint/action/schema/UI]
- [migration/backfill]
- [tests and browser proof]

### Explicitly excluded

- [adjacent feature]
- [future track item]
- [tempting refactor]

---

## 5. Canonical model and invariants

Define:

- authoritative tables/events/projections;
- object identity and ownership;
- append-forward behavior;
- idempotency requirements;
- ordering/versioning;
- privacy;
- reconnect/late-join behavior;
- ruleset ownership versus Victory ownership.

State invariants as testable sentences.

---

## 6. Authority and security

For each operation state:

- who may request it;
- which active object it targets;
- how identity and authority are resolved server-side;
- what client fields are ignored;
- forbidden paths and expected errors;
- whether a trusted public event is server-produced only;
- what happens when access is revoked between preview and execute.

Include at least one negative security acceptance test for any new trusted action or privileged mutation.

---

## 7. Backend work

### Data model and migration

- tables/columns/indexes;
- migration filename expectation;
- bootstrap/fresh-install updates;
- backfill/migration behavior;
- rollback or compatibility notes.

### Domain operations

List typed operations. Do not specify arbitrary JSON mutation.

### HTTP/WebSocket/action surface

List routes, request/response contracts, action types, and broadcast behavior.

### Projection and cache behavior

State how current state is rebuilt and how reconnect/late join receive it.

---

## 8. Frontend work

For each surface state:

- venue/page;
- control and label;
- loading/empty/error/locked states;
- confirmation behavior;
- keyboard and mobile behavior;
- refresh and cross-client updates;
- source/lock/priority indicators where relevant.

Do not treat CSS polish as proof of domain behavior.

---

## 9. Compatibility and migration

State:

- old API/payload compatibility;
- legacy field migration;
- existing database upgrade behavior;
- clean database behavior;
- old client behavior if relevant;
- whether existing actions/history remain readable.

---

## 10. Automated acceptance tests

Number every required test.

Include as applicable:

1. unit/domain tests;
2. authority and forbidden-path tests;
3. idempotency/concurrency tests;
4. migration from clean database;
5. migration from representative existing data;
6. projection/reconnect tests;
7. privacy tests;
8. parser/API tests;
9. regression tests for earlier kernels.

---

## 11. Browser and multi-user acceptance

Specify exact actors, browsers/tabs, venue, session, steps, and expected observations.

Example:

1. User A opens First Theater with Character A active.
2. User B joins the same Session.
3. User A executes `/value add HP 1` and confirms the preview naming Character A.
4. Both clients see one Game Event.
5. User A's shared sheet changes from 6 to 7 HP.
6. Refresh both clients; state remains 7.
7. User B cannot see User A's private Journal entry.

If browser behavior is in scope, this section is mandatory.

---

## 12. Required validation commands

Use the project field guide and tailor commands to touched packages.

Minimum examples:

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go build ./...
GOCACHE=/tmp/victory-gocache go vet ./...
GOCACHE=/tmp/victory-gocache go test ./...
```

Frontend:

```bash
node --check [each touched JS file or extracted inline script]
```

Repository:

```bash
git diff --check
```

When schema/install behavior changes:

```bash
cd /opt/victory
scripts/smoke/fresh-install.sh --local
```

Record any known excluded stateful test and why. Do not quietly omit a failing suite.

---

## 13. Operator and documentation obligations

Before reporting completion, the builder must:

1. update this kernel specification status and add the final commit hash/report path;
2. create the repository reportback file;
3. append a dated entry to `operator-log.md`;
4. update `operator-notes.md` for new durable decisions, authority rules, or runtime traps;
5. update `kernel-maker-field-guide.md` when repo layout, test commands, runtime modes, or common failure modes changed;
6. update `dev-workflow.md` when startup, ports, services, migrations, or validation steps changed;
7. update the Master Actual Implementation Guide row and move unfinished work into a continuation kernel;
8. add migrations to fresh-install/bootstrap paths;
9. update user/operator help for new commands or controls;
10. commit documentation with the implementation or in an immediately linked documentation commit.

---

## 14. Evidence required for PASS

List the exact artifacts the reportback must contain:

- commands and results;
- test names;
- curl/API examples;
- database assertions;
- browser screenshots or precise observed behavior;
- second-user observation;
- negative authority/security test;
- clean-install result;
- commit hash;
- changed-file list.

No evidence means the criterion remains unproven.

---

## 15. Restraint / non-goals

Repeat the boundaries most likely to be violated during implementation.

---

## 16. Definition of done

Summarize the user-visible completed loop in one sequence:

```text
[setup]
→ [user action]
→ [server-authoritative mutation]
→ [projection/broadcast]
→ [refresh/late join]
→ [same canonical result]
```
```

---

# Part III — Mandatory Reportback Template

Save as:

```text
Construction/Reports/kernel-XX-short-slug-reportback.md
```

```markdown
# Kernel Report Back — Kernel XX: [Title]

**Kernel spec:** [repo-relative path]  
**Commit(s):** [hashes]  
**Date:** YYYY-MM-DD

## 1. Status

PASS / PARTIAL / FAIL

Give a one- or two-sentence reason tied to acceptance evidence.

Do not use PASS when required browser, multi-user, migration, or security verification is pending.

---

## 2. Acceptance-criterion ledger

| Criterion | Status | Evidence |
|---|---|---|
| AC-1 [name] | PASS/PARTIAL/FAIL | Test, command, screenshot, observation, or explanation |
| AC-2 [name] | PASS/PARTIAL/FAIL | ... |

Every criterion from the kernel spec must appear. This prevents a polished narrative from hiding omitted scope.

---

## 3. What was built

List concrete shipped deliverables only:

- schema/migration;
- domain operation;
- endpoint/action;
- projection;
- UI control;
- integration;
- documentation.

Do not list intentions or files merely opened.

---

## 4. Evidence

### Commands run

```bash
[exact commands]
```

### Results

Include relevant test counts, outputs, API payloads, database assertions, logs, and errors.

### Browser/multi-user proof

Describe:

- accounts/roles;
- venue/session;
- exact steps;
- what each client observed;
- refresh/reconnect behavior;
- screenshots or recording paths when available.

### Negative/security proof

Describe the forbidden request and exact rejection.

---

## 5. How to run from a clean state

Provide exact operator steps with no hidden assumptions:

1. services to start;
2. environment variables;
3. migration/install command;
4. backend start;
5. URLs;
6. accounts/roles/test data;
7. reproduction steps.

---

## 6. Operator notes

Record non-obvious runtime facts:

- ports;
- services;
- environment variables;
- data/volume paths;
- startup order;
- cache/rebuild behavior;
- migration requirements;
- performance/storage implications;
- security assumptions.

State which notes were also promoted into `operator-notes.md`, the field guide, or dev workflow.

---

## 7. Blockers and workarounds

For each blocker:

**BLOCKER:**  
[description]

**CAUSE:**  
[root issue]

**WORKAROUND:**  
[temporary or permanent response]

**OPERATOR ACTION REQUIRED:**  
[exact action or `None`]

---

## 8. Deviations from the kernel

List every intentional or accidental deviation.

For each deviation state:

- original requirement;
- actual implementation;
- reason;
- consequence;
- whether owner approval was obtained;
- continuation work required.

A documented deviation does not automatically count as completion.

---

## 9. Known issues

List non-blocking rough edges and latent findings.

Distinguish:

- product issue;
- test gap;
- operational trap;
- future enhancement.

---

## 10. Files changed or created

High-level paths grouped by:

- backend;
- database;
- frontend;
- scripts/tests;
- Construction/docs.

---

## 11. Required project-memory updates completed

Mark each:

- [ ] Kernel spec status/commit/report path updated
- [ ] Reportback saved in repository
- [ ] `operator-log.md` appended
- [ ] `operator-notes.md` updated or explicitly not needed
- [ ] `kernel-maker-field-guide.md` updated or explicitly not needed
- [ ] `dev-workflow.md` updated or explicitly not needed
- [ ] Master Actual Implementation Guide updated
- [ ] Fresh-install/bootstrap migration list updated or not applicable
- [ ] Help/command documentation updated or not applicable

Unchecked required memory work means the kernel remains PARTIAL.

---

## 12. Next recommended step

Name the smallest coherent continuation or next implementation slot. Do not silently expand it into a new roadmap.
```

---

# Part IV — Operator Log Entry Template

Append a concise dated entry after every kernel, even when PARTIAL.

```markdown
## YYYY-MM-DD — Kernel XX: [Title] — PASS/PARTIAL/FAIL

### Backend
- [canonical domain/schema/API changes]

### Frontend
- [user-facing surfaces]

### Database / Migration
- [migration number and upgrade/fresh-install result]

### Evidence
- [tests, browser, multi-user, security proof]

### Operational Notes
- [restart, ports, env, cache, install, storage, authority traps]

### Remaining Work
- [only unfinished acceptance or named continuation]
```

The operator log is a chronological changelog. Do not fill it with speculative future design.

---

# Part V — Operator Notes Promotion Rule

Update `operator-notes.md` only when the kernel establishes or changes a durable fact future builders must not rediscover, such as:

- canonical authority rule;
- settled domain distinction;
- persistent runtime trap;
- required startup sequence;
- migration/bootstrap convention;
- privacy boundary;
- active-character targeting rule;
- trusted event production rule;
- source-of-truth decision.

Do not copy the full reportback into Operator Notes.

---

# Part VI — Master Guide Update Rule

After the reportback:

1. mark the actual implementation row PASS/PARTIAL/FAIL;
2. add commit and reportback path;
3. add a short evidence summary;
4. create a continuation row for missing acceptance criteria;
5. reassess only the next three provisional rows unless a major dependency changed;
6. update track milestone status;
7. never erase historical plans—mark them superseded and explain why.

