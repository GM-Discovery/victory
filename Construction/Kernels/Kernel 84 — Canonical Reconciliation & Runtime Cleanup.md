# Kernel 84 — Canonical Reconciliation & Runtime Cleanup

**Status:** READY FOR IMPLEMENTATION  
**Type:** Operational integrity / cleanup / roadmap reconciliation  
**Primary tracks:** O — Operational Integrity, Victory Core  
**Secondary tracks:** Socio, Anthology, Concierge  
**Sequence position:** After Kernel 83  
**Planning authority:** Victory Canonical Roadmap v2 (canonical repository filename)  
**Implementation authority:** repository behavior, current database state, kernel reportbacks, operator log, operator decisions

---

## 0. Kernel contract

Kernel 84 is a cleanup and reconciliation kernel. It must not become a new product-feature band.

It has two equal responsibilities:

1. Repair bounded known runtime/documentation debt discovered through Kernels 76'83.
2. Bring Victory's single canonical roadmap and current-state documentation into alignment with the repository through Kernel 83, recovering any missing kernel history that can be proven from repository/reportback evidence.

The kernel should leave Victory in a state where:

```text
current repository
→ current operator documentation
→ current canonical roadmap
```

Known discrepancies must be recorded rather than hidden.

Prefer repair over refactor. If the audit discovers a larger architectural problem, document it, rank it, and split it into a named continuation/follow-up rather than allowing Kernel 84 to expand without bound.

---

# 1. Locked product/operator decisions

## 1.1 Cleanup aggressiveness

Fix:

- known defects;
- stale or contradictory documentation;
- bounded adjacent bugs of the same root cause;
- disposable test residue;
- broken or obsolete verification instructions;
- roadmap omissions that repository evidence can resolve.

Do **not** refactor merely for elegance.

If a repair crosses several domains, requires a major new abstraction, risks user-visible behavior, or cannot be adequately proven inside this kernel, record it and split it into a follow-up.

## 1.2 Canonical roadmap update is required

Kernel 84 must update the **single Canonical Roadmap** through Kernel 83.

The update must:

- reconcile the implementation baseline;
- recover kernels missing from the roadmap but provable from repository/reportback evidence;
- correct stale track statuses;
- update the skip/defer ledger;
- update open decisions where now resolved;
- set the next three committed kernels;
- append a dated change-log entry.

## 1.3 Recover kernel history where evidence exists

Do not rely on stale roadmap numbering.

Recover actual post-roadmap kernel history from repository evidence.

At minimum investigate the real sequence after Kernel 75 and reconcile all provable work through Kernel 83, including continuation/split kernels.

Likely areas include:

- Kernel 76 hosted-readiness/security audit;
- Kernel 77 stabilization/private-client/security work;
- Kernel 77A migration-backed venue seed repair;
- Kernel 78 Victory Documents/eWrite foundation;
- Kernel 79 and continuation/split passes;
- Kernel 80 Storyboards core;
- Kernel 81 Storyboards presentation;
- Kernel 81A structural-name serialization;
- Kernel 82 Timeline mode;
- Kernel 83 Venue Leadership & Turn State.

These names are orientation only. Repository/reportback evidence determines exact titles, statuses, dates, commits, and scope.

If additional kernels or continuations exist, recover them too. If an expected kernel cannot be proven, do not invent it; mark the historical gap explicitly.

## 1.4 Old Master Actual Implementation Guide

The old Master Actual Implementation Guide is historical and non-canonical.

Kernel 84 must:

- preserve it for history;
- mark it clearly as superseded/non-canonical;
- point readers to the Canonical Roadmap;
- stop future agents from treating it as active sequence control.

Do not delete historical planning evidence unless repository conventions explicitly prefer archival relocation.

The Canonical Roadmap's one-file rule controls future planning.

## 1.5 Next committed horizon

Subject to Kernel 84 discovering no material blocker, update the Canonical Roadmap toward:

