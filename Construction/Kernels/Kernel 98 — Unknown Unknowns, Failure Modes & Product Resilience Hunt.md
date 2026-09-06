# Kernel 98 — Unknown Unknowns, Failure Modes & Product Resilience Hunt

**Status:** DRAFT — ready for implementation  
**Type:** Broad discovery audit + resilience proof + launch-risk triage  
**Sequence position:** After Kernel 97  
**Primary proof:** Victory’s remaining blind spots are actively hunted, classified, and either fixed or routed before release  
**Core doctrine:** Assume prior kernels, prior agents, and Grant all have blind spots. Find them deliberately.

---

## 0. Kernel mode

Kernel 98 is not another dress rehearsal.

It is not a ceremonial “run the whole product” pass.

It is not a rewrite-everything kernel.

It is a **failure hunt**.

The agent should search for things Victory’s builders may not have thought to ask about:

- hidden architectural contradictions;
- dead or superseded runtime paths;
- workflows that only work with tribal knowledge;
- stale state;
- recovery failures;
- practical performance cliffs;
- partial-save problems;
- reconnect issues;
- weird but non-malicious user behavior;
- inaccessible or undiscoverable features;
- abandoned models still influencing current behavior;
- deployment/runtime assumptions that only work on the original installation.

The kernel should produce useful surprises.

---

## 1. Launch-blocker definition

Treat the following as launch blockers unless proven otherwise:

- data loss;
- data corruption;
- privacy/security failure not already contained by K96;
- authority failure not already contained by K97;
- inability to install, start, join, or play;
- repeatable crash;
- broken save/restore;
- unrecoverable stale state;
- major workflow dead end;
- required tribal knowledge with no documented recovery;
- severe cross-user state contamination;
- broken reconnect/restart behavior that destroys active work;
- user actions that silently fail while appearing successful.

Do not classify cosmetic inconsistency as a launch blocker unless it makes the product misleading or unusable.

---

## 2. Fix versus route

K98 should not become an endless repair marathon.

Use this policy:

### Fix now
- CRITICAL findings;
- HIGH findings;
- bounded launch-blocking defects;
- obvious MEDIUM defects that are cheap and low-risk to fix.

### Route
If a finding is real but broad, classify and route it to:

- K99 — fresh install/docs;
- K100 — consumer packaging;
- K101 — release bug burn/polish;
- K102+ — post-launch capability/architecture.

Do not silently expand K98 into another architecture kernel.

---

## 3. Discovery categories

Search across at least these categories:

- data integrity;
- runtime resilience;
- state synchronization;
- stale state;
- reconnect/restart;
- dead code / dead routes;
- duplicate truths;
- abandoned models;
- inconsistent defaults;
- undiscoverable features;
- incomplete workflows;
- error recovery;
- performance cliffs;
- asset handling;
- long-running session/show behavior;
- cross-tab behavior;
- browser refresh behavior;
- multi-user contention;
- migration assumptions;
- environment/config assumptions;
- observability;
- operator recovery;
- partial failures;
- cleanup/orphan behavior;
- historical compatibility;
- frontend/backend disagreement.

---

## 4. Dead code and abandoned models

K98 should inspect both runtime behavior and repository history/current architecture.

Search for:

- routes no longer linked from UI;
- handlers with no legitimate callers;
- old models superseded by newer authority/state models;
- legacy session/persona paths;
- old access-grant semantics;
- stale feature flags;
- unused columns/tables still affecting behavior;
- duplicated state written to two places;
- old frontend modules still loaded;
- abandoned CSS/JS bundles;
- migration-era compatibility branches that never retire.

Do not delete aggressively.

Classify each finding as:

- safe dead code;
- compatibility path;
- suspicious duplicate truth;
- still-active legacy behavior.

Only remove when confidence is high and proof is bounded.

---

## 5. Tribal knowledge hunt

Explicit question:

> “What only works because Grant already knows how Victory works?”

Look for:

- undocumented setup steps;
- hidden right-clicks;
- magic order-of-operations;
- obscure slash commands;
- browser-refresh rituals;
- manual database assumptions;
- hidden file locations;
- hard-coded environment knowledge;
- special sequence for creating Shows/Showings;
- special sequence for entering venues;
- things that only work after visiting another screen first.

