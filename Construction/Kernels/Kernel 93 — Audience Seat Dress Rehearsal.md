# Kernel 93 — Audience Seat Dress Rehearsal

**Status:** READY FOR LIVE DESIGN / IMPLEMENTATION  
**Type:** Audience admission + live experience audit + bounded polish  
**Sequence position:** After Kernel 92  
**Primary proof:** Grant can enter one Showing as a real Audience member and watch Victory start-to-finish, with the coding agent fixing only bounded issues discovered during the run  
**Core doctrine:** Audience is king. Victory should support the theater in their minds, not compete with it.

---

## 0. Kernel mode

Kernel 93 is intentionally different from a normal implementation kernel.

Do **not** receive this file, disappear for hours, and attempt to autonomously perfect the entire Audience experience.

The working mode is:

1. establish the minimum legitimate Audience seat;
2. stop and hand control to Grant;
3. Grant watches Victory as Audience;
4. record feedback as it occurs;
5. classify feedback;
6. fix bounded issues from the current pass;
7. rerun;
8. advance to the next pass.

The human Audience experience is primary evidence.

Automated tests prove authority, projection, persistence, and regressions. They do not decide whether the performance is pleasant to watch.

---

## 1. First blocker: a legitimate Audience seat

Kernel 90 proved a pure Audience account cannot currently enter Catharsis through the ordinary venue gate.

Kernel 93 must establish a canonical single-Showing Audience admission primitive.

This is **not** the existing two-punch campus/access ticket system.

Do not conflate them.

Conceptually:

> **Audience Showing Ticket** — admits one user to one Showing, as Audience, into the Showing's live venue.

Kernel 96 owns the full Ticket Booth / parking-lot-to-seat journey.

Kernel 93 only needs enough legitimate admission to let Grant become Audience and test the performance.

---

## 2. Existing two-punch tickets remain distinct

Preserve the current two-punch ticket/access system.

Do not rename or silently repurpose it.

The new/rectified Audience Showing admission must have a clear domain distinction.

Audit existing Show, Showing, ticket, admission, access-grant, and venue-entry models before adding storage.

Reuse an existing single-Showing admission primitive if one already exists.

Do not create duplicate ticket truth.

---

## 3. How Grant becomes Audience

By the end of setup, the operator must have a simple documented path:

1. create/select a Showing;
2. issue or assign a single-Showing Audience admission to an account;
3. log in as that account;
4. enter the correct live venue as Audience;
5. receive Audience projection, not Cast projection.

This is the first acceptance gate.

Stop broad implementation if this does not work.

---

## 4. Audience projection controls per Showing

Director gets one simple configuration surface for Audience projection.

Required toggles:

- **Dice Rolls** — ON / OFF
- **Presence** — ON / OFF
- **Character Health Statuses** — ON / OFF

Use obvious pressed/depressed toggle controls.

Do not create a dense settings form.

Configuration is per Showing.

Persist it.

---

## 5. Defaults

Initial recommended behavior:

- Dice Rolls: preserve current public behavior unless repository evidence strongly indicates a better existing default.
- Presence: configurable, but Audience Presence **starts collapsed** when enabled.
- Character Health Statuses: configurable.

Do not expose exact health numbers to Audience.

Health Status means qualitative/public condition only.

---

## 6. Audience left overlay

Audience should have a left-side overlay that fits the same broad spatial language already used for Actors/Cast, but remains simpler.

At minimum:

### Diagnostic
Available where current product conventions permit.

### Presence
- only visible if enabled for this Showing;
- starts collapsed;
- may be opened locally by the Audience member.

### Character Health Status
Only surfaced when enabled for the Showing and only at the qualitative/public level.

Do not expose Director controls or hidden mechanics.

---

## 7. Reactions and chat

Audit what already exists before adding anything.

Product direction:

- Audience reactions live in the chat space;
- reactions may briefly float/disrupt chat presentation;
- reactions must not cover or interrupt the actual Pixi stage;
- if the Audience user closes/hides chat locally, they may stop seeing reactions/chat entirely.

Do not create a second stage-wide reaction projection.

Do not invent new reaction types merely to make this kernel feel featureful.

---

## 8. Silence is allowed

When no stage object is moving, Victory does not need to manufacture stimulation.

The Audience may be watching:

> **the theater in their minds.**