1. Kernel 85 — Socio Sustained Play
2. Kernel 86 — Cartograph-Style Drawing Foundation
3. a third committed slot only if repository evidence makes its dependency clear.

Do not invent Kernel 87 merely to fill a box. If the roadmap format requires a third future slot and no evidence supports one, record it as explicitly decision-gated/provisional.

If Kernel 84 finds a serious blocker, a bounded 84A/85 repair may precede Socio. Record why.

---

# 2. Known cleanup target: Storyboards WebSocket context lifetime

Kernel 83 reported a pre-existing issue: `ServeStoryboardWS` creates a short-lived context and some mid-connection database work may reuse that context after it has expired.

Kernel 83 worked around this only for its new disconnect cleanup by creating a fresh context.

Kernel 84 must investigate the full Storyboards WebSocket path.

## 2.1 Required investigation

Trace:

- connection establishment;
- authentication;
- initial `watch_board`;
- subsequent `watch_board` messages;
- board switching;
- snapshot loading;
- presence/watch registration;
- coordination-session lifecycle;
- disconnect cleanup;
- database queries performed after connection establishment;
- event broadcasting.

Identify every DB/network operation that incorrectly relies on a context tied to a short initial setup window.

## 2.2 Required repair

Use lifecycle-appropriate contexts.

Do not solve this by making every context effectively infinite.

Expected principles:

- request/setup context for handshake/setup work;
- connection-lifetime cancellation where appropriate;
- fresh bounded contexts for discrete DB operations;
- disconnect cleanup with a fresh bounded context;
- cancellation on actual socket/session shutdown.

Document the chosen lifecycle.

## 2.3 Regression proof

Prove:

- connect and watch board;
- remain connected longer than the old timeout;
- send a second valid `watch_board` or board-switch operation;
- DB-backed snapshot succeeds;
- old-board cleanup succeeds;
- new-board registration succeeds;
- disconnect cleanup succeeds;
- no leaked watcher/session state remains;
- Kernel 83 coordination lifecycle still behaves correctly.

---

# 3. Adjacent WebSocket/session audit

Perform a bounded same-root-cause audit of adjacent live systems.

Look for:

- short setup contexts reused for long-lived sockets;
- cleanup code using already-cancelled request contexts;
- watcher/presence entries not removed on disconnect;
- session end tied incorrectly to one browser tab rather than real last-watcher semantics;
- board-switch paths leaving stale state;
- live broadcasts crossing venue/board boundaries;
- duplicate in-memory session registries accidentally overlapping responsibilities.

Do not redesign unrelated WebSocket architecture.

If another live domain has the same clear context-lifetime bug and the repair is low-risk, fix it. Otherwise report it as follow-up debt.

---

# 4. Disposable test account / browser-proof workflow cleanup

Kernel 83 established that production password signup is closed and older testing guidance is stale.

## 4.1 Field guide update

Update the kernel-maker field guide's multi-user / two-browser / disposable-user technique.

The canonical method should reflect current security policy.

If direct fixture-row creation is the supported test method, document:

- where it is permitted;
- that it is for disposable test accounts only;
- required user/session rows;
- token hashing convention;
- isolation expectations;
- cleanup requirements;
- why production password-signup routes must not be reopened merely for tests.

Do not copy secrets or real account data into documentation.

## 4.2 Verify current signup/auth documentation

Search active:

- field guide;
- dev workflow;
- operator notes;
- security notes;
- current workflow docs.

Correct instructions that still claim password signup is available.

Historical reportbacks remain historical and must not be rewritten.

---

# 5. Disposable fixture and test residue cleanup

Audit for kernel-created disposable state, including obvious disposable accounts, Storyboards/Timelines, sessions, grants, browser-proof fixtures, and temporary files outside intended evidence directories.

Requirements:

- do not delete real user content;
- prove each deletion target is disposable;
- preserve intended screenshots/logs;
- record what was cleaned;
- leave cleanup scripts safer if a bounded obvious improvement exists.

---

# 6. Clean-install / migration / bootstrap verification

