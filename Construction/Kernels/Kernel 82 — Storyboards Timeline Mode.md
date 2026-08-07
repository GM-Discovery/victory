# Kernel 82 — Storyboards Timeline Mode

**Status:** READY FOR IMPLEMENTATION
**Type:** Bounded Storyboards mode/template kernel
**Sequence position:** After Kernel 81A
**Product surface:** Storyboards
**Mode name:** Timeline

---

## 0. Kernel contract

Kernel 82 adds **Timeline** as a built-in Storyboard mode/template.

Timeline is not a separate board engine and is not a game-specific implementation. It is an immutable starting configuration that instantiates a normal owned Storyboard.

Required vertical:

```text
User chooses New Timeline
→ Victory instantiates a new owned Storyboard from the built-in Timeline template
→ the saved Timeline begins with a configurable Reference Panel
→ the saved Timeline begins with three columns:
   Beginning | ordinary | Ending
→ Beginning and Ending remain protected timeline boundaries
→ users expand only inward between those boundaries
→ bands organize broad time frames / eras / phases
→ rows support concurrent lanes, locations, themes, events, or other user-defined tracks
→ cards describe individual moments
→ middle-mouse drag pans large boards
→ the saved Timeline persists independently from the built-in template
→ JSON export preserves Timeline-mode metadata and Reference Panel structure
```

Kernel 82 must not hardcode Microscope terminology, trade dress, turn rules, or semantic object types.

---

# 1. Locked product decisions

## 1.1 Product name

The mode is called **Timeline**.

Storyboard creation should offer at minimum:

```text
Blank
Timeline
```

No "history game" wording is required.

## 1.2 Template versus saved board

The built-in Timeline is an immutable template/factory definition.

Users do not play on or edit that template directly.

Creating a Timeline produces a new normal owned Storyboard instance:

```text
Built-in Timeline Template
        │
        ├── create → Saved Timeline A
        ├── create → Saved Timeline B
        └── create → Saved Timeline C
```

Each saved Timeline has its own:

- ID;
- owner;
- title;
- sharing and permissions;
- Reference Panel configuration;
- columns;
- rows;
- bands;
- cards;
- images;
- locks;
- export state.

Editing Saved Timeline A must never modify the built-in Timeline template or Saved Timeline B.

## 1.3 Future personal templates

Do not implement **Save as Template** in Kernel 82.

Do not design the schema so tightly that a later user-created template feature would require replacing the model.

For this kernel:

> Templates instantiate Storyboards. Storyboards never mutate their source template.

## 1.4 Generic Timeline semantics

Do not hardcode:

- Period;
- Event;
- Scene;
- Lens;
- Focus;
- Light/Dark tone;
- placement assistance;
- game-specific hierarchy;
- question/answer fields;
- rules adjudication.

The generic interpretation is:

- **Band** → broad era, time frame, phase, act, epoch, or large grouping;
- **Row** → concurrent lane, location, event thread, perspective, theme, faction, or other track;
- **Column** → relative chronological position;
- **Card** → individual moment or unit of content.

Users remain free to interpret these differently.

## 1.5 Cards

Use the existing Storyboard card model.

Do not introduce Timeline-specific card subclasses.

A user may place a question on the front/body and an answer on the back if desired. Victory does not need dedicated fields for that.

## 1.6 Color / tone

Do not implement a tone system.

Cards already support color. Users decide what colors mean.

## 1.7 Placement

Do not add placement assistance or parent-selection UI.

Band, row, and column placement already imply enough structure.

## 1.8 Group Leader and current turn

Do not implement Group Leader, current-turn state, participant ordering, or Presence Tray right-click behavior in Kernel 82.

Those are broader reusable Victory features and should be implemented separately.

Kernel 82 may expose a small optional integration seam for a future provider to display Group Leader / Current Turn information, but must not temporarily own or duplicate that state.

---

# 2. Timeline default structure

## 2.1 Three-column default

A new Timeline starts with exactly three columns:

```text
Beginning | [ordinary middle column] | Ending
```

The middle column may have a neutral default label.

## 2.2 Stable boundary roles

Beginning and Ending are structural roles, not merely labels.

Conceptually:

```text
column_role = beginning
column_role = ordinary
column_role = ending
```

