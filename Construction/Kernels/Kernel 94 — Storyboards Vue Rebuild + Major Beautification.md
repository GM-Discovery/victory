# Kernel 94 — Storyboards Vue Rebuild + Major Beautification

**Status:** Passes 1–5 built and live-reviewed by Grant pass-by-pass (positive reactions throughout), plus a PDF export added per his 2026-08-31 follow-up request. Not yet formally closed — the explicit combined Empty/Working/Full acceptance gate (spec §32/34) hasn't been asked for as its own verdict. See `Construction/OperatorLogs/kernel-94-storyboards-vue-rebuild-ledger.md` for the full pass-by-pass record, bugs found and fixed, decisions, and deferred items.  
**Type:** Frontend rebuild + visual design proving ground + bounded interaction polish  
**Sequence position:** After Kernel 93  
**Primary proof:** Storyboards feels inviting when empty, fluid while being built, and impressive enough to present as a finished digital exhibition when full  
**Core doctrine:** The work is the spectacle. Controls support creation, then get out of the way.

---

## 0. Kernel mode

Kernel 94 is a major presentation-layer kernel.

This is not a request to “convert Storyboards to Vue” and stop.

It is also not permission to redesign Victory’s entire product architecture.

The goal is:

> **Rebuild Storyboards as a spatial creative worktable that can mature into a digital showing space.**

Storyboards is the visual proving ground for the next generation of Victory’s interface.

Kernel 95 will carry successful visual language outward into the broader application.

K94 should therefore go deep on one venue instead of shallow across every venue.

---

## 1. Product feeling

Storyboards has two equally important emotional states.

### Empty

An empty board should feel like:

> **space to create**

It should feel open, calm, inviting, and unfinished in a productive way.

It should not feel like:

- an empty admin dashboard;
- a blank spreadsheet;
- a settings form waiting for data;
- a dead gray canvas;
- a wall of empty bordered boxes.

The user should understand where creation begins without being surrounded by controls.

### Full

A mature board should feel like:

> **a masterpiece in a digital showing space**

The board itself should become presentable.

It should feel composed, rich, legible, and intentional rather than merely “populated.”

A full board should reward looking at it.

The interface should support the possibility that the artifact itself is being shown to other people.

---

## 2. Spatial metaphor

The primary metaphor is:

> **A spatial creative worktable, not a dashboard.**

Cards, Time Frames, Events, and Moments should feel placed.

The structure remains meaningful, but the visible interface should not over-emphasize its underlying grid.

Prefer:

- open visual fields;
- subtle alignment;
- spatial rhythm;
- whitespace;
- contextual controls;
- clear hierarchy;
- depth when useful;
- transitions that explain movement.

Avoid:

- nested cards inside panels inside cards;
- permanent sidebars unless clearly justified;
- dense toolbars;
- enterprise dashboard composition;
- generic admin surfaces;
- excessive borders;
- every region being boxed merely because it can be.

---

## 3. Structural model stays canonical

Preserve the current Storyboards / Timeline domain model unless an actual defect requires a bounded correction.

The current semantic structure remains:

- **Time Frame**
- **Event**
- **Moment**

Do not turn Storyboards into a Miro clone.

Do not replace the structured timeline with a universal freeform canvas.

Do not invent a new board domain merely because Vue makes it tempting.

---

## 4. Placement model

The visual presentation may become looser than the current implementation.

Product direction:

- snapping is acceptable;
- clear cell/lane membership is acceptable;
- free placement **within** a meaningful cell/region is acceptable if it remains understandable;
- exact pixel freedom across the entire board is not required;
- structural placement must remain stable and serializable.

Prefer the simplest interaction model that feels spatial.

If full free placement creates significant persistence, collision, responsive-layout, or drag-debugging complexity, keep snapping.

The user should perceive freedom without Victory becoming a layout engine.

---

## 5. Vue rebuild

Rebuild the Storyboards frontend in Vue where doing so materially improves:

- state management;
- component boundaries;
- drag/drop behavior;
- contextual controls;
- edit states;
- visual transitions;
- maintainability;
- future polish.

Vue is a means, not the deliverable.

Do not preserve awkward interaction merely to minimize rewrite cost.