Do not add animation, speaker meters, pulsing UI, or fake activity merely because narration continues without Pixi movement.

Victory should know when to get out of the way.

---

## 9. Audience information boundary

Audience may see public theatrical information such as:

- Character identity/Face where intentionally projected;
- public dice rolls if enabled;
- qualitative Character health status if enabled;
- public Scene/stage state;
- public announcements;
- Audience-visible Presence if enabled;
- chat/reactions where applicable.

Audience must not receive:

- exact HP/pool values;
- Director-only state;
- hidden objects;
- private Character mechanics;
- backstage notes;
- Director preparations;
- private Cohort/Character visibility;
- Director controls.

Reuse Kernel 90 canonical projection.

Do not rely on CSS concealment.

---

## 10. Pass-based workflow

Maintain a live issue/decision ledger during this kernel.

Once Grant answers a product question, record the answer.

Do **not** re-ask the same underlying question in later passes merely because the feedback becomes relevant again.

If Grant gives feedback belonging to a later pass, record it under that pass and continue the current pass unless it is blocking.

---

## 11. Feedback classification

Every observation during live testing should be classified as one of:

- BUG
- CLARITY
- VISUAL POLISH
- ATTENTION / THEATRICAL DIRECTION
- AUDIENCE MISSING INFORMATION
- AUDIENCE SEEING TOO MUCH
- ACCESS / ADMISSION
- RECONNECT / CONTINUITY
- ENDING / CLOSURE
- FUTURE FEATURE — NOT K93

Do not immediately implement FUTURE FEATURE items.

---

## 12. Pass A — Can I watch this?

Goal:

> The Audience can enter, understand the basic frame, and follow what is happening.

Test sequence should include:

- admitted Audience enters venue;
- Showtime begins;
- Character(s) present;
- ordinary narration/chat;
- a public roll;
- Director announcement;
- a reaction;
- a visibility Cue/reveal;
- a Scene transition.

Grant reports confusion/friction as it occurs.

Fix only bounded clarity/bug issues discovered in this pass.

---

## 13. Pass B — Is it enjoyable to watch?

Goal:

> The interface supports performance rather than demanding attention.

Focus on:

- visual hierarchy;
- stage dominance;
- chat/reaction behavior;
- left overlay weight;
- Presence behavior;
- roll presentation;
- announcements;
- Scene transitions;
- Character identity clarity;
- excess chrome;
- ugly/debug-looking surfaces.

This pass may include meaningful visual polish.

Do not redesign unrelated Director/Cast tools.

---

## 14. Pass C — Late arrival and reconnect

Goal:

> An Audience member who arrives late or reconnects can understand where they are.

Test:

- join after Showtime already began;
- reload;
- disconnect/reconnect;
- leave venue and return if supported;
- Scene changes while disconnected if practical.

Verify:

- correct Showing;
- correct Scene;
- correct public objects;
- no hidden-state leak;
- projection toggles still apply;
- Presence state is sensible;
- chat/reaction continuity is understandable under current product model.

Fix bounded continuity issues.

---

## 15. Pass D — Ending

Goal:

> A Showing ends like theater, not like a server socket disappearing.

Test End Showtime.

Observe:

- what Audience sees immediately before end;
- what happens at end;
- whether stage/chat suddenly vanish;
- whether Audience understands that the Showing has concluded;
- whether there is an appropriate route away/back;
- whether Showing Review or existing post-Showing surfaces make sense.

Aftercare remains primarily a Cast/participant care operation unless current product rules say otherwise.

Do not automatically include Audience in Aftercare.

---

## 16. Director Audience configuration

Director needs one easy Audience configuration control for the Showing.

It should be reachable without hunting.

Preferred UI:

```text
Audience
  [ Dice Rolls      ]
  [ Presence        ]
  [ Health Statuses ]
```

Each button visually indicates ON/OFF through depressed/pressed state plus accessible text/state.

Do not add twenty Audience toggles.

Kernel 93 may add another toggle only if live testing demonstrates a clear, repeated need and Grant explicitly approves it.

---

## 17. Presence

Audit existing Presence behavior and projection.

Audience Presence should:

- fit into the left-side overlay;
- begin collapsed when enabled;
- remain absent when disabled;
- not expose backstage coordination;
- not imply Audience has Cast authority.

Do not fork the Presence system.

---

## 18. Dice

When Audience Dice Rolls are OFF:

- do not project/show the Audience roll presentation.

When ON:

- show only rolls already appropriate to public Audience projection;
- preserve hide-the-ball mechanics;
- do not reveal private modifiers/mechanics merely because a die is public.

Do not create a separate dice engine.

---

## 19. Health Statuses

When OFF:
- Audience sees no Character health-status presentation.

When ON:
- Audience may see qualitative/public health status only.

Exact values never become Audience-visible through this setting.

Reuse Kernel 88 qualitative projection where appropriate.

---

## 20. Showing-level persistence

Audience projection settings belong to the Showing/event scope.

They must survive:

- reload;
- reconnect;
- live Session restart/resume where the same Showing continues.

Do not make them global account preferences.

Do not make them Scene state.

---

## 21. Director preview

If an existing truthful Audience preview is available, use it.

It must not replace the real Audience browser/account as acceptance evidence.

This kernel is specifically about occupying the Audience seat.

---

## 22. Ticket Booth boundary

The physical Ticket Booth belongs in the parking lot and will become part of the larger Audience arrival journey.

Kernel 93 does not need to build the final Ticket Booth UX.

Kernel 96 will audit:

> parking lot → ticketing → admission → venue → seat.

For 93, provide/document the minimum legitimate Audience Showing admission necessary to test.

---

## 23. Info Booth boundary

Audit what the Info Booth currently provides.

Do not repurpose it into a Ticket Booth merely because admission is currently awkward.

Preserve venue identities.

---

## 24. Role/access bug boundary

Kernel 90 found broader role-resolution and pure-Audience venue-gate problems.

Fix the **smallest canonical access path required for a valid Audience Showing ticket to admit its holder to the correct venue**.

Do not perform Kernel 97’s complete authority audit here.

Record adjacent role inconsistencies for 97.

---

## 25. Showtime integration

Use Kernel 92 Showtime.

Audience admission should align with the selected Showing.

Do not invent a second live-event selector.

A ticket/admission for Showing A must not grant Audience access to unrelated Showing B merely because they share a venue.

---

## 26. Showing configuration model guidance

Prefer one canonical Showing-level Audience configuration object.

Conceptually:

```text
AudienceProjectionConfig
- showing_id
- show_dice_rolls
- show_presence
- show_health_statuses
- updated_by
- updated_at
```

Use existing Show/Showing tables if a clean extension is preferable.

Do not create a generic settings engine.

---

## 27. Audience admission model guidance

Conceptually:

```text
AudienceAdmission
- user_id
- showing_id
- role = audience
- issued_at
- redeemed/admitted state if existing ticket model requires it
```

This is conceptual only.

Audit current ticket/admission schema first.

The goal is single-Showing Audience authority, not a new commerce system.

---

## 28. Security proof

Required:

- Audience ticket for Showing A cannot enter Showing B;
- pure Audience can enter the correct venue for Showing A;
- Audience receives Audience projection;
- Audience cannot mutate Director configuration;
- Audience cannot obtain hidden stage objects;
- Audience cannot obtain exact health;
- toggling Presence does not grant Cast coordination authority;
- disabling Dice removes Audience roll projection;
- Director/Operator authority remains intact.

---

## 29. Automated proof is supporting evidence

Automated tests should cover:

- admission authority;
- Showing scoping;
- Audience config persistence;
- projection toggles;
- hidden-data omission;
- reconnect restoration.

Do not use a giant Playwright checklist as a substitute for Grant’s actual watching pass.

If Playwright becomes flaky after the human-visible product behavior is already proven, document the test limitation and move on.

---

## 30. Agent operating rule

During live passes:

> **Do not fix things merely because you noticed them.**

When an issue appears:

1. record it;
2. classify it;
3. determine whether it blocks the current pass;
4. ask/observe Grant’s direction if it is a product choice;
5. make the smallest relevant fix;
6. report what changed;
7. return control to Grant.

Do not disappear into unrelated refactors.

Do not chase test flakiness for hours.

---

## 31. Live ledger

Create:

`Construction/OperatorLogs/kernel-93-audience-dress-rehearsal-ledger.md`

Suggested format:

```text
ID:
Pass:
Observation:
Classification:
Decision:
Action:
Status:
Files changed:
Retest result:
Deferred kernel, if any:
```

This ledger is the working heart of Kernel 93.

---

## 32. Decision memory inside the kernel