## 6.1 Migration inventory

Verify:

- migration numbering/order;
- embedded migration inventory;
- no missing referenced migration;
- no duplicate migration number;
- all migrations apply from empty DB;
- current migration count matches repository truth;
- Kernels 76'83 migrations are represented correctly.

Do not infer the count from an old report.

## 6.2 Fresh test database

Run the canonical isolated reset/bootstrap process.

Prove:

- empty test DB — all migrations;
- Go-side bootstrap/seeds;
- canonical venue/content seeds;
- Blank Storyboard creation;
- Timeline creation;
- eWrite/Victory Documents required seeds;
- no production DB touched.

## 6.3 Production safety

Verify migration/test tooling still fails closed against production-like database names/URLs.

---

# 7. Full regression and browser smoke

Kernel 84 is allowed to be boring. Prove the product still works after cleanup.

## 7.1 Backend

Run the full Go suite against a freshly reset test database.

Investigate failures rather than rerunning until green.

If accumulated dirty test state causes failure, repair test isolation if bounded.

## 7.2 Static

Run at minimum:

```bash
git diff --check
gofmt checks
node --check / equivalent frontend syntax checks
```

Use stronger repository-standard commands where available.

## 7.3 Browser

Perform a focused live browser regression:

### Storyboards

- create Blank Storyboard;
- create Timeline;
- verify three-column Timeline default;
- verify Reference Panel;
- insert inward column;
- drag card;
- middle-pan;
- export.

### Coordination

- two users enter same Storyboard;
- Presence Tray appears;
- Group Leader initializes correctly;
- assign Group Leader;
- assign Current Turn;
- second client updates live;
- disconnect does not auto-reassign;
- session end clears state.

### eWrite / Documents

Run the smallest browser path that proves the modern document system still opens and resolves a real rule/document link.

### Authority

Include at least one negative browser/API proof showing an unauthorized user cannot perform a protected recent action.

---

# 8. Accessibility debt reconciliation

Kernel 84 does not need to solve every known accessibility issue.

Gather known current gaps into one canonical backlog/location.

At minimum reconcile:

- right-click Presence Tray actions lacking keyboard entry;
- occupied-cell 'Move existing' keyboard limitation;
- middle-mouse panning lacking an equivalent convenience gesture;
- already-recorded contrast/screen-reader gaps.

Requirements:

- remove duplicates;
- distinguish blocker vs convenience;
- record workaround where one exists;
- make no fake compliance claim;
- do not expand 84 into an accessibility redesign.

A trivial low-risk local fix may be included, but the kernel does not fail because the backlog remains.

---

# 9. Security/operator documentation reconciliation

Compare current repository behavior against:

- `Security-notes.md`;
- `dev-workflow.md`;
- `kernel-maker-field-guide.md`;
- `operator-log.md`;
- `operator-notes.md`;
- vendor/dependency acknowledgements where relevant;
- active current-state/implementation files.

Look for contradictions in signup mode, authentication, recovery, backup/restore, database exposure, test DB workflow, WebSocket lifecycle, venue/session semantics, and canonical roadmap authority.

Fix active docs. Preserve historical reports.

---

# 10. Canonical roadmap recovery and reconciliation

This is a primary deliverable.

## 10.1 Source-of-truth order

Use the Canonical Roadmap's own authority order:

1. repository behavior/current DB state;
2. kernel reportbacks/browser evidence;
3. operator log/current decisions;
4. Canonical Roadmap;
5. older roadmap text/plans.

## 10.2 Recover post-75 kernel sequence

Search:

- `Construction/Kernels/*`;
- `Construction/OperatorLogs/kernel-*reportback*`;
- operator log entries;
- commit history if available;
- migrations;
- feature docs tied to kernel numbers.

Build a reconciliation ledger containing:

```text
kernel number
suffix if any
exact title
date
status
reportback path
commit hash if provable
major capabilities
open findings
track impact
```

Recover through Kernel 83.