Do not rewrite backend APIs solely to make the Vue code prettier.

Audit the existing implementation first and retain proven domain behavior.

---

## 6. Cards get a real redesign

K94 is allowed and expected to substantially redesign Storyboards cards.

Review:

- dimensions;
- proportions;
- typography;
- title hierarchy;
- body text;
- metadata;
- color treatment;
- imagery;
- front/back behavior;
- flip behavior;
- selection;
- hover;
- editing;
- dragging;
- focus;
- Q/A presentation for Moments;
- empty-state card creation;
- long-content behavior.

Cards should feel like authored artifacts, not form containers.

They should remain readable when many are visible at once.

Do not solve “beautiful” by making every card enormous.

Do not bury content under ornamental styling.

---

## 7. Progressive visual density

The board should respond gracefully to content density.

### Sparse board

Prefer:

- generous breathing room;
- restrained controls;
- obvious creation affordances;
- visual invitation.

### Working board

Prefer:

- clear placement;
- easy drag/reorder;
- quick editing;
- strong active/focus indication;
- contextual tools near the work;
- enough structure to prevent confusion.

### Dense / completed board

Prefer:

- composition;
- scanability;
- visual rhythm;
- reduced chrome;
- coherent relationships among Time Frames, Events, and Moments;
- a sense that the board is worth presenting.

Do not allow the full board to collapse into a noisy wall of controls.

---

## 8. Controls should emerge and recede

Creation and editing controls should generally appear when relevant.

Use contextual interaction where practical:

- hover;
- selection;
- focus;
- right-click/context menus where consistent with Victory;
- small inline creation affordances;
- temporary edit surfaces;
- lightweight overlays.

Do not permanently display every possible control.

The board should spend more screen area showing the work than explaining how to manipulate the work.

---

## 9. Motion is allowed to be fun

K94 may use meaningful animation and transitions.

Appropriate examples:

- card relocation that visibly preserves identity;
- soft insertion/removal;
- expansion and collapse;
- card flip;
- smooth structural reflow;
- contextual controls entering/leaving;
- board navigation that preserves spatial continuity;
- gentle drag feedback;
- restrained depth/parallax if it materially improves spatial comprehension.

The standard is:

> **fun is allowed; fragile cleverness is not.**

If a motion effect becomes a debugging sink, simplify it.

Do not spend hours preserving an animation that does not materially improve understanding or delight.

Avoid:

- bouncing for its own sake;
- excessive swooping;
- slow cinematic transitions;
- motion that blocks input;
- motion that makes dense boards tiring;
- animation that obscures where content moved.

Respect reduced-motion preferences.

---

## 10. Empty-state design

The empty board deserves deliberate design.

Required:

- clear sense that this is a creative space;
- obvious first action;
- no intimidating sea of placeholders;
- no giant instruction manual;
- no dense toolbar as the dominant object.

The first creation action should feel natural.

Prefer invitation over instruction.

---

## 11. Editing experience

Editing should feel immediate.

Audit and improve:

- title editing;
- description/content editing;
- card type distinctions;
- Moment Q/A;
- color selection;
- structural movement;
- creation;
- deletion;
- duplication if already supported;
- flip/front-back interaction;
- image/thumbnail handling where already supported.

Avoid modal dialogs for routine edits unless there is a compelling reason.

Do not redesign the underlying content model merely to avoid one awkward control.

---

## 12. Dragging and movement

Dragging should feel physical enough to understand.

Required:

- clear grabbed state;
- clear valid destination;
- no mysterious disappearing;
- no silent structure changes;
- stable post-drop placement;
- server persistence remains canonical;
- reload reflects the same structure.

If a card changes structural parent/region through dragging, the visual transition should make that understandable.

Keyboard-accessible alternatives must remain possible for important movement/actions.

---

## 13. Time Frames, Events, and Moments

Hierarchy must remain legible without relying on heavy boxes.

Explore visual techniques such as:

- spacing;
- scale;
- typography;
- subtle backgrounds;
- bands;
- dividers;
- depth;
- alignment;
- labeling;
- restrained color fields.

Do not automatically render each semantic level as a bordered rectangle.

Time Frames may remain visually band-like if that continues to work.

Events and Moments should clearly relate to their context without looking like spreadsheet cells.