If the behavior is legitimate but undocumented, route to K99.

If the behavior is accidental, fix or classify.

---

## 6. Undiscoverable but valid features

Find features that technically exist but ordinary users are unlikely to discover.

Examples:

- hidden context menu actions;
- unlabeled icons;
- required right-click behavior;
- useful Guide content nobody can find;
- commands with no discoverability path;
- setup actions buried in unrelated venues;
- role tools with weak affordance.

K98 should not redesign them all.

Record:

- launch blocker;
- usability debt;
- K101 polish;
- K102+ redesign.

---

## 7. Weird user pass

Test users who are not malicious, just unpredictable.

Examples:

- click things out of order;
- open multiple tabs;
- refresh repeatedly;
- close browser mid-action;
- leave a venue mid-edit;
- reconnect later;
- change Character at odd moments;
- start an upload then leave;
- open stale pages from history;
- keep two different Shows open;
- return a week later;
- click disabled-looking things;
- double-click actions;
- submit twice;
- navigate away during save;
- switch roles/context without reloading.

The goal is not to punish weird users.

The goal is to ensure ordinary weirdness does not corrupt shared state.

---

## 8. Minimal human testing

Human testing should be **minimal**.

Grant should not be turned into the primary test harness.

The agent should:

1. automate/mechanically reproduce as much as possible;
2. use browser proof where needed;
3. bring Grant only findings requiring human experiential judgment;
4. bundle those into a short review set.

Do not ask Grant to perform repetitive role-by-role or venue-by-venue walkthroughs.

Do not recreate K93.

---

## 9. Browser refresh resilience

Test refresh during:

- live Show;
- Scene editing;
- Storyboards editing;
- Character editing;
- tray state;
- Director prep;
- audience admission;
- map/stage interaction;
- drawing;
- dice state;
- chat;
- Guide state;
- asset upload;
- Showtime start/end flow.

After refresh:

- authoritative state should restore;
- local-only state may restore where intended;
- no duplicate actions should fire;
- no stale phantom state should remain.

---

## 10. Multi-tab behavior

Test same account in two browser tabs.

Look for:

- duplicated WebSocket subscriptions;
- conflicting selected Character state;
- stale role UI;
- duplicated messages;
- double-started Showtime;
- duplicate saves;
- race conditions;
- local-storage collisions;
- tray-state oddities;
- conflicting Scene edits.

Do not guarantee perfect collaborative editing unless product intends it.

Do prevent obvious corruption.

---

## 11. Backend restart resilience

Restart backend during realistic activity.

Observe:

- reconnect;
- session validity;
- WebSocket recovery;
- client error state;
- retry behavior;
- Scene state;
- stage state;
- unsaved edits;
- message flow;
- dice;
- presence.

A backend restart should not silently destroy durable work.

If transient live state is intentionally lost, make that behavior explicit.

---

## 12. Database interruption

If safely reproducible in local/test environment:

- briefly interrupt DB connectivity;
- observe request failures;
- restore DB;
- verify recovery.

Check for:

- partial writes;
- misleading success UI;
- stuck transactions;
- stale client optimism;
- duplicate retry writes.

Do not conduct destructive chaos testing against production.

---

## 13. Partial save / interrupted write

Target multi-step operations.

Examples:

- save Scene + placements;
- asset metadata + file;
- Storyboards structural edits;
- Character state;
- Showtime start;
- Showing creation;
- Director prepared play;
- audience configuration.

Ask:

> “If step 2 fails after step 1 succeeds, what remains?”

Prefer transactional behavior where integrity matters.

Where full transactions are impractical, ensure retry/recovery is safe.

---

## 14. Duplicate submission

Attempt:

- double-click create;
- repeated POST;
- retry after timeout;
- browser resend;
- two tabs performing same action.

Check for duplicate:

- Shows;
- Showings;
- Characters;
- messages;
- Scene saves;
- tickets/admissions;
- assets;
- Storyboard cards;
- stage elements.

Use idempotency where appropriate.