Visible labels may be renamed without changing the stable boundary role.

## 2.3 Boundary constraints

Beginning remains the left boundary.

Ending remains the right boundary.

Users may not reorder either boundary inward.

New columns may only be inserted between Beginning and Ending.

Boundary columns may contain cards.

## 2.4 Positional insertion controls

Column expansion must happen where the user is working.

### Beginning column

Expose:

```text
Add column right
```

Do not expose Add column left.

### Ending column

Expose:

```text
Add column left
```

Do not expose Add column right.

### Ordinary middle columns

Expose both:

```text
Add column left
Add column right
```

The user should not need to append a column and then reorder it just to expand the Timeline locally.

## 2.5 Column safeguard

Preserve the existing 200-column technical safeguard.

Do not present it as a permanent product maximum.

---

# 3. Configurable Reference Panel

## 3.1 Purpose

Timeline adds a persistent **Reference Panel** attached to each saved Timeline instance.

The panel holds shared setup/context outside the grid.

It is generic Storyboards infrastructure, not a Microscope-specific object.

## 3.2 Default Timeline fields

The built-in Timeline template should seed:

```text
Premise
Beginning
Ending
Include
Exclude
```

These are initial labels only. They are not sacred system fields.

Once instantiated, authorized users may reconfigure the saved Timeline's panel independently.

## 3.3 Field types

Support only:

- short text;
- long text;
- list;
- paired list.

Do not grow this into a general form-builder kernel.

## 3.4 Permissions

Use established Storyboards roles.

### Audience / Cast

May view permitted Reference Panel content only.

### Crew

May edit field content.

Crew may not add/remove/reorder/rename fields or change field type.

### Director+

May:

- edit content;
- rename fields;
- add/remove fields;
- reorder fields;
- change field type where safe.

### Owner

Has Director+ panel capabilities plus the existing board ownership/sharing authority.

Do not create new visible roles.

## 3.5 Field identity and serialization

Reference Panel fields require:

- stable UUID identity;
- deterministic serialized key/slug;
- human-readable label;
- field type;
- explicit sort order;
- content.

Follow Kernel 81A serialization rules:

- duplicate visible labels may exist when useful;
- internal serialized keys must remain unique in board scope;
- reordering must not change stable identity or serialized key;
- export must preserve fields unambiguously.

## 3.6 Structural edits

For one saved Timeline:

- field content persists;
- structure persists;
- order persists;
- renaming affects only that board;
- deleting a nonempty field requires deliberate confirmation;
- none of these operations modify future Timeline defaults.

## 3.7 Paired list

Paired list supports layouts such as:

```text
Include | Exclude
```

without hardcoding those concepts.

A paired list has:

- one field identity;
- two editable sublabels;
- two independently ordered item lists.

Examples could later be:

```text
Wanted | Unwanted
Allowed | Avoid
Include | Exclude
```

---

# 4. Template / instantiation architecture

## 4.1 Built-in modes

Kernel 82 supports at least:

```text
blank
timeline
```

Internal naming may use `mode`, `template_type`, or another repository-consistent term.

## 4.2 Built-in template immutability

Built-in templates may be implemented as:

- code-defined seed data;
- migration-backed immutable rows;
- versioned template records;
- another durable model.

The contract is what matters:

```text
instantiate template ≠ edit template
```

## 4.3 Instance metadata

A saved Timeline must record enough information to identify its origin:

```text
mode = timeline
template_version = ...
```

Boards are snapshots/instances after creation. They do not dynamically inherit later template changes.

## 4.4 Existing disposable test content

The implementation agent may delete prior Timeline/Storyboard test content that is confirmed disposable.

Do not spend significant work migrating test boards created while the Timeline model was still being designed.

Do not delete real user-owned content merely because it is old.

---

# 5. Reference Panel UI

## 5.1 Placement

Keep the Reference Panel persistently reachable while viewing the Timeline.

Acceptable presentation:

- side rail;
- collapsible panel;
- top panel;
- another Storyboards-consistent layout.

It must not obscure the board.

## 5.2 Collapse

Allow the panel to collapse so the user can maximize board space.

Persist collapsed state if inexpensive.

## 5.3 Editing affordances

Crew sees content editing affordances.