Record explicit Grant decisions so they are not resurfaced later.

Initial locked decisions:

- Audience exists without a single pre-defined market/customer model; build the theater and learn from use.
- Full Ticket Booth journey is Kernel 96.
- Reactions/chat belong in chat space, not over the actual stage.
- Closing chat may mean not seeing reactions/chat.
- Audience exact health values are hidden.
- Showing-level toggles: Dice Rolls, Presence, Character Health Statuses.
- Presence starts collapsed when enabled.
- The stage does not need artificial activity during narration.
- Feedback may arrive early for later passes; record it rather than re-asking settled questions.

---

## 33. Repository audit before implementation

Inspect:

1. current two-punch ticket system;
2. any existing Showing ticket/admission model;
3. ticket second-punch behavior;
4. venue access resolution;
5. Kernel 90 Audience projection;
6. Show/Showing identifiers from Kernel 92;
7. Audience role resolution;
8. Presence UI/data;
9. left-side Cast overlay;
10. Audience overlay if any;
11. dice Audience projection;
12. qualitative health projection;
13. chat/reactions;
14. Showtime start/end behavior;
15. Showing Review/end surfaces;
16. Info Booth;
17. Ticket Booth placeholders/assets if any.

Do not ask Grant repository-answerable questions.

---

## 34. Setup acceptance gate

Before the live dress rehearsal begins, prove:

1. a Showing exists;
2. Director can issue/assign Audience admission;
3. Audience account can enter the correct venue;
4. Audience account is truly resolved as Audience;
5. Audience sees no Director/Cast-only controls;
6. Director can toggle Dice/Presence/Health Status;
7. Audience projection changes accordingly;
8. Presence begins collapsed if enabled.

Then **STOP and hand control to Grant**.

Do not autonomously run the rest of the kernel.

---

## 35. Pass criteria

Kernel 93 passes when:

- legitimate single-Showing Audience admission exists;
- existing two-punch tickets remain distinct;
- Grant can become Audience without a role hack;
- Audience config has Dice/Presence/Health toggles;
- toggles are Showing-scoped;
- Presence starts collapsed;
- Audience left overlay is coherent;
- reactions/chat remain off-stage;
- exact health remains hidden;
- hidden objects remain hidden;
- Pass A is completed with bounded fixes;
- Pass B is completed with bounded polish;
- Pass C is completed;
- Pass D is completed;
- decisions are recorded and not repeatedly reopened;
- deferred feature ideas are captured rather than implemented by drift;
- Audience experience is coherent enough that Grant can actually watch the performance rather than operate the software.

---

## 36. Partial criteria

Mark PARTIAL if:

- Audience still requires a fake Cast membership hack;
- Showing ticket/admission is not truly Showing-scoped;
- Audience configuration exists only client-side;
- Presence cannot be disabled/collapsed;
- health toggle leaks exact values;
- Audience view is technically secure but too confusing to watch;
- reconnect loses the Showing;
- ending is incomprehensible;
- agent completes setup but live human passes are not performed.

---

## 37. Fail criteria

Mark FAIL if:

- new Audience ticket silently replaces/reinterprets the existing two-punch ticket;
- Audience admission grants broad venue access outside the Showing;
- Audience can see Director/backstage state;
- hidden content is merely CSS-hidden but sent to Audience;
- Scene state is forked to implement Audience settings;
- reaction effects cover the live stage;
- agent turns Kernel 93 into a broad autonomous redesign;
- Grant loses operator access.

---

## 38. Immediate operator outcome

Before continuing to Kernel 94, Grant should be able to answer:

1. How do I become Audience?
2. Can I enter with a legitimate single-Showing Audience admission?
3. Can I watch without seeing backstage controls?
4. Can the Director turn public dice on/off?
5. Can the Director turn Presence on/off?
6. Can the Director turn qualitative health statuses on/off?
7. Does Presence start collapsed?
8. Do reactions live in chat rather than over the stage?
9. Can I close chat and simply watch?
10. Can I understand what is happening?
11. Does the interface get out of narration’s way?
12. Can I arrive late?
13. Can I reconnect?
14. Does the end of the Showing make sense?
15. Did we fix what actual watching exposed rather than inventing hypothetical Audience features?

Kernel 93 succeeds when Victory has a real **Audience seat**, and that seat is pleasant enough to sit in.