Do not silently collapse multi-pass Kernel 79 work if separate continuation specs/reportbacks exist.

Do not create phantom kernel numbers for work that was only a sub-pass unless repository convention names it as a kernel/continuation.

## 10.3 Update implementation baseline

Extend the baseline through Kernel 83.

Do not merely append titles. Summarize capability bands so future kernel makers can reason from them.

Likely grouping, subject to repository truth:

- 76'77: hosted readiness, security, private-client/recovery;
- 78'79: Victory Documents/eWrite, Library, rules linking, directories/export;
- 80'83: Storyboards, presentation, Timeline, venue coordination.

Use exact actual names/statuses.

## 10.4 Update Victory Core tracks

At minimum reconcile:

### V3 — Scenes, composition, and storyboarding

Mark actual Storyboards/Timeline capabilities now present.

Explicitly distinguish old speculative requirements intentionally not built, such as merged regions or nested cards. If no longer desired, move them to skip/defer rather than leaving them as false open requirements.

### V9 — Victory Documents

Update from 'required capabilities' to actual established capabilities proven by Kernels 78'79.

Keep genuine open items open.

### V6 / C / O

Record actual hosted-readiness/private-client/recovery/backup security progress from Kernels 76'77.

Do not overstate readiness beyond evidence.

### Coordination primitive

Record Group Leader / Current Turn in the appropriate Victory Core area.

Do not make it a game-rule track.

## 10.5 Update Anthology Track A

For A1 distinguish:

**Generic platform capability proven:**

- semantic Storyboards;
- Timeline mode;
- bands/rows/cards;
- Reference Panel;
- long navigation;
- structured export;
- live Group Leader;
- live explicit Current Turn.

from:

**Actual Microscope package not yet built:**

- exact licensed terminology/content;
- permission/license;
- complete game-specific setup/procedures;
- exact trade dress.

Do not falsely mark Microscope shipped.

Record the intentional decision to remain generic until permission/licensing is established.

## 10.6 Update skip/defer ledger

At minimum evaluate:

- nested-card/merged-cell assumptions;
- actual Microscope terminology;
- tone/light-dark hardcoding;
- participant turn order;
- automatic turn advancement;
- persisted Group Leader/Current Turn;
- user-created Storyboard templates;
- Google-Docs-grade collaborative editing;
- stale old roadmap kernel numbering;
- repaired security findings.

Categorize removed requirements as replaced, deferred, abandoned, or not planned, with reason.

## 10.7 Update open decisions

Remove decisions now resolved by product work.

Add only unresolved decisions that materially affect future planning.

## 10.8 Update committed horizon

After 84 reconciliation establish:

### Kernel 85 — Socio Sustained Play

Move Socio beyond tutorial/onboarding into repeatable connected play using current Victory primitives.

Do not fully spec 85 inside 84.

### Kernel 86 — Cartograph-Style Drawing Foundation

Build the next meaningfully different reusable creative/game primitive: shared map/drawing authoring suitable for an original or permission-safe Cartograph-style proof.

Do not fully spec 86 inside 84.

### Third committed slot

Choose only if repository evidence makes the dependency clear. Otherwise record it as decision-gated/provisional according to roadmap rules.

## 10.9 Change log

Append a dated Kernel 84 reconciliation entry describing:

- repository history recovered through 83;
- roadmap baseline brought current;
- Documents status changed;
- Storyboards/Timeline status changed;
- A1 generic-capability vs actual-package distinction;
- old Master Guide formally superseded;
- next committed horizon.

---

# 11. Master Actual Implementation Guide disposition

Locate the historical Master Actual Implementation Guide.

Add a prominent header near the top such as:

```text
STATUS: SUPERSEDED / HISTORICAL
Replaced by: Victory Canonical Roadmap v2
Do not use this file for kernel sequencing.
```

Preserve its historical content.

If repository convention supports archival relocation, moving it is acceptable only if references are updated safely.

Do not delete it merely because it is stale.

---

# 12. Current-state / implementation guide recovery