---

## 14. The protected Ending

Audit the current protected Ending behavior during the rebuild.

Known prior behavior allowed generic column insertion after the protected Ending in some circumstances.

If the Vue rebuild naturally touches the relevant ordering control, prevent new content from being appended beyond the protected Ending.

Do not expand this into a Timeline architecture rewrite.

If correction is non-trivial or unrelated to the rebuilt surface, record it for later polish.

---

## 15. Presence / coordination accessibility debt

Prior coordination surfaces have had keyboard-access concerns.

Where K94 touches shared Storyboards presence/coordination controls:

- preserve or improve keyboard reachability;
- use semantic buttons;
- maintain visible focus;
- do not create mouse-only critical controls.

Do not undertake a whole-product accessibility rewrite in K94.

---

## 16. Responsive behavior

Storyboards is fundamentally a large spatial workspace.

Desktop/laptop is primary.

Still ensure:

- resizing does not corrupt placement;
- ordinary laptop widths remain usable;
- controls do not overlap catastrophically;
- board navigation remains understandable;
- text remains readable;
- narrow views fail gracefully.

Do not spend the kernel attempting to make a dense production board ideal on a phone.

---

## 17. Performance

A dense board may contain substantial content.

The rebuild must remain comfortable on realistic boards.

Avoid:

- re-rendering the entire board on every minor input if avoidable;
- animation storms;
- excessive DOM duplication;
- giant unbounded observers;
- expensive visual effects on every card.

Do not prematurely optimize hypothetical scale.

Use existing realistic board sizes as proof.

---

## 18. Preserve server authority

The server remains authoritative.

Vue local state may provide responsiveness, but it must reconcile with canonical persisted state.

Required:

- refresh preserves board;
- reconnect preserves board;
- multi-client changes do not silently fork;
- failed saves become understandable;
- optimistic UI must not create false success.

Do not create a second client-only Storyboards truth.

---

## 19. Networking / synchronization

Preserve existing synchronization behavior.

Audit current Storyboards serialization and networking before changing component structure.

Required proof should include:

- create;
- edit;
- move;
- flip if networked;
- color/style persistence;
- structural placement;
- deletion;
- reload;
- second client where practical.

Do not re-solve networking unless the existing implementation prevents the rebuild.

---

## 20. Visual language proving ground

K94 should establish candidate Victory-wide design principles for K95.

During implementation, record successful patterns such as:

- typography hierarchy;
- control shapes;
- hover/focus treatment;
- panel treatment;
- spacing scale;
- contextual toolbar behavior;
- motion principles;
- form/input treatment;
- card depth;
- empty-state composition;
- destructive-action treatment.

Create a small visual-design note for K95 rather than forcing every venue to change now.

Suggested path:

`Construction/Design/Kernel 94 Visual Language.md`

If an equivalent design reference already exists, update it instead of creating duplicate doctrine.

---

## 21. Do not beautify every venue in K94

K94 goes deep on Storyboards.

Do not turn this kernel into a site-wide reskin.

Kernel 95 owns the broader propagation pass.

Small shared-component improvements are acceptable when:

- Storyboards directly depends on them;
- they clearly improve the rebuilt venue;
- they do not create a second large workstream.

Do not wander through unrelated venues fixing visual annoyances.

Record them for K95.

---

## 22. Human critique loop

Visual quality cannot be accepted solely from tests.

The implementation loop is:

1. agent makes a coherent visual pass;
2. app runs;
3. Grant inspects it in the browser;
4. Grant critiques what he sees;
5. agent makes the next bounded pass;
6. repeat until the experience reaches the kernel bar.

The agent should not disappear for hours attempting autonomous perfection.

Do not ask Grant repository-answerable questions.

Do ask Grant when an actual aesthetic/product choice cannot be inferred.

---

## 23. Git workflow for visual iteration

K94 is expected to involve many visual experiments.

Do **not** create meaningless permanent history for every tweak.

Preferred workflow:

- work on `kernel-94-storyboards` or equivalent feature branch;
- allow multiple uncommitted edits between reviews;
- use `git diff` to inspect work;
- commit only coherent visual/functional checkpoints;
- if temporary commits are required to transport/deploy builds, treat them as disposable WIP commits and squash/rebase before integration;
- Grant decides when a visual direction is worth preserving.

