# Audit Kernel — Recovered Work Reconciliation & Release Prioritization

**Status:** READY FOR AUDIT  
**Type:** Forensic reconciliation / release-prioritization audit  
**Implementation:** NONE  
**Primary input:** Current Victory repository plus the agent’s existing `Construction/Everything Implemented.md`

## 0. Kernel contract

You already produced `Construction/Everything Implemented.md`, which answered:

> What does Victory actually implement now?

This audit answers:

> What important work did earlier Victory kernel makers knowingly leave unfinished, defer, bound narrowly, work around temporarily, or expect later kernels to finish — and which of those obligations still matter today?

Do **not** rely on separate handoff summaries. Recover the historical intent yourself from Victory’s own evidence:

- kernel specs;
- kernel reportbacks;
- Construction documents;
- operator logs and notes;
- current-state documents;
- Git history;
- migrations;
- tests;
- current source.

Then reconcile that recovered intent against the current repository and `Everything Implemented.md`.

The goal is not a larger backlog. The goal is:

> many historical deferrals collapsed into the smallest evidence-backed set of current release gaps.

Create:

`Construction/Recovered Work Reconciliation.md`

Do not implement features. Do not create future kernels. Do not create a new roadmap. Do not commit unless explicitly instructed.

---

# 1. Source priority

Use:

1. current code and database schema;
2. current tests and executable behavior;
3. current working tree, including identified uncommitted work;
4. current reportbacks;
5. `Construction/Everything Implemented.md`;
6. current-state / operator logs;
7. Canonical Roadmap;
8. historical kernel specs/reportbacks;
9. superseded plans.

Historical records are authoritative for **what was intended, bounded, deferred, or knowingly left behind**. They are not automatically authoritative for what is still missing. Later kernels may have solved the issue.

---

# 2. Recover historical unfinished work

Work backward through recoverable kernel history.

For each kernel or coherent kernel era, inspect:

- specification;
- reportback;
- known issues;
- deviations;
- non-goals;
- deferred/follow-up sections;
- operator notes;
- current-state entries;
- later kernels that superseded or consumed it.

Pay particular attention to terms such as:

`deferred`, `out of scope`, `later kernel`, `future`, `follow-up`, `not implemented`, `bounded`, `temporary`, `workaround`, `compatibility`, `placeholder`, `proof`, `proving ground`, `seed-authored`, `developer-authored`, `API-only`, `HTTP-only`, `not generalized`, `known limitation`, `partial`, `fallback`, `legacy`, `superseded`.

Read enough context to recover the intended eventual capability. Do not merely copy keyword hits.

Include only evidence-backed inherited work such as:

- missing capability;
- bounded proof that was meant to generalize later;
- backend/runtime capability lacking usable Director/player controls;
- temporary architecture or compatibility path;
- expected integration seam;
- known correctness/hardening debt;
- clear historical promise later development may have forgotten.

Exclude speculative features, random TODOs, abandoned roadmap ideas, and “other VTTs have this” assumptions.

---

# 3. Classification

Every recovered item gets exactly one:

### CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT
Later implementation genuinely fulfills the old intent.

### CLOSED — SUPERSEDED BY BETTER CURRENT MODEL
A later design solved the underlying need differently.

### PARTIAL — SMALL BOUNDED GAP
Most capability exists; remaining work is a direct repair/continuation.

### PARTIAL — PRODUCTIZATION GAP
Canonical/backend/runtime behavior exists but usable Director/player controls are thin.

### OPEN — REQUIRED BEFORE FIRST RELEASE
Victory 1.0 would be meaningfully broken, misleading, insecure, or unable to support intended launch play without it.

### OPEN — IMPORTANT BUT NOT RELEASE-BLOCKING
Strong pre-100 candidate, but Victory could credibly launch without it.

### DEFER — POST-RELEASE
Useful or historically intended, but should not consume remaining pre-freeze feature kernels.

### RETIRE — LEGACY CONCEPT
An old path/concept should no longer influence new work, though compatibility may temporarily remain.

For every non-closed item record:

```markdown
### [Capability]
**Classification:**
**Historical source(s):**
**What was intentionally left:**
**Why it was left:**
**Intended eventual capability:**
**Current implementation:**
**What is actually still missing:**
**Modern foundation to build on:**
**Dependencies:**
**Release importance:**
**Estimated scope:** XS / S / M / L
**Likely consolidation group:**
**Evidence:**
```