Director+ additionally sees structural configuration.

Keep configuration secondary to actually using the Timeline.

## 5.4 Long content

Long-text fields must remain usable without forcing the entire panel to expand indefinitely.

Use bounded areas, expandable editors, or an equivalent compact interaction.

---

# 6. Timeline navigation and panning

## 6.1 Preserve normal scrolling

Do not hijack standard browser interaction.

Preserve:

- mouse wheel scrolling;
- scrollbars;
- keyboard scrolling;
- touch scrolling.

## 6.2 Middle-mouse panning

Add:

> **Middle mouse button hold + drag = pan the Storyboard viewport.**

Requirements:

- horizontal and vertical pan;
- board follows pointer movement;
- cursor reflects panning state;
- release ends pan cleanly;
- no Storyboard mutation is sent;
- scroll position remains coherent;
- browser autoscroll interference is prevented where practical.

## 6.3 Interaction precedence

- left drag on draggable card → card movement;
- middle drag on board → viewport pan;
- wheel → normal scroll;
- left click → normal controls;
- keyboard behavior remains supported.

Middle-drag must not interfere with card drag, click, edit, or selection.

## 6.4 Long Timeline proof

Test with:

- many columns;
- several bands;
- multiple rows per band;
- long labels;
- collapsed bands;
- Reference Panel open and closed.

No minimap or zoom system belongs in Kernel 82.

---

# 7. Bands, rows, and cards remain generic

## 7.1 Bands

Use existing Storyboard bands unchanged.

The Timeline template should seed at least one reasonable default band.

Bands may represent eras/time frames but Victory does not enforce that meaning.

## 7.2 Rows

Use existing rows unchanged.

Rows may represent concurrent events, places, themes, factions, or any other lane.

## 7.3 Cards

Use existing Storyboard cards unchanged.

Card placement already implies:

- band;
- row;
- column.

Do not add redundant explicit Timeline parentage fields.

---

# 8. Session-state integration seam

A future platform kernel is expected to provide reusable session state such as:

- Group Leader;
- Current Turn;
- participant order;
- Presence Tray right-click actions.

Kernel 82 must not implement temporary versions of these features.

Document a clean integration point so Timeline can later consume them, for example:

```text
Reference Panel
→ optional session-state slot/provider
→ provider unavailable
→ slot hidden
```

Do not add Timeline-owned leader/turn tables.

---

# 9. Export

Preserve the existing `victory-storyboard` JSON export.

Timeline exports must additionally preserve:

- mode/template type;
- template version;
- boundary-column roles;
- Reference Panel definitions;
- field order;
- field content;
- stable field IDs;
- serialized keys;
- paired-list structure;
- Timeline-specific configuration.

Export the saved Timeline instance, not the shared built-in template definition.

Full import is not required, but the export must be sufficient for a future importer to reconstruct the Timeline accurately.

---

# 10. Account lifecycle

Timeline boards remain normal owned Storyboards.

Existing lifecycle behavior remains authoritative.

Kernel 82 must ensure:

- account export includes Timeline metadata and Reference Panel state;
- account deletion still resolves owned boards correctly;
- built-in templates never become user-owned;
- deleting a Timeline instance never deletes/modifies the built-in template;
- sharing remains instance-specific.

---

# 11. Persistence and migrations

Likely additions:

```text
storyboards.mode
storyboards.template_version
storyboard_reference_fields
storyboard_reference_items
storyboard_columns.column_role
```

Exact schema should follow repository conventions.

Requirements:

- additive migration(s);
- clean-install safe;
- stable field IDs;
- explicit ordering;
- deterministic serialized keys;
- existing Blank boards unchanged;
- existing Storyboards continue loading;
- account lifecycle updated;
- export updated;
- Kernel 81A serialization behavior preserved.

---

# 12. Backend change budget

Kernel 82 may add backend code for:

- Timeline instantiation;
- built-in template definition/versioning;
- Reference Panel CRUD;
- boundary roles and constraints;
- Timeline-aware column insertion;
- mode/template metadata;
- export extension.

Do not redesign:

- Storyboards permissions;
- card model;
- card images;
- card swapping;
- WebSockets;
- hidden filtering;
- ownership;
- sharing;
- account lifecycle architecture.