Kernel 82/83 reportbacks indicate current-state/master implementation documentation has lagged several kernels.

Find actual active file(s), including any `current-state.md` or equivalent.

Update them through Kernel 83 where they are meant to describe actual implementation.

Do not create another roadmap.

Current-state describes reality; the Canonical Roadmap controls sequence.

---

# 13. Required investigations before changing code

Before implementation, produce a short internal ledger answering:

1. What exact post-75 kernels exist?
2. Which have reportbacks?
3. Which are PASS/PARTIAL/FAIL?
4. Which missing roadmap claims can now be proven?
5. What WebSocket context is currently reused incorrectly?
6. Are other live systems using the same anti-pattern?
7. What current documentation still says password signup is available?
8. Are disposable test fixtures still present?
9. Does a clean DB migrate/bootstrap successfully now?
10. Which roadmap requirements are obsolete because product decisions changed?

Do not ask Grant to reconstruct information the repository/reportbacks can answer.

---

# 14. Required tests

## 14.1 WebSocket context regression

Add automated proof for:

- connection lasting beyond old setup timeout;
- late second `watch_board`;
- board switch;
- DB-backed snapshot after timeout;
- cleanup on old board;
- cleanup on disconnect;
- coordination registry correct after switch/end.

## 14.2 Full backend suite

Fresh isolated test DB:

```bash
TEST_DATABASE_URL=... GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
```

Use canonical reset/bootstrap guard.

## 14.3 Migration/bootstrap

Prove from empty database.

## 14.4 Browser smoke

Playwright required for recent major user-facing systems described in §7.

## 14.5 Negative authority

At least one forged/direct protected operation must fail server-side.

---

# 15. Required artifacts

At minimum:

```text
Construction/Kernels/Kernel 84 — Canonical Reconciliation & Runtime Cleanup.md
Construction/OperatorLogs/kernel-84-reportback.md
Construction/OperatorLogs/kernel-history-reconciliation-through-83.md
Construction/Domains/Operations/websocket-context-lifecycle.md
Construction/Domains/Operations/cleanup-ledger-kernel-84.md
```

Update in place where applicable:

```text
Victory Canonical Roadmap
kernel-maker-field-guide.md
operator-log.md
operator-notes.md
Security-notes.md
dev-workflow.md
current-state.md or actual equivalent
Master Actual Implementation Guide
storyboards-accessibility.md or canonical accessibility backlog
```

Do not create duplicate competing docs when a canonical file exists.

---

# 16. Required evidence