Scope:
- XS = direct repair/polish
- S = bounded continuation
- M = normal kernel-sized feature
- L = too broad/dangerous as one kernel

Do not estimate hours.

---

# 4. Audit bands

Use these bands to organize the recovery. The listed items are audit questions, not assumed missing features.

## BAND A — Map, grid, and spatial tabletop

Recover and verify historical intent around:

- grid;
- physical map scale;
- units;
- ruler;
- straight-line distance;
- path distance;
- diagonal policy;
- square/hex/gridless behavior;
- map coordinates;
- token movement;
- pan/zoom;
- Cave-era versus Pixi-era math.

Explicitly determine whether Victory currently has:

`grid → scale → unit → ruler → path distance → diagonal rule`

If the grid remains only visual/configurational, say so.

Determine whether all new spatial work should converge on the shared Pixi runtime and whether any old DOM/Cave path still matters behaviorally.

## BAND B — Visibility, hidden information, and projection

Recover intent around:

- reveal/hide;
- Director-only objects;
- selected-player visibility;
- Cohort visibility;
- Audience segmentation;
- fog;
- explored/unexplored regions;
- participant-local projection;
- stage projection;
- role-filtered views.

Determine whether current special cases have converged into one reusable visibility primitive or remain fragmented.

Test the doctrine:

> one canonical Scene/state, projected differently to authorized viewers.

Do not turn participant-local projections into alternate Scene state.

Treat generalized object visibility and spatial fog/exploration as separate unless current architecture genuinely unifies them.

## BAND C — Scene-element state and Cues

Recover whether Scene elements were expected to support bounded canonical state:

- visible/hidden;
- enabled/disabled;
- open/closed;
- revealed/unrevealed;
- interaction enabled/disabled.

Verify current Cue capabilities and whether deferred actions such as:

`GO → reveal/hide → enable/disable interaction → bounded state change`

are already present or still missing.

Do not invent a scripting language.

## BAND D — Director authoring and productization

Search for the recurring pattern:

`backend exists → runtime exists → tests exist → Director creation is thin`

Audit:

- Scene composition;
- update current Scene;
- Save as New Scene;
- Scene recall;
- Cue authoring;
- participant interactions;
- merchant packets;
- dialogue packets;
- clickable doors/clues;
- choice prompts;
- freeform submissions;
- equipment interactions;
- bindings;
- rule links;
- visibility/state bindings;
- stage effects.

Ask:

> Can Grant actually create/configure this through Victory without editing seeds, SQL, source, or raw HTTP?

Look for ways several seed/developer-authored proofs may collapse into one bounded Director interaction-authoring primitive.

## BAND E — Identity, participation, roles, and access

Recover early generations of:

- raw Cave/session join;
- Location membership;
- Production authority;
- Show Run roster;
- tickets/invites;
- session personas;
- site-wide active Character;
- Show-selected Character;
- Operator;
- Producer;
- access grants;
- memberships.

Audit:

- whether old proof paths can still create meaningful public authority;
- whether Location membership is mistaken for Show participation;
- whether legacy access grants can bypass modern participation;
- whether session persona/site active Character still act as authority;
- whether Operator/dev escape hatches leak into ordinary product behavior.

Do not delete compatibility paths for purity. Find actual behavioral risk.

## BAND F — Character participation identity

Determine the canonical answer to:

> Who is this user acting/speaking as in this Show?

Audit:

- stage tokens;
- Character sheets;
- skills;
- dice;
- Game Status;
- Socio;
- Cohorts;
- Greenroom;
- First Theater;
- Catharsis;
- chat;
- In Character behavior.

Classify legacy persona/site-active-Character use as compatibility, ambiguity, authority problem, or harmless preference/default.

## BAND G — Dice and roll authority

Reconcile historical dice deferrals against current 86/86A behavior.

Verify:

- server RNG;
- `/roll`;
- dice tray;
- Character skill clicks;
- Cast/Player roll authority;
- Show/Cohort/Director/Private visibility;
- snapshot/history filtering;
- spatial projection;
- static/pinned dice;
- announcement timing.

### Explosion semantics

Separate:

1. parser behavior;
2. expression/macro construction;
3. renderer behavior.

Desired product rule:

> Dice do not unexpectedly explode unless the relevant game operation/macro explicitly requests exploding behavior.

If parser only explodes when `!` is present, do not call the parser broken. If skill-click code always inserts `!`, determine whether that is intended Socio policy or historical convenience.