Suggested meaningful checkpoints might include:

1. Vue foundation / behavior parity;
2. spatial board + hierarchy;
3. card redesign;
4. interaction/motion polish;
5. final K94 acceptance.

This is guidance, not a required commit count.

---

## 24. Frontend-design assistance

If the Claude Code frontend-design capability/plugin is available, use it.

Its role is to improve:

- composition;
- typography;
- hierarchy;
- spacing;
- interaction taste;
- motion;
- visual distinctiveness.

It does not override Victory product doctrine.

Avoid generic AI-generated dashboard aesthetics.

If the tool suggests a visually striking direction that conflicts with board usability or domain structure, preserve usability and structure.

---

## 25. No backend rewrite without evidence

Before changing storage or API contracts, prove why the current contract blocks the desired experience.

Backend changes are allowed for:

- correctness;
- missing persistence required by the redesigned interaction;
- serialization defects;
- clear performance blockers;
- accessibility/interaction needs that cannot be represented otherwise.

Do not redesign the Storyboards backend because the frontend is being rebuilt.

---

## 26. Existing behavior audit

Before implementation, inspect at minimum:

1. current Storyboards frontend entry point;
2. board loading;
3. Storyboard/Timeline APIs;
4. serialization;
5. Time Frame ordering;
6. Event ordering;
7. Moment ordering;
8. card creation/edit/delete;
9. color persistence;
10. flip behavior;
11. drag/reorder;
12. Presence/coordination controls;
13. permissions by role;
14. realtime/network updates;
15. protected Ending behavior;
16. existing Vue infrastructure elsewhere in Victory;
17. shared frontend components/styles worth reusing;
18. current realistic large Storyboards data.

Do not ask Grant questions the repository can answer.

---

## 27. Role and permission preservation

The visual rebuild must not widen authority.

Preserve current role rules.

At minimum verify:

- Audience/Cast remain view-only where currently intended;
- Crew can perform only currently authorized creation/edit actions;
- Director+ retain editor authority;
- Owner/Operator authority remains intact;
- hidden controls are not the security boundary.

Do not perform Kernel 97’s full authority audit here.

If the rebuild exposes a clear permission bug, make the smallest canonical fix and record the broader issue for K97.

---

## 28. Accessibility baseline

K94 must not trade accessibility for beauty.

Required baseline:

- semantic buttons/inputs;
- visible keyboard focus;
- keyboard access to critical operations;
- labels/tooltips where icon meaning is not obvious;
- readable contrast;
- reduced-motion support;
- no information encoded only by color where practical.

Do not let accessibility force the board back into an ugly form-heavy interface.

---

## 29. Destructive actions

Deletion should be visually clear and appropriately difficult to trigger accidentally.

Use the existing Victory confirmation doctrine where one exists.

Avoid:

- tiny adjacent destructive icons;
- invisible right-click-only deletion;
- immediate irreversible deletion with no confirmation if the current product expects confirmation.

Do not add elaborate trash/archive systems unless already present.

---

## 30. Visual polish debt policy

During K94, distinguish:

- **blocking visual defects** — prevent successful use/presentation;
- **meaningful polish** — materially improves the board;
- **micro-polish** — tiny spacing/animation/detail issues.

Fix blocking defects.

Spend meaningful time on meaningful polish.

Do not burn the kernel on endless micro-polish.

If a tiny visual effect repeatedly fails, simplify it and move on.

---

## 31. Alpha-tester readiness

K94 should leave Storyboards suitable for near-term alpha exposure.

That does not mean feature complete.

It means an alpha tester should encounter something that feels intentionally designed rather than obviously transitional.

Before closing, a fresh user should be able to:

- understand that they are looking at a Storyboard;
- understand how to begin;
- create content;
- manipulate it;
- read a populated board;
- recover after refresh;
- distinguish structure;
- avoid obvious visual dead ends.

Do not redesign onboarding globally; K91/K95 own broader teaching surfaces.

---

## 32. Three required acceptance states

### State A — Empty board

Grant sees an empty Storyboard and judges:

- inviting;
- spacious;
- clear first action;
- no dashboard feeling;
- no intimidating wall of controls.

