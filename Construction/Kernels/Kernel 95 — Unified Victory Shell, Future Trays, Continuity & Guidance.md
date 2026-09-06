# Kernel 95 — Unified Victory Shell, Future Trays, Continuity & Guidance

**Status:** DRAFT — ready for implementation  
**Type:** Shared live-shell rebuild + visual unification + continuity/event surfacing + contextual guidance  
**Sequence position:** After Kernel 94  
**Primary proof:** Victory feels like one coherent modern product while preserving the distinct identity and purpose of each venue  
**Core doctrine:** The interface should feel like future augmented reality designed so well for humans that its imposition is negligible and its benefit appears on demand.

---

## 0. Kernel mode

Kernel 95 has one deep rebuild and one broad but bounded visual pass.

### Deep rebuild
Rebuild the shared live-theater tray/shell primitive used by Catharsis / First Theater where appropriate.

### Broad pass
Normalize the shared visual language across Victory without flattening venue identity.

This kernel also owns:

- continuity/event surfacing;
- Trailer notification pips;
- new “My People” notification surfacing;
- deeper slash-command guidance through the existing Guide tab;
- cleanup of inconsistent shared navigation/chrome such as duplicate Return to Map controls.

Do not turn this into a whole-product capability expansion.

---

## 1. Product feeling

Victory’s shared interface should feel like:

> **augmented reality from the future that is so well designed for humans that its imposition is negligible and its benefit is available on demand.**

The stage, story, people, and venue are primary.

Interface appears when useful, recedes when not, and should rarely feel like a dashboard wrapped around the experience.

Prefer:

- contextual presence;
- translucency;
- depth;
- subtle motion;
- spatial hierarchy;
- restrained chrome;
- remembered local preferences;
- controls close to the work;
- visual calm.

Avoid:

- permanent heavy panels;
- admin-dashboard grids;
- excessive borders;
- redundant navigation;
- duplicated controls;
- tool clutter;
- visual sameness across venues.

---

## 2. Venue identity is preserved

Victory should feel unified without making every venue look identical.

Theaters are different. Shows are different. Purposes are different.

A middle-school stage should not feel like the same room as a mature theatrical venue with different labels.

K95 should unify interaction grammar, typography, common control behavior, shared navigation, modal treatment, spacing, focus/hover treatment, button/input language, tray mechanics, notification language, and shared accessibility behavior.

K95 should not force identical layouts, palettes, density, theatrical tone, or show capabilities.

A shared product language is the goal, not visual homogenization.

---

## 3. The shared tray primitive

Audit the current shared tray implementation used by Catharsis / First Theater before changing it.

The current primitive is believed to be close, but can be better.

Rebuild or substantially refactor it into a cleaner Vue-managed client-state model where that materially improves:

- open/closed state;
- pinned/unpinned state;
- transparency;
- sizing;
- position;
- role-specific contents;
- user customization;
- focus;
- transitions;
- persistence of local preferences;
- future maintainability.

Vue owns presentation state. The server remains canonical for shared game/show state.

---

## 4. Tray role model

### Audience
**No trays.**

### Cast
Cast has **all four trays**.

### Director
Director uses the same underlying live-shell/tray primitive where appropriate, plus Director tools in the **top-right corner**.

### Producer
Producer has the **Production Room**.

### Operator
Operator has the **Cabin in the Woods**.

The shared primitive should support common live-interface mechanics without erasing role-specific spaces.

---

## 5. Tray behavior

Trays should feel useful on demand and negligible when not needed.

Support where practical:

- **pinning**;
- **collapsing**;
- **transparency / translucency adjustment**;
- **customization**;
- remembered local state;
- smooth open/close behavior;
- clear active state;
- keyboard reachability;
- graceful overlap avoidance;
- sensible defaults.

Do not build a universal desktop window manager.

---

## 6. Stage dominance

In live theater surfaces:

> **the stage wins.**

Trays must not unnecessarily cover or visually compete with the stage.

When trays are collapsed/receded, the stage should feel substantially more open.

When trays are needed, they should feel layered over the experience rather than replacing it.

---

## 7. Director tools boundary

First Theater may eventually benefit from Catharsis Director capabilities such as announcements and other shared tools.

**Not in K95.**

K95 may improve the presentation of tools that already exist in each venue.