Do not over-engineer every form.

---

## 15. Long-running Show behavior

Simulate or test prolonged use.

Look for:

- memory growth;
- unbounded event accumulation;
- WebSocket degradation;
- presence drift;
- stale participants;
- timer/state drift;
- duplicate subscriptions;
- client slowdown;
- oversized DOM/state stores.

Do not optimize hypothetical week-long sessions unless evidence shows a real issue.

---

## 16. Dense-board performance

Use realistic stress cases for:

- Storyboards;
- Timeline;
- stage objects;
- drawing objects;
- token counts;
- chat history;
- message lists;
- asset-heavy maps.

Find practical thresholds.

Classify:

- acceptable;
- degraded but usable;
- launch concern;
- architectural future work.

Do not chase benchmark vanity metrics.

---

## 17. Asset-heavy map behavior

Test:

- many token assets;
- large map image;
- multiple overlay elements;
- repeated enter/leave;
- cache invalidation;
- removed/replaced assets;
- broken asset URLs.

Check for:

- memory leaks;
- render failure;
- stale images;
- missing cleanup;
- layout collapse.

---

## 18. Reconnect storm

Simulate several users disconnecting/reconnecting around the same time.

Observe:

- presence;
- duplicate sockets;
- duplicate subscriptions;
- stale users;
- message replay;
- stage state;
- Audience projection;
- Show membership.

A reconnect storm should not create phantom participants.

---

## 19. Frontend/backend disagreement

Search for places where:

- frontend assumes one state;
- backend stores another;
- API names imply outdated semantics;
- UI labels disagree with canonical domain model;
- frontend defaults differ from server defaults.

Examples may include:

- Scene;
- Show;
- Showing;
- Character;
- cohort;
- Audience;
- role;
- visibility;
- selected Character.

Prefer server-authoritative truth.

---

## 20. Default-value audit

Audit default values for:

- visibility;
- role;
- audience controls;
- Scene fields;
- health/status projection;
- tray state;
- dice visibility;
- Show creation;
- Showing creation;
- asset permissions;
- Storyboards structure.

Ask:

> “If the user does nothing, is the default safe and sensible?”

Unexpected defaults are a common unknown-unknown source.

---

## 21. Empty-state audit

Test empty installations / empty data for:

- campus;
- Shows;
- Showings;
- Characters;
- Storyboards;
- assets;
- messages;
- My People;
- Producer Office;
- Director prep;
- First Theater;
- Catharsis;
- Guide.

No empty state should produce:

- crash;
- blank unexplained screen;
- impossible first action;
- stale fixture data.

---

## 22. Orphan cleanup

Search for orphaned:

- files;
- assets;
- stage objects;
- Scene placements;
- messages;
- memberships;
- audience admissions;
- Storyboard elements;
- cohort links;
- Character references.

Do not delete legitimate history blindly.

Identify whether cleanup is:

- automatic;
- operator-managed;
- migration-managed;
- missing.

---

## 23. Migration assumptions

Review migrations for:

- order dependencies;
- non-idempotent assumptions;
- data-dependent failures;
- production-only assumptions;
- old schema compatibility;
- fresh-install behavior;
- rollback hazards;
- seed dependencies.

K99 will prove fresh install fully.

K98 should identify suspicious migration behavior early.

---

## 24. Environment/config assumptions

Search for assumptions about:

- hostnames;
- ports;
- filesystem paths;
- HTTPS;
- localhost;
- Discord;
- database name;
- container name;
- Caddy/reverse proxy;
- Linux-only paths;
- browser origin;
- production base URL.

Classify anything that will matter to K99/K100.

---

## 25. Observability

Ask:

> “When something breaks, can the Operator tell what happened?”

Review:

- logs;
- health endpoints;
- startup diagnostics;
- migration failures;
- DB failures;
- WebSocket failures;
- upload failures;
- auth failures;
- asset failures.

Do not build enterprise observability.

Do ensure errors are diagnosable without exposing secrets.

---

## 26. Silent failure hunt

Search for UI actions that:

- appear to succeed but fail;
- fail without visible feedback;
- swallow exceptions;
- only log to console;
- leave stale state;
- require refresh to reveal failure.