### Player roll authority

Verify whether an ordinary Cast/Player controlling a valid Character can submit canonical rolls.

Do not assume.

### Presentation quality

Separate functional correctness from visual quality. If dice are technically correct but visibly cheap, classify that honestly.

Do not implement a renderer in this audit.

## BAND H — Socio rules-native play

Audit current reusable support for:

- current Social Stance;
- Stance change;
- stance wheel/relationships;
- stance-aware actions;
- Major Action;
- Minor Action;
- Reactions;
- action spending/reset;
- HP pools;
- statuses;
- Fate/EP if canonical;
- skills;
- contextual modifiers;
- Director assistance;
- equipment;
- barter;
- inventory;
- Kessa;
- Ra;
- locked door;
- Cohorts;
- Game Status;
- Story So Far;
- aftercare.

Distinguish:

`Kessa has stance choices`

from:

`Victory has a reusable Socio Stance system`.

Likewise distinguish one merchant from generic economy automation.

Do not turn Socio into a CRPG rules executor.

Explicitly answer:

> Can two Players and a Director run a short genuine Socio encounter outside the seeded tutorial using native Victory surfaces?

Name only concrete blockers.

## BAND I — Communication lanes

Recover the intended separation between:

- stage speech;
- venue/OOC chat;
- durable notes/mail;
- backstage/Director communication;
- Discord bridge.

Audit whether the lanes remain coherent and understandable.

### In Character chat

Check whether current code includes a generic `In Character` chat tab.

If missing, evaluate the desired bounded behavior:

- server resolves current Show-selected/active Character;
- Character displays as speaker;
- accountable user remains stored;
- client cannot impersonate arbitrary Characters;
- switching Character affects future IC messages.

Classify it as productization if the existing chat/identity systems already support it.

## BAND J — Show lifecycle, rehearsal, live, Session, Showing

Recover early intent around:

- Session;
- Show;
- Show Run;
- Showing;
- Scene;
- rehearsal;
- live;
- closed/review.

Audit whether mode changes server-enforced behavior for:

- structural edits;
- recording/history;
- audience projection;
- Scene mutation;
- Director actions;
- close/resume.

Find actual old handlers that still behave as though active Session is the canonical performance object. Do not refactor naming-only debt.

## BAND K — Story So Far and durable continuity

Do not mark Story So Far missing if it exists.

Audit whether newer meaningful systems emit appropriate biography-level events where historically intended, such as:

- significant Scene completion;
- major status;
- important item gain/loss;
- advancement;
- milestone;
- major Character choice;
- relationship event.

Biography, not telemetry.

Also ensure ephemeral presence is not being used as durable attendance/history truth.

## BAND L — Storyboards / Timeline hardening

Verify historical/current issues such as:

- structural `sort_order` race;
- protected Ending append loophole;
- boundary insertion;
- concurrency;
- keyboard access to Presence/coordination menus.

Classify as correctness, bounded hardening, accessibility, or polish unless evidence says otherwise.

## BAND M — Library, assets, placement, and promotion

Recover the early doctrine:

> reusable canonical asset is not its placement in a Scene.

Verify modern systems preserve it.

Search for historical intent like:

`session-created artifact → promote_to_library → Producer approval`

Determine whether this exists, was superseded, remains useful, or should be abandoned.

Also determine whether release genuinely needs generalized upload/media library/lifecycle/licensing capability.

Do not resurrect old ideas merely because they existed in a contract.

## BAND N — Cards, hands, transfers, and commerce

Distinguish:

- editorial Index Cards;
- Storyboard cards;
- equipment inventory;
- player hand/deck mechanics.

Determine whether launch requires:

- private hands;
- decks;
- card zones;
- player-to-player item/card transfer;
- generic commerce ledger.

If current launch productions do not require them, defer.

## BAND O — Audience

Recover the distinction:

`Audience watching → Audience bounded interaction → Player participation`

Audit whether first release actually needs voting, applause, polls, bounded Cue selection, or other Audience interaction.

Never solve this by granting Audience Player authority.

## BAND P — Accessibility, responsiveness, and usability

Search reportbacks for deferred:

- keyboard access;
- focus behavior;
- mouse-only essential controls;
- responsive layouts;
- labels;
- reduced-motion needs;
- fullscreen/theatrical usability.

Find concrete release-impacting issues. Do not turn this into a theoretical certification exercise.

## BAND Q — Operational/release integrity