K95 must not broaden venue capability simply because the live-shell primitive is being rebuilt.

Record useful parity opportunities for Kernel 102+.

---

## 8. Broad Victory visual normalization

Outside the live shell, K95 performs a lighter unification pass.

Audit and normalize:

- typography;
- spacing;
- buttons;
- inputs;
- select controls;
- tabs;
- modals;
- overlays;
- menus;
- headers;
- card treatment;
- hover states;
- focus states;
- disabled states;
- destructive states;
- loading states;
- empty states;
- shared status indicators;
- notification badges/pips.

Do not redesign every venue layout.

If a venue’s layout is coherent but visually older, modernize the shared parts and move on.

---

## 9. Navigation cleanup

Victory currently has inconsistent/shared navigation placement in some venues, including duplicate Return to Map controls.

K95 should audit common navigation.

Goals:

- one understandable primary route back to the campus/map;
- consistent placement where the venue allows;
- no duplicate Return to Map controls competing with one another;
- venue-specific navigation preserved when it serves a real purpose;
- browser back behavior remains ordinary.

Do not create a giant global navigation bar merely to enforce consistency.

---

## 10. Campus / main-shell chrome

K95 may improve the shared campus shell.

Review:

- top-level navigation;
- map menu;
- account/login surfaces;
- common modal language;
- global buttons;
- notification treatment;
- shared headers.

The campus should immediately feel like the same product as the modernized venues.

Do not flatten the campus into a generic SaaS shell.

---

## 11. Continuity should stay mostly hidden

Victory should not constantly narrate its own state.

Continuity/event surfacing should be:

> **mostly hidden until useful.**

Prefer small pips, badges, subtle indicators, changed-state markers, contextual resurfacing, and user-invoked detail.

Avoid permanent activity feeds, noisy banners, event logs dominating the interface, or social-network-style notification walls.

---

## 12. Trailer notification pip

The Trailers should visibly indicate when the user has new messages.

Required:

- a small clear notification pip/badge on the relevant Trailer surface/entry;
- no need to open the Trailer to know something new exists;
- the indicator clears or updates based on the existing read-state model;
- do not invent duplicate message truth.

Use existing message/unread state if available.

If unread state is missing, add the smallest canonical state necessary.

---

## 13. “My People” notification

When someone adds the user in the existing “My People” / relationship system, surface that change.

Prefer:

- subtle pip/badge;
- no intrusive modal;
- no social-feed expansion;
- existing relationship truth remains canonical.

The user should be able to discover that someone new has added/connected with them without Victory interrupting unrelated work.

Audit the current relationship model before adding notification state.

---

## 14. Guide tab remains the teaching home

Victory already has a **Guide** tab in the chat pod that works well when needed.

Use it.

K95 should deepen slash-command training through the Guide rather than adding a separate command manual.

Goals:

- commands are discoverable;
- guidance appears when relevant;
- users can intentionally open the Guide for help;
- command examples are concise;
- role/context determines what is shown where possible.

Do not dump the full command corpus into the main interface.

---

## 15. Contextual slash-command teaching

Prefer contextual teaching.

The Guide may surface particularly useful commands in relevant contexts and should organize command groups by purpose, not implementation namespace.

Do not nag.

Do not make slash-command literacy necessary for ordinary use.

Commands are power tools. The GUI remains legitimate.

---

## 16. Vue scope

K95 should use Vue deeply where state complexity justifies it.

Priority:

1. shared theater tray/live-shell state;
2. shared interactive components naturally touched by the rebuild;
3. notification components if useful;
4. common shell components where Vue meaningfully reduces duplication.

Do **not** convert every static page to Vue for ideological consistency.

If a plain page works well, visual normalization may be enough.

---

## 17. Shared component strategy

Do not create a giant formal design-system project.

Do create/reuse shared components where repeated interaction patterns clearly exist.

Good candidates may include:

- tray shell;
- icon button;
- notification pip;
- shared modal treatment;
- shared tab treatment;
- shared header/navigation control;
- common form controls;
- tooltip/help affordances.

A component should exist because it reduces inconsistency or state complexity, not because every element needs abstraction.

---

## 18. Visual language inherited from K94

Audit the successful Storyboards patterns from Kernel 94.

Carry forward what actually worked:

- typography;
- spacing rhythm;
- control treatment;
- focus/hover language;
- motion principles;
- translucency/depth;
- empty-state tone;
- contextual controls;
- modal treatment;
- destructive-action language.