Silent failure is higher priority than ugly failure.

---

## 27. Recovery path audit

For major workflows, ask:

> “If this fails halfway, what does the user do next?”

Test recovery for:

- Showing creation;
- Showtime;
- Scene save;
- Character edit;
- Storyboards edit;
- asset upload;
- map changes;
- Director prep;
- audience admission;
- role changes.

If the answer is “Grant knows how to fix the DB,” classify it.

---

## 28. Compatibility with old data

Use an older/long-lived database snapshot or representative seeded state where available.

Look for:

- old rows missing new fields;
- stale enum/string values;
- historical venue config;
- legacy assets;
- old session data;
- migration assumptions;
- deprecated role fields.

Do not mutate production data during discovery.

---

## 29. Fresh-state versus legacy-state comparison

Compare:

- fresh DB behavior;
- long-lived DB behavior.

Any feature that only works on one side deserves investigation.

This is especially important for:

- seed venues;
- operator/bootstrap state;
- Character defaults;
- Show defaults;
- permissions;
- Storyboards;
- asset availability.

---

## 30. Dependency/runtime surprises

Look for:

- frontend package only available through dev tooling;
- runtime CDN dependency;
- missing production build artifact;
- browser-specific feature assumptions;
- native dependency hidden in dev environment;
- system package requirement not documented.

Route install/package implications to K99/K100.

---

## 31. Browser compatibility sanity

Do a bounded sanity check on the browsers Victory realistically expects at launch.

At minimum:

- current Chromium-family browser;
- current Firefox-family browser.

If Windows packaging embeds or recommends a browser later, K100 owns that decision.

K98 should only identify glaring incompatibilities.

---

## 32. Permission-denied UX

When a user is correctly denied:

- error should be understandable;
- UI should not imply success;
- no private detail should leak;
- user should know whether the action is impossible or merely unavailable in this context.

K97 owns authority correctness.

K98 checks failure experience.

---

## 33. Race conditions

Target obvious concurrent actions:

- two Directors editing same Scene;
- two users moving same object;
- two users editing same Storyboard structure;
- simultaneous Showing edits;
- simultaneous role changes;
- simultaneous Character updates.

Do not build CRDT infrastructure.

Find and contain obvious last-write corruption or impossible state.

---

## 34. Time/date edge cases

Audit:

- timezone handling;
- DST;
- showing timestamps;
- “today”/upcoming classification;
- midnight rollover;
- historical buckets;
- scheduler labels.

Use real date/time tests.

Do not redesign scheduling.

---

## 35. Unicode and strange input

Test:

- emoji;
- non-Latin names;
- long names;
- combining characters;
- punctuation;
- quotes/apostrophes;
- RTL text if rendering naturally supports it;
- very long descriptions;
- empty strings.

Look for crashes, truncation bugs, layout breakage, or unsafe assumptions.

---

## 36. File/path edge cases

Test safe asset names with:

- spaces;
- unicode;
- duplicate names;
- long names;
- reserved-looking characters.

Storage should not depend on display filename.

K96 owns hostile upload security.

K98 checks ordinary weird filenames and recovery.

---

## 37. Feature debt classification

Every finding should end in one bucket:

- **BLOCKER — fix before release**
- **K99 — fresh install/docs**
- **K100 — packaging/distribution**
- **K101 — polish/bug burn**
- **K102+ — post-launch architecture/capability**
- **ACCEPTED — intentional behavior**
- **DEAD — safe to retire**
- **UNKNOWN — requires targeted follow-up**

Do not leave a giant unclassified issue dump.

---

## 38. Human checkpoint rule

Do not stop Grant for routine findings.

Only interrupt for:

- destructive choice;
- product semantics ambiguity;
- user-visible behavior where multiple valid directions exist;
- experiential judgment the agent genuinely cannot make.

Bundle questions.

Avoid one-question-at-a-time interruption.

---

## 39. Agent work discipline

Do not spend hours proving one low-severity edge case.

Use timeboxes.

When a path is clearly low-value:

- record;
- classify;
- move on.