Recover and verify:

- recovery email/provider state;
- K85 smoke cleanup hazard;
- eWrite raw DB-error leakage;
- migration anomalies;
- reportback/commit hygiene;
- uncommitted kernel work;
- fresh bootstrap;
- backup/restore;
- deletion/export;
- security regressions.

Separate application defects, operational blockers, external dependencies, and harmless bookkeeping oddities.

---

# 5. Legacy concepts to test for retirement/containment

Verify whether any of these still influence current product behavior:

- raw Cave session join as meaningful admission;
- active Cave session as product center;
- `current_session_personas` as authority;
- site-wide active Character as Show authority;
- legacy memberships/access grants as primary authority;
- presence as durable participation;
- Operator/dev routes as product mechanisms;
- DOM-era stage math;
- Index Card as universal card abstraction;
- Session as canonical Show/performance object.

For each surviving concept classify:

- harmless compatibility;
- contain before release;
- retire after release;
- active release/security risk.

---

# 6. Deduplicate recovered work

This is a primary deliverable.

Historical kernels may describe one missing primitive many ways.

Investigate whether these consolidate:

### Visibility convergence

`object visibility + Director-only + Cohort + selected-player + reveal/hide + Cue reveal + interaction enable/disable`

Do not force fog into the same primitive if region exploration is genuinely separate.

### Interaction authoring

`merchant + NPC + dialogue packet + clue + door + choice prompt + freeform submission + equipment interaction + Cue trigger`

### Participation identity

`session persona + site active Character + Show-selected Character + IC speaker + Character-controlled roll`

### Measured tabletop

`grid scale + unit + ruler + path measurement + diagonal rule`

### Director control productization

`Scene composition + save/recall + Cues + bindings + visibility + interaction configuration + stage effects`

Do not consolidate where architecture genuinely differs.

---

# 7. Release scoring

For every remaining consolidated item, score 0–3:

- User impact
- Frequency of use
- Release expectation
- Security/data-integrity risk
- Architectural dependency
- Product differentiation
- Existing implementation leverage

Also assign Complexity: XS / S / M / L.

Do not mechanically rank by sum. Explain judgment.

---

# 8. Priority tiers

Place every remaining consolidated item into:

## Tier 0 — Must fix before continuing feature work
Only true corruption/security/current-build/canonical-state blockers.

## Tier 1 — Must exist before Victory 1.0
Absence would make launch meaningfully incomplete, misleading, unsafe, or unable to support intended play.

## Tier 2 — Strongly desirable before Kernel 100
High-value finishing/productization work.

## Tier 3 — Polish / hardening
Important release quality work, not architectural feature scope.

## Tier 4 — Explicitly post-release
Do not consume pre-100 feature capacity.

---

# 9. Functional completion versus release quality

Do not repeat:

> code exists, therefore finished.

For important user-facing capabilities distinguish:

- functional;
- authorable;
- usable;
- visually acceptable;
- release-quality.

Examples:

- spatial dice can function while looking cheap;
- Scene capture can have routes/tests but weak Director UX;
- Storyboards can function with structural/accessibility roughness;
- Socio HP/status can exist while stance/action play is still absent.

Do not inflate polish into architecture, but do not dismiss visibly unshippable presentation as irrelevant.

---

# 10. Cross-check Everything Implemented

For every recovered request:

1. locate the matching entry in `Everything Implemented.md`;
2. verify it against current code;
3. decide whether its classification remains accurate.

Create:

## Corrections to Everything Implemented

Only list real factual corrections discovered here. Do not rewrite the whole file.

---

# 11. Authoring asymmetry table

Create:

| Capability | Runtime exists? | Backend exists? | Director UI exists? | Usable without code? | Release importance |
|---|---:|---:|---:|---:|---|

Focus on:

- Scenes;
- Cues;
- participant interactions;
- merchants;
- equipment;
- visibility;
- Scene state;
- stage effects;
- Socio encounter setup.

This is a major release concern now.

---

# 12. Candidate bundles

The project intends to stop feature development at Kernel 100. Kernel 101 is for bugs/polish/regression/release convergence, not broad feature expansion.

Propose **candidate bundles**, not kernel specs and not final numbers.

Prefer:

`finish existing primitive + expose usable controls + prove in real play`

over:

`invent another foundation`

Example:

```markdown
### Candidate Bundle — Measured Tabletop
Includes:
- physical map scale
- unit
- ruler
- path distance
- diagonal policy

Existing foundation:
- shared Pixi runtime
- grid configuration

Explicitly excludes:
- initiative
- movement automation
- fog
```

Do not assume each historical request deserves a kernel.

---

# 13. Smallest credible release

Explicitly answer:

> If Victory freezes broad feature development near Kernel 100, what is the smallest credible set of additional capabilities required to ship publicly as a serious VTT?

Support the answer from repository/history evidence.

Also explicitly assess whether any everyday launch-critical VTT capability is absent, including:

- measured maps;
- token movement;
- hidden information;
- player rolls;
- Character identity;
- chat;
- Scene switching;
- rules/handouts;
- inventory;
- Director control;
- reconnect;
- permissions;
- save/restore.

Do not add conventional features merely because other VTTs have them.

---

# 14. Explicit do-not-build-before-100 section

Create:

## Explicit Post-Release / Do-Not-Build-Before-100

Include historical ideas that remain interesting but should not consume feature-freeze capacity unless repository evidence shows a launch dependency.

Potential examples:

- generic gamification/XP;
- full Audience participation product;
- generic deck/hand engine;
- broad asset marketplace/library;
- analytics;
- open-ended stage-effects authoring;
- full fog if launch does not need it;
- arbitrary scripting;
- universal CRPG-style Socio automation.

This section is required so historical archaeology does not create scope expansion.

---

# 15. Required output structure

`Construction/Recovered Work Reconciliation.md` must contain:

1. Executive Summary
2. Repository State Audited
3. Method and Source Priority
4. Historical Recovery Index
5. Recovered Work Reconciliation Matrix
6. Detailed Findings by Band A–Q
7. Consolidated Missing Primitives
8. Closed / Already Solved Historical Requests
9. Legacy Concepts to Retire or Contain
10. Corrections to Everything Implemented
11. Authoring Asymmetry Table
12. Tier 0 — Immediate Blockers
13. Tier 1 — Required for Victory 1.0
14. Tier 2 — Strong Pre-100 Candidates
15. Tier 3 — Polish / Hardening
16. Tier 4 — Post-Release
17. Dependency Graph
18. Candidate Kernel Bundles
19. Smallest Credible Public-Release Set
20. Explicit Post-Release / Do-Not-Build-Before-100
21. Contradictions / Uncertainties Requiring Grant
22. Appendix — Evidence Index

Ask Grant only product questions the repository/history cannot answer.

---

# 16. Negative instructions

Do not:

- implement anything;
- create kernel specs;
- assign 91–100;
- create a roadmap;
- treat every historical promise as still valid;
- trust old docs over current code;
- trust code existence as product completion;
- rewrite modern foundations to resemble old architecture;
- revive superseded persona/session/access models;
- expand participant-local projections into alternate Scenes;
- invent arbitrary scripting;
- invent XP/gamification;
- automate all Socio rules;
- merge Audience into Player authority;
- turn Story So Far into telemetry;
- add a generic deck engine without launch need;
- polish The Cave if it is only an internal proof harness;
- call visual ugliness an architectural failure;
- call an unusable Director capability “done” merely because an endpoint exists.

---

# 17. Final report summary

At completion report:

```text
Historical kernel/reportback sets inspected:
Recovered unfinished/deferred requests:
CLOSED — current implementation:
CLOSED — superseded:
PARTIAL — bounded:
PARTIAL — productization:
OPEN — required before release:
OPEN — important/non-blocking:
DEFER — post-release:
RETIRE — legacy:
Consolidated underlying primitives:
Tier 0 count:
Tier 1 count:
Tier 2 count:
Tier 3 count:
Tier 4 count:
```

Then list:

- top 10 release-relevant gaps;
- top 5 historical requests later kernels already solved;
- top 5 legacy concepts that should no longer influence development;
- top authoring asymmetries;
- candidate bundles;
- smallest credible public-release set;
- questions requiring Grant’s product judgment.

End exactly with:

> **This audit reconciles recovered historical intent against the current Victory implementation. It is a release-prioritization aid, not a new roadmap.**

---

# 18. Success criterion

This audit succeeds if it turns Victory’s historical record from:

> dozens of overlapping old promises and deferrals

into:

> a small number of verified current gaps, clear release tiers, coherent implementation bundles, and an explicit list of things not worth building before feature freeze.

The most valuable outcome is not finding more work.

The most valuable outcome is proving how much old work **no longer needs to be done**.