Do not copy Storyboards-specific composition into every venue.

Extract principles, not screenshots.

---

## 19. Motion

K95 may use motion to support the future-AR feeling.

Appropriate:

- tray reveal/recede;
- subtle pin state;
- opacity transitions;
- notification pip arrival;
- menu/modal continuity;
- contextual tool appearance.

If motion becomes a troubleshooting sink, simplify it.

Respect reduced-motion preferences.

---

## 20. Accessibility

The modernization must preserve or improve accessibility.

Required baseline:

- semantic controls;
- visible focus;
- keyboard operation of critical tray actions;
- readable contrast even with translucency;
- reduced-motion support;
- descriptive labels/tooltips for unfamiliar icons;
- no important state conveyed by color alone where practical.

Do not make “futuristic” mean cryptic.

---

## 21. Server authority

Client presentation state and shared game/show state are different.

Vue/local client state may own:

- tray open/closed;
- pinned state;
- opacity;
- local arrangement/customization;
- local Guide presentation state.

Server truth continues to own:

- roles;
- permissions;
- live show state;
- Presence;
- messages;
- relationships;
- stage state;
- Director actions;
- authoritative game data.

Do not blur the boundary.

---

## 22. Persistence

Local UI preferences should survive ordinary reloads where practical.

Examples:

- tray pin state;
- tray collapse state;
- opacity preference.

Use appropriate local/user preference storage already present in Victory if available.

Do not create database schema for every cosmetic preference unless existing architecture clearly favors server-synced preferences.

---

## 23. Role and authority boundary

K95 is not Kernel 97.

Do not broaden or redefine authority.

Verify that presentation changes do not accidentally expose controls or data to the wrong role.

At minimum:

- Audience receives no new trays;
- Cast tray controls remain within Cast authority;
- Director tools remain Director-authorized;
- Producer surfaces remain properly scoped;
- Operator surfaces remain properly scoped.

If a real authority defect is exposed, make the smallest canonical fix and record broader audit debt for K97.

---

## 24. Venue capability boundary

K95 changes presentation and client-state architecture.

It does not broaden theater capabilities.

Examples explicitly deferred:

- giving First Theater Catharsis announcements;
- adding new Director tool families;
- cross-venue feature parity for its own sake;
- new show-control mechanics.

Record those for Kernel 102+.

---

## 25. Repository audit before implementation

Inspect at minimum:

1. shared tray implementation;
2. First Theater live shell;
3. Catharsis live shell;
4. Cast tray state;
5. Director top-right tools;
6. Audience live presentation;
7. Producer Production Room;
8. Operator Cabin;
9. Presence integration;
10. diagnostics;
11. health/status trays;
12. dice-related tray/surface state;
13. current shared CSS/components;
14. current Vue infrastructure from K94;
15. campus/main shell;
16. Return to Map implementations;
17. common modals;
18. common buttons/forms;
19. Trailer messaging/unread state;
20. My People relationship state;
21. chat pod Guide tab;
22. slash-command catalogue/help behavior;
23. permissions around each shared surface.

Do not ask Grant repository-answerable questions.

---

## 26. Implementation passes

### Pass 1 — Audit + shell map
Identify shared tray primitive, role compositions, duplicated chrome, notification sources, and K94 patterns worth carrying.

### Pass 2 — Vue tray shell
Rebuild/refactor shared tray state and preserve behavior.

### Pass 3 — Future-AR interaction pass
Pinning, collapsing, translucency, customization, motion, focus, accessibility.

### Pass 4 — Catharsis + First Theater exercise
Prove the shared primitive in both venues without adding new venue capabilities.

### Pass 5 — Shared visual normalization
Buttons, inputs, menus, modals, typography, headers, navigation, common shell.

### Pass 6 — Continuity + notifications
Trailer message pip, My People pip, restrained event surfacing.

### Pass 7 — Guide/slash-command refinement
Improve contextual command learning inside the existing Guide tab.

### Pass 8 — Human browser review
Stop and let Grant inspect the actual product across representative venues/roles.

Do not keep inventing passes after the acceptance bar is met.

---

## 27. Human review route

Grant should inspect representative surfaces, not every file.

At minimum review:

- campus/main map shell;
- First Theater as Cast;
- Catharsis as Cast;
- Catharsis/First Theater as Director where currently supported;
- Trailer with unread message pip;
- My People new-connection indicator;
- chat pod Guide;
- one or two non-theater venues to confirm shared visual normalization;
- Return to Map behavior from multiple venues.

---

## 28. Acceptance — trays

K95 passes the tray portion when Grant judges:

- trays feel lightweight and futuristic;
- stage remains dominant;
- open/close is smooth and understandable;
- pinning is useful;
- translucency is useful;
- customization is enough without becoming fiddly;
- Cast has the expected four-tray experience;
- Audience remains tray-free;
- Director tools remain appropriately separate;
- state behaves correctly across reload where intended.

---

## 29. Acceptance — unified Victory

K95 passes the broad visual portion when Grant judges:

- Victory feels substantially more like one product;
- shared controls look and behave consistently;
- obvious legacy visual inconsistencies are reduced;
- Return to Map/navigation confusion is cleaned up;
- venues still retain distinct identity;
- no major venue was flattened into generic SaaS design.

---

## 30. Acceptance — continuity

K95 passes continuity/event surfacing when:

- new Trailer messages can be noticed without opening the Trailer;
- new My People activity can be noticed without intrusive interruption;
- indicators use canonical underlying state;
- continuity cues remain mostly hidden when nothing needs attention;
- no noisy universal activity feed was introduced.

---

## 31. Acceptance — Guide

K95 passes slash-command guidance when:

- the existing Guide tab remains the teaching home;
- useful commands are easier to discover;
- guidance is contextual where practical;
- ordinary users are not forced into slash commands;
- command help does not dominate live play.

---

## 32. Alpha readiness

K95 should materially improve the first impression for alpha testers.

A tester should encounter:

- consistent interaction grammar;
- cleaner shared navigation;
- modern shared chrome;
- less visual disorientation;
- live theater UI that feels intentional;
- help available when wanted;
- subtle indications when something new needs attention.

Do not delay alpha over micro-polish.

---

## 33. Deferred to 102+

Explicitly record rather than implement:

- First Theater gaining Catharsis announcements;
- broader Director-tool parity across theaters;
- new shared theater capabilities;
- venue capability redesign;
- other “while we are here” enhancements discovered during the shell rebuild.

K95 proves the presentation primitive.

Later kernels may expand what is placed inside it.

---

## 34. Decision memory

Locked product decisions:

- Shared interface feeling: **future augmented reality with negligible imposition and benefit on demand**.
- Audience has **no trays**.
- Cast has **all four trays**.
- Directors retain Director tools in the **top-right corner**.
- Producers retain the **Production Room**.
- Operator retains the **Cabin in the Woods**.
- Trays may be **pinnable, translucent, collapsible, and customizable**.
- Broad beautification is visual normalization, not venue-layout homogenization.
- Different theaters and shows should remain visually/purposefully distinct.
- Continuity/event surfacing stays **mostly hidden**.
- Trailers need a **new-message pip**.
- “My People” needs a subtle indicator when someone newly adds/connects with the user.
- Slash-command teaching belongs in the existing **Guide tab** in chat.
- Duplicate/inconsistent **Return to Map** controls should be cleaned up.
- First Theater gaining Catharsis announcements or other Director tools is **102+**, not K95.

---

## 35. Reportback

Report:

1. exact shared tray primitive found;
2. what was moved/refactored into Vue;
3. role-by-role tray behavior;
4. pin/collapse/translucency/customization behavior;
5. local preference persistence;
6. Catharsis proof;
7. First Theater proof;
8. broad shared visual changes;
9. navigation/Return to Map cleanup;
10. Trailer notification implementation;
11. My People notification implementation;
12. Guide/slash-command improvements;
13. accessibility changes;
14. backend/API/schema changes, if any, and why necessary;
15. authority regressions checked;
16. deferred 102+ capability ideas;
17. automated proof;
18. browser proof;
19. Grant acceptance result.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

Do not call PASS until the real browser experience has been reviewed.

---

## 36. Completion condition

Kernel 95 passes when Victory no longer feels like a collection of separately evolved interfaces.

It should feel like:

> **one coherent future system whose interface appears when useful, recedes when not, and changes character to fit the venue without changing its underlying design intelligence.**

The product should be more unified.

The venues should still be themselves.