## 16.1 Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}	{{.Status}}	{{.Ports}}'
```

## 16.2 History recovery

Provide search/command evidence used to recover kernel specs, reportbacks, operator-log entries, commit hashes, and migrations where relevant.

## 16.3 WebSocket repair

Show:

- old failure/reproduction if safely reproducible;
- code path repaired;
- new regression test;
- successful >timeout board switch/watch proof.

## 16.4 Fresh install

Show:

- fresh DB reset;
- migration count;
- successful bootstrap;
- representative post-83 objects created successfully.

## 16.5 Browser

Provide Playwright assertions/screenshots for the bounded smoke.

## 16.6 Documentation

Report exact files updated and major contradictions resolved.

---

# 17. Pass criteria

Kernel 84 passes when:

- the known Storyboards WebSocket context-lifetime bug is repaired;
- bounded adjacent same-root-cause live-session issues are repaired or explicitly split;
- Storyboards late board-switch/watch works beyond the old timeout;
- disconnect/session cleanup leaves no stale watcher/coordination state;
- active testing docs no longer rely on closed password signup;
- disposable kernel test residue is verified clean;
- clean database migration/bootstrap succeeds;
- full Go suite passes on fresh isolated DB;
- recent Storyboards/Timeline/coordination browser smoke passes;
- at least one negative authority proof passes;
- accessibility debt is reconciled into one current backlog without fake completeness;
- active security/operator docs match current behavior;
- actual post-75 kernel history is recovered through Kernel 83 as far as evidence allows;
- Canonical Roadmap baseline is updated through Kernel 83;
- V3/V9/A1/C/O statuses reflect actual implementation;
- obsolete requirements are moved into skip/defer rather than left as fake open work;
- old Master Actual Implementation Guide is clearly superseded/non-canonical;
- current-state documentation is brought current;
- post-84 committed horizon points toward Socio Sustained Play, then Cartograph-style drawing unless the audit proves a blocker;
- no new competing roadmap is created;
- no unrelated product redesign occurs.

---

# 18. Partial rules

Mark PARTIAL if:

- WebSocket context bug is repaired only on disconnect but not late in-connection operations;
- clean bootstrap cannot be reproduced;
- kernel history through 83 contains unexplained gaps repository evidence should resolve;
- Canonical Roadmap remains materially stale;
- old Master Guide still appears active/canonical;
- current testing docs still prescribe closed signup;
- browser regression cannot prove recent live systems;
- current-state docs remain substantially behind.

A PARTIAL report must name a bounded continuation.

---

# 19. Fail rules

Mark FAIL if:

- cleanup causes user-visible feature regression;
- production/live data is damaged;
- test tooling touches production data;
- roadmap history is invented rather than recovered;
- old reportbacks are rewritten to manufacture consistency;
- Canonical Roadmap is replaced by yet another roadmap file;
- security controls are weakened to make testing easier;
- password signup is reopened merely for browser fixtures;
- WebSocket cleanup leaks cross-board/session state;
- authorization regresses;
- kernel expands into unrelated product feature work;
- Grant is locked out.

---

# 20. Non-goals

Kernel 84 does not:

- build Socio Sustained Play;
- build Cartograph;
- add new Storyboards features;
- add turn order;
- add initiative;
- add an accessibility redesign;
- add user-created templates;
- implement Microscope;
- implement vector drawing;
- redesign Presence Tray;
- rebuild WebSocket infrastructure from scratch;
- perform speculative refactoring;
- build SaaS;
- build billing;
- build 3D;
- rewrite historical reportbacks;
- create another roadmap.

---

# 21. Kernel-size guardrail

This kernel may touch many documentation files, but its **code repair surface must remain narrow**.

Target shape:

- one WebSocket/session lifecycle repair band;
- one bounded adjacent audit;
- one test-fixture/documentation cleanup band;
- one clean-install/regression pass;
- one canonical roadmap/history reconciliation.

If a second major code subsystem needs redesign, split it.

Do not repeat Kernel 79's mega-kernel pattern.

---

# 22. Immediate operator outcome

At completion, Grant should be able to answer:

1. Does Storyboards WebSocket still work correctly after being open longer than the old timeout?
2. Can a user switch/watch boards late in a connection without stale-context failures?
3. Does disconnect cleanup leave no stale coordination state?
4. Do current browser-testing instructions reflect closed password signup?
5. Are disposable test users/boards gone?
6. Can Victory build from a clean database through every current migration?
7. Does the full backend suite pass from clean test state?
8. Do Blank Storyboards still work?
9. Do Timelines still work?
10. Does Group Leader/Current Turn still work live?
11. Does an unauthorized action still fail?
12. Does the Canonical Roadmap now describe what actually shipped through Kernel 83?
13. Can we see the exact recovered sequence of Kernels 76'83 and continuations?
14. Are Documents/eWrite marked according to actual capability rather than the old July plan?
15. Are Storyboards/Timeline marked according to actual capability?
16. Does A1 distinguish generic Timeline capability from an actual Microscope package?
17. Are discarded Storyboards assumptions marked skipped/deferred rather than hanging open?
18. Is the old Master Actual Implementation Guide unmistakably historical?
19. Is there still only one active roadmap?
20. Is the next product direction clearly Socio Sustained Play followed by Cartograph-style drawing, unless 84 discovered a real blocker?

Kernel 84 succeeds when Victory's **runtime, evidence, and planning documents agree about what Victory actually is today**.