### State B — Working board

Grant uses a moderately populated Storyboard and judges:

- manipulation is clear;
- editing is quick;
- structure remains understandable;
- controls stay out of the way;
- dragging/movement feels good enough;
- motion helps rather than irritates.

### State C — Full board

Grant views a dense/finished Storyboard and judges:

- it feels composed;
- it remains readable;
- the content dominates the interface;
- it feels suitable for presentation;
- it resembles a finished digital showing space rather than a database editor.

All three states matter.

---

## 33. Functional regression proof

At minimum prove:

- board loads;
- new Storyboard can be created if currently supported;
- card/content creation works;
- edits persist;
- colors persist;
- front/back/flip persists where applicable;
- movement/reorder persists;
- Time Frames remain structurally correct;
- Events remain structurally correct;
- Moments remain structurally correct;
- protected structure remains protected;
- permissions remain enforced;
- reload restores state;
- network update still works where currently supported;
- large board remains usable.

Automated tests are supporting evidence.

Browser proof is required.

---

## 34. Human acceptance gate

Do not close K94 based only on:

- unit tests;
- build success;
- lint;
- screenshots generated by the agent;
- “looks modern” self-assessment.

Grant must inspect the real running interface.

The kernel passes only after Grant accepts all three required visual states.

---

## 35. Agent operating rule

The agent should work in bounded passes.

Recommended sequence:

### Pass 1 — Audit + parity
Understand existing behavior and establish Vue foundation without losing capability.

### Pass 2 — Spatial composition
Rework board structure, hierarchy, whitespace, and empty state.

### Pass 3 — Card system
Redesign cards, editing, focus, selection, and front/back behavior.

### Pass 4 — Interaction + motion
Improve drag/drop, contextual controls, transitions, and tactile feedback.

### Pass 5 — Dense-board exhibition pass
Use a genuinely populated board and make the full composition feel presentable.

### Pass 6 — Grant critique
Stop. Let Grant inspect. Apply bounded corrections.

Do not autonomously invent Pass 7 through Pass 35 because perfection remains possible.

---

## 36. Decision memory

Locked product decisions for K94:

- Empty Storyboards should feel like **space to create**.
- Full Storyboards should feel like **a masterpiece in a digital showing space**.
- Free placement within a cell/region is acceptable.
- Snapping is acceptable and preferred if it avoids needless implementation complexity.
- Cards should receive a substantial redesign.
- Motion may be fun.
- Motion should be simplified rather than becoming a troubleshooting sink.
- K94 is the deep Storyboards beautification/rebuild.
- K95 carries successful visual language across the rest of Victory.
- Vue is intended to support a more fluid experience, not to become the point of the kernel.
- The coding workflow may use dirty local iterations and meaningful checkpoints instead of committing every visual tweak.

---

## 37. Out of scope

Do not turn K94 into:

- full-site beautification;
- complete Victory design-system migration;
- Figma dependency;
- backend Storyboards rewrite;
- universal freeform canvas;
- full mobile Storyboards redesign;
- Kernel 97 role/authority audit;
- Kernel 95 continuity/event-surfacing work;
- slash-command training;
- broad new Storyboards gameplay mechanics;
- new generic card abstraction.

If a needed adjacent fix is small and directly blocks Storyboards, fix it.

Otherwise record it.

---

## 38. Reportback

Report:

1. what was rebuilt in Vue;
2. what existing behavior was preserved;
3. backend/API changes, if any, and why they were necessary;
4. Storyboards visual-design decisions;
5. card-system changes;
6. placement/drag model;
7. motion added and any motion deliberately simplified;
8. accessibility changes;
9. correctness bugs fixed;
10. known residual defects;
11. items explicitly deferred to K95;
12. automated proof;
13. browser proof;
14. Grant acceptance result for:
   - Empty board;
   - Working board;
   - Full board.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

Do not call PASS until Grant has accepted the actual browser experience.

---

## 39. Completion condition

Kernel 94 passes when Storyboards has crossed the line from:

> **functional board software**

into:

> **a creative workspace people want to build in, and a digital artifact people may want to show.**

The work should look like the reason the software exists.

The software should not look like the reason the work exists.