K98 succeeds by breadth plus good triage, not exhaustive perfection.

---

## 40. Required outputs

Produce:

1. a findings ledger;
2. severity;
3. reproducibility;
4. affected subsystem;
5. fix/routing decision;
6. evidence;
7. residual risk;
8. dead-code/duplicate-truth list;
9. tribal-knowledge list;
10. resilience matrix;
11. performance threshold notes;
12. K99/K100/K101/K102+ carry-forward list.

---

## 41. Resilience matrix

At minimum record behavior for:

| Scenario | Expected | Actual | Result |
|---|---|---|---|
| Browser refresh | durable state restored | | |
| Multi-tab | no corruption | | |
| Backend restart | reconnect/recover | | |
| DB interruption | safe failure/recovery | | |
| Duplicate submit | no harmful duplicate | | |
| WebSocket reconnect | no phantom state | | |
| Partial upload | no corrupt asset | | |
| Interrupted save | clear recovery | | |
| Long-running Show | stable enough | | |
| Dense board | usable threshold known | | |

---

## 42. Launch blocker policy

K98 cannot call PASS with an unresolved BLOCKER.

If a BLOCKER is too broad to fix inside K98:

- stop;
- explain scope;
- propose exact next kernel routing;
- mark PARTIAL.

Do not relabel blockers as “future work” merely to finish the kernel.

---

## 43. Decision memory

Locked product decisions:

- K98 should deliberately hunt unknown unknowns.
- Data loss, corruption, privacy/security failure, authority failure, inability to install/start/join/play, repeatable crashes, broken save/restore, major workflow dead ends, and unrecoverable tribal-knowledge dependencies are launch blockers.
- CRITICAL/HIGH and bounded launch blockers should be fixed in K98.
- Obvious cheap MEDIUM findings may be fixed.
- Broader findings should be routed rather than turning K98 into an endless rewrite.
- K98 should inspect both runtime behavior and dead/abandoned repository paths.
- Undiscoverable-but-valid features are in scope for discovery/classification.
- Realistic performance/scalability thresholds are in scope.
- Recovery/resilience is in scope.
- Weird non-malicious user behavior is in scope.
- Human testing should be minimal.
- Grant should only be pulled in for genuine experiential/product ambiguity.

---

## 44. Pass criteria

K98 passes when:

- broad blind-spot discovery has been performed;
- launch blockers found are fixed or explicitly force PARTIAL;
- dead/legacy/duplicate-truth paths are classified;
- tribal-knowledge dependencies are identified;
- weird-user scenarios have been exercised;
- refresh/reconnect/restart behavior is known;
- partial-save and duplicate-submit behavior is known;
- realistic dense-state thresholds are known;
- fresh-state versus legacy-state differences are known;
- silent failures are materially reduced;
- recovery paths exist or are routed;
- install/package findings are routed to K99/K100;
- polish findings are routed to K101;
- capability/architecture findings are routed to K102+;
- Grant was not used as the repetitive test harness.

---

## 45. Reportback

Report:

1. total findings by severity;
2. launch blockers found;
3. launch blockers fixed;
4. unresolved blockers;
5. dead-code findings;
6. duplicate-truth findings;
7. tribal-knowledge findings;
8. undiscoverable-feature findings;
9. weird-user findings;
10. browser refresh findings;
11. multi-tab findings;
12. backend restart findings;
13. DB interruption findings;
14. partial-save findings;
15. duplicate-submit findings;
16. WebSocket reconnect findings;
17. performance threshold findings;
18. asset-heavy map findings;
19. fresh-versus-legacy findings;
20. environment/config assumptions;
21. silent failures;
22. recovery gaps;
23. migration concerns;
24. observability concerns;
25. routed K99 items;
26. routed K100 items;
27. routed K101 items;
28. routed K102+ items;
29. minimal human questions/spot-check result;
30. final risk assessment.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

---

## 46. Completion condition

Kernel 98 passes when Victory has been challenged not just with the questions its builders already knew to ask, but with the uncomfortable possibility that the builders missed something.

The goal is not to prove perfection.

The goal is to make surprises smaller, rarer, and better classified before strangers depend on the software.