---

# 13. Frontend requirements

## Storyboards creation

Offer:

```text
Blank
Timeline
```

Choosing Timeline creates a new owned saved instance.

## Timeline board

Display:

- existing Storyboard grid/cards;
- configurable Reference Panel;
- Beginning / ordinary / Ending default columns;
- position-aware insert-left/right controls;
- middle-mouse panning.

Do not brand the experience as Microscope.

## Boundary controls

Beginning:

- rename;
- Add Right;
- no Add Left;
- cannot move inward.

Ending:

- rename;
- Add Left;
- no Add Right;
- cannot move inward.

Middle columns:

- Add Left;
- Add Right;
- rename;
- reorder inside boundaries;
- remove using existing occupied-content safeguards.

---

# 14. Required tests

## 14.1 Template instantiation

Prove:

- Blank still creates Blank;
- Timeline creates Timeline;
- Timeline starts with exactly three columns;
- Beginning role is left;
- Ending role is right;
- middle ordinary column exists;
- default Reference Panel exists;
- two new Timelines are independent;
- editing Timeline A does not affect Timeline B;
- editing an instance does not mutate future Timeline defaults.

## 14.2 Boundary behavior

Prove:

- Beginning cannot move inward;
- Ending cannot move inward;
- nothing inserts outside Beginning;
- nothing inserts outside Ending;
- Add Right from Beginning works;
- Add Left from Ending works;
- middle Add Left/Right work;
- labels may rename;
- boundary roles survive rename;
- Blank boards are unaffected.

## 14.3 Reference Panel

Prove:

- default fields seed correctly;
- Audience/Cast read only;
- Crew edits content;
- Crew cannot restructure;
- Director+ adds/removes/renames/reorders;
- duplicate labels serialize uniquely;
- IDs/serialized keys survive reorder;
- deletion of nonempty field requires deliberate action;
- paired-list values persist;
- all changes remain instance-local.

## 14.4 Panning

Frontend tests should prove where practical:

- middle-down begins pan;
- movement changes viewport scroll;
- release stops pan;
- no mutation request sent;
- left card drag still works;
- wheel scroll remains functional.

## 14.5 Export

Prove:

- mode/template version preserved;
- boundary roles preserved;
- Reference Panel structure/content preserved;
- serialized keys preserved;
- Blank export remains backward-compatible.

## 14.6 Regression

Run:

- full Go suite;
- Storyboards tests;
- Kernel 81 browser/UI tests;
- Kernel 81A serialization tests;
- hidden-card tests;
- WebSocket tests;
- eWrite-link tests;
- export tests;
- lifecycle tests.

---

# 15. Required browser automation

Playwright proof is mandatory.

Use disposable users and disposable boards.

Prove:

1. Create flow offers Blank and Timeline.
2. New Timeline creates a saved instance.
3. Second new Timeline starts fresh.
4. Editing Timeline A Reference Panel does not alter Timeline B.
5. Beginning / middle / Ending render correctly.
6. Beginning exposes Add Right but not Add Left.
7. Ending exposes Add Left but not Add Right.
8. Middle exposes both.
9. Inserted columns land in the correct place.
10. Boundary labels rename without losing boundary behavior.
11. Crew edits Reference Panel content.
12. Crew cannot restructure Reference Panel.
13. Director+ can restructure it.
14. Paired list works.
15. Middle-mouse drag pans horizontally.
16. Middle-mouse drag pans vertically.
17. Card drag still works.
18. A long Timeline remains navigable.
19. Export contains Timeline metadata and Reference Panel state.
20. Disposable test Timelines can be deleted cleanly with zero residue.

No PASS based only on API evidence for the Timeline UI.

---

# 16. Required artifacts

```text
Construction/Kernels/Kernel 82 — Storyboards Timeline Mode.md
Construction/Storyboards/timeline-mode-contract.md
Construction/Storyboards/storyboard-template-contract.md
Construction/Storyboards/reference-panel-contract.md
Construction/Storyboards/timeline-navigation.md
Construction/Storyboards/session-state-integration-seam.md
Construction/OperatorLogs/kernel-82-reportback.md
```

Update Storyboards domain/UI docs where needed.

---

# 17. Evidence

Baseline:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

Backend:

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
```

Use isolated `TEST_DATABASE_URL`.

Also run frontend logic tests, Node syntax checks, Playwright proof, and:

```bash
git diff --check
```

---

# 18. Pass criteria

Kernel 82 passes when:

- Timeline exists as a built-in immutable Storyboard template/mode;
- users create normal owned instances from it;
- saved Timelines do not mutate the source template;
- every new Timeline starts fresh;
- Timeline starts with exactly three columns;
- Beginning and Ending remain protected boundary roles;
- expansion occurs only inward;
- positional Add Left/Right controls work;
- Reference Panel exists and is configurable per saved instance;
- Crew edits panel content;
- Director+ edits panel structure;
- field IDs and serialized keys remain stable;
- paired-list fields work;
- bands/rows/cards remain generic;
- no Microscope-specific terminology/mechanics are hardcoded;
- no tone system exists;
- no placement assistant exists;
- Group Leader/current-turn logic remains deferred;
- middle-mouse panning works;
- normal scrolling and card dragging still work;
- Timeline metadata exports correctly;
- account lifecycle remains intact;
- Playwright proof passes;
- no Kernel 80/81/81A regressions occur.

---

# 19. Partial and fail rules

## PARTIAL

Use when:

- Timeline instances still share mutable template state;
- Reference Panel content works but structure cannot be configured;
- Beginning/Ending are only labels and not protected roles;
- insertion remains append-and-reorder;
- panning breaks ordinary scrolling;
- export omits Timeline-specific state.

## FAIL

Use when:

- editing one Timeline changes another;
- editing a saved Timeline changes future defaults;
- Beginning/Ending can be displaced outside the timeline;
- Audience/Cast gain panel edit authority;
- Crew gains structural panel authority;
- duplicate serialized keys occur;
- middle-pan breaks card drag;
- Timeline introduces Microscope-specific rules/trade dress;
- Storyboards security/permissions regress;
- Grant is locked out.

---

# 20. Non-goals

Kernel 82 does not:

- implement Microscope;
- use Lens or Focus terminology;
- use Period/Event/Scene terminology;
- implement tone;
- implement Group Leader;
- implement current-turn state;
- change Presence Tray behavior;
- enforce game turns;
- provide placement assistance;
- add semantic card types;
- add question/answer fields;
- add visual/PDF export;
- add user-created templates;
- add Save as Template;
- add minimap or zoom;
- redesign Storyboards permissions;
- redesign card rendering.

---

# 21. Kernel-size guardrail

Kernel 82 is a mode/template kernel.

Target implementation shape:

- one Timeline template/mode definition;
- one Reference Panel persistence model;
- one boundary-column extension;
- one middle-pan interaction;
- one export extension;
- one complete browser journey.

Do not expand it into:

- generic template marketplace;
- session leadership;
- turn-order system;
- Presence Tray rewrite;
- game-rule enforcement;
- alternate Storyboard modes.

---

# 22. Immediate operator outcome

At completion, Grant must be able to answer:

1. Can I choose Timeline when creating a Storyboard?
2. Does that create a saved Timeline rather than edit a shared template?
3. Does every new Timeline start fresh?
4. Can I radically change one Timeline without changing the next one I create?
5. Does it begin with Beginning, a middle column, and Ending?
6. Can the timeline expand only inward between the bookends?
7. Can I insert a column exactly where I need it?
8. Can I rename Beginning and Ending without losing boundary behavior?
9. Can I configure the Reference Panel on this Timeline?
10. Can Crew edit panel content without restructuring it?
11. Can Director+ restructure the panel?
12. Can paired lists support things like Include/Exclude without hardcoding those words?
13. Can bands represent eras while rows represent concurrent lanes?
14. Can cards remain generic moments?
15. Can I middle-drag across a long Timeline?
16. Does normal scrolling still work?
17. Does card dragging still work?
18. Does export preserve Timeline and Reference Panel state?
19. Is Group Leader/current-turn wiring still cleanly deferred to a future platform feature?
20. Did the kernel avoid turning Timeline into somebody else's game?

Kernel 82 succeeds when Victory can create and use long, flexible, independently saved Timeline Storyboards without hardcoding a particular tabletop game into the platform.
