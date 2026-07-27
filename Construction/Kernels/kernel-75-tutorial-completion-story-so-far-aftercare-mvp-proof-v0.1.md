# Kernel 75 — Tutorial Completion, Story So Far, Aftercare, and MVP Proof

**Revision:** 0.1
**Primary track:** MVP completion / Golden Journey closure
**Depends on:** Kernel 74 and all prior Golden Journey kernels
**Expected first migration:** `069`, but implementation must confirm both `backend/migrations/` and the live `schema_migrations` ledger immediately before naming files.

> **Note on revisions.** This is the operator-issued revision and is authoritative.
> `kernel-75-tutorial-completion-aftercare-continuation-v0.1.md` is the earlier
> builder-drafted cut, retained for provenance. Where the two disagree, this
> document wins. The earlier draft has no §5.3, §12, or §13.

---

## 0. Purpose

Kernel 75 closes the MVP strand.

It completes the automated tutorial, turns recorded play into durable Character and Player history, collects optional Aftercare, delivers that information to the Director's Chair, and proves the complete first-time journey from sign-up through waiting for scheduled human play.

The intended flow is:

```text
Ra dialogue already open
→ Ra reveals and operates the concealed lock
→ Ra finishes opening the gate
→ Player clicks Continue
→ participant-local tutorial-complete projection opens
→ Player reviews a deterministic Story So Far
→ Player may Save and Close Aftercare or explicitly Skip
→ Player may close the full-screen panel and inspect Catharsis
→ Player remains in the tutorial-handoff state
→ future Director-controlled shared Scene advancement clears the local projection
```

The completion screen must make the boundary explicit:

```text
The automated beginning is complete.
Your Character, equipment, choices, and progress are saved.
The story continues with real people during a scheduled Session.
```

Kernel 75 is the MVP finish line for the first guided Socio journey.

It is not the end of Victory development.

---

## 1. Locked product decisions

### 1.1 Ra completes the gate-opening beat before tutorial completion

The tutorial-complete projection must not interrupt Ra's final physical action.

Sequence:

```text
Ra dialogue topics complete
→ Ra's portrait/card remains open
→ closing narration reveals the concealed lock
→ Ra operates the key/lock mechanism
→ gate opens
→ Continue becomes available
→ Player clicks Continue
→ tutorial-complete projection opens
```

The closing narration should preserve the established interpretation:

- no obvious lock or keyhole was visible at first;
- Ra exposes a recessed or concealed iron mechanism on the courtyard side;
- the gate visibly opens before the completion screen takes over.

### 1.2 Tutorial-complete projection is participant-local

The completion projection is private to the Player who completed the tutorial.

It must:

- survive refresh/reconnect;
- not change `shows.current_show_scene_placement_id`;
- not move other Players;
- not alter Audience projection;
- clear when a later Director-controlled shared Scene is flown;
- remain inspectable until then.

### 1.3 The Player may close the full-screen panel

After reaching the tutorial-complete projection, the Player may close the full-screen panel and inspect Catharsis.

They should be able to inspect:

- Character creation and Character controls;
- Character Inventory;
- Story So Far;
- My People;
- journal tools;
- ordinary Catharsis UI.

Closing the panel must not return the Player to the Courtyard fiction or erase completion.

### 1.4 Human continuation is explicit

Victory should assume that a Narrator is usually absent until a Session is scheduled.

The waiting state should say plainly:

> You have completed Victory's automated Socio tutorial.
> Your progress is saved.
> The story continues with real people during a scheduled Session.

Do not imply that an unattended Director is about to appear immediately.

### 1.5 Setting neutrality

The tutorial may introduce Niava and the Crown Bet, but the completion screen must not assume that the next shared Scene is Niava.

The Player may continue into any setting the Narrator chooses.

### 1.6 Deterministic reflection only

There is no AI author inside Victory for this feature.

Personalized completion text must be generated from deterministic templates and recorded data.

Do not call external AI services.

Do not invent unsupported Character traits or motivations.

### 1.7 Archetype selects the primary Character lens

When choosing the Character's primary strength for the Story So Far summary:

```text
Character archetype
→ mapped primary attribute
→ matching life-stage or Face Sheet history line
```

If the archetype mapping is absent or invalid, use the highest attribute.

If several attributes tie, prefer the attribute connected to the Character's archetype.

If the relevant Face Sheet line is missing, omit that quotation rather than fabricating one.

### 1.8 Story So Far is separate from authored identity

Generated play history belongs in a dedicated:

```text
Story So Far
```

Do not merge generated events indistinguishably into Player-authored Face Sheet history.

Story So Far entries are:

- private by default;
- optionally revealable later;
- chronologically ordered;
- sourced from durable play events or explicit Director journal entries.

### 1.9 Director-authored history uses the existing journal system

Audit and reuse the existing `/journal` system and related journal storage where sound.

Directors+ should be able to create a Character Story So Far entry through the established journal path or the smallest compatible extension.

Do not create a second unrelated Director-note history system if `/journal` can support the need.

### 1.10 Leave room for gamification without inventing it

K75 must leave clean extension points for:

- points;
- badges;
- titles;
- unlocks;
- participation streaks;
- later advancement systems.

For MVP, store semantic milestones and extensible award records.

Do not invent a numeric economy, level curve, or reward values.

### 1.11 Tutorial replay

The guided tutorial may be completed again with another Character.

Player-level first-time recognition must not be awarded repeatedly.

Character-level completion is per Character.

### 1.12 Aftercare is qualitative

Do not add numerical ratings.

The three prompts are:

1. **What were your favorite moments from the Show?**
2. **Who surprised you the most?**
3. **What do you hope to see in the next Session?**

Question two should support selecting a relevant person or Character plus optional freeform explanation.

### 1.13 Aftercare is optional and saveable

No prompt is mandatory.

Primary actions:

```text
Save and Close
Skip
```

`Save and Close` may submit partial responses.

`Skip` means the Player chose not to answer at this opportunity.

Closing the window without pressing `Skip` must not increment the skip counter.

Draft text should persist until saved, skipped, or intentionally discarded.

### 1.14 Skip warning behavior

Track consecutive explicit skips.

On `Skip`:

- show the current consecutive-skip context;
- require the Player to choose:
  - **Continue Without Aftercare**
  - **Go Back**

Examples:

```text
You skipped Aftercare last time.
```

```text
You skipped the last two Aftercare check-ins.
```

```text
Are you sure you want to skip again?
This will be your third consecutive skip.
```

Completing any partial Aftercare submission resets the consecutive-skip count to zero.

The warning is factual, not punitive.

### 1.15 Director's Chair is read-only

Aftercare belongs in the Director's Chair.

Director+ may:

- view;
- filter;
- sort;
- export.

Director+ may not reply, annotate, score, or open a comment thread through this MVP surface.

### 1.16 My People is the default note follow-up

After Aftercare, show:

> Did you take notes? Don't forget to update your notes on yourself and your fellow Players.

Primary follow-up:

```text
Update My People
Next
```

The Character journal may also be available elsewhere in Catharsis, but the default reminder action goes to My People.

### 1.17 Director continuation remains future-configured

K75 does not invent a special tutorial continuation control.

A later Director-controlled shared Scene change will use the ordinary Scene/Cue system.

### 1.18 Desktop is the MVP target

Required:

- desktop browser;
- controlled Discord walkthrough;
- multiple accounts/windows where needed.

Mobile is not a full venue-navigation target for this kernel.

### 1.19 Advertising target

The immediate audience is a friend or interested person receiving a controlled Discord tour.

The first advertising step is:

- demonstration video;
- interest sign-ups;
- controlled walkthroughs;
- no immediate open public access requirement.

---

## 2. Required pre-implementation audit

Before implementation, audit:

- actual next migration number;
- K74 Ra dialogue closing path;
- K74 participant-local projection;
- tutorial completion records;
- Character Face Sheet model;
- archetype storage and archetype-to-attribute mapping;
- life-stage lines and Face Sheet feed lines;
- existing `/journal` command and journal storage;
- current Character activity/history models;
- Trailer Face reveal/privacy rules;
- Player-level account milestones;
- Show/Session participation history;
- Director's Chair venue and permissions;
- export/spreadsheet utilities already present;
- My People routing;
- local projection clear behavior;
- Character-switch behavior;
- current tutorial placeholder assets and copy.

The reportback must state:

1. which existing models were reused;
2. whether Story So Far required a new table;
3. how public/private reveal is represented;
4. how archetype maps to the reflection template;
5. how first-time Player awards differ from per-Character awards;
6. how Aftercare is organized for Director's Chair;
7. how `/journal` participates;
8. what remains deliberately unimplemented.

---

## 3. Ra closing sequence

### 3.1 Closing narration

After required Ra topics are viewed, `Leave Ra` should first advance Ra's existing card into the final narrated beat.

Required content:

- Ra exposes the concealed lock;
- Ra operates it with a key or physical mechanism;
- the bolt releases;
- the gate opens;
- the threshold beyond becomes visible.

Use operator-edited copy where supplied.

Do not open the tutorial-complete projection until the Player presses:

```text
Continue
```

### 3.2 Continue behavior

`Continue`:

- records the final Ra/gate-open milestone;
- activates or updates the existing participant-local tutorial-handoff projection;
- opens the full-screen Tutorial Complete Program;
- does not alter the shared current Scene;
- is retry-safe and idempotent.

---

## 4. Tutorial-complete Program

### 4.1 Required sections

The full-screen completion Program should contain:

```text
Tutorial Complete
What You Did
Story So Far
Progress Earned
Aftercare
Waiting for Human Play
```

The Player may move between sections without losing data.

### 4.2 Required completion message

Use wording materially equivalent to:

> You completed Victory's automated Socio tutorial.
>
> You created and equipped a Character, entered a shared social space, made meaningful choices, and carried those choices into a continuing history.
>
> Your Character, equipment, and progress are saved.
>
> The next part of the story is played with real people during a scheduled Session. Your Narrator may continue into Niava or any other setting the table chooses.

The tone is reflective, not celebratory confetti.

### 4.3 Close/minimize behavior

The Player may close the full-screen Program and explore Catharsis.

Provide a persistent, unobtrusive way to reopen:

```text
Tutorial Complete
```

or:

```text
View Story So Far
```

Do not force the completion Program to remain modal while waiting.

---

## 5. Deterministic personalized "You did this" reflection

### 5.1 Source data

The summary may draw only from:

- Character name;
- archetype;
- archetype-linked attribute;
- highest attribute values;
- matching life-stage Face Sheet line;
- Kessa stance attempts;
- Haggle result;
- equipment acquired;
- door intention;
- Ra topics completed;
- tutorial milestones;
- durable Show participation events.

### 5.2 Template behavior

Use deterministic sentence templates.

Omit missing clauses.

Never generate an unsupported psychological conclusion.

Suggested structure:

```text
[Character] entered the Courtyard as [archetype].

Your strongest lens was [attribute], a pattern already visible in your history:
"[matching Face Sheet line]"

At Kessa's stall, you [recorded Kessa behavior].
You equipped yourself with [selected equipment summary].

At the locked gate, you intended to:
"[door intention]"

Ra interrupted before the attempt resolved and introduced the Crown Bet.

You completed your first guided Socio Show.
```

### 5.3 Draft template library

Provide deterministic templates for at least:

- archetype identified;
- archetype missing;
- one highest attribute;
- tied highest attributes with archetype resolution;
- matching Face Sheet line present;
- matching line absent;
- no purchase;
- one acquired item;
- several acquired items;
- no Kessa stance;
- stance attempted;
- Haggle attempted;
- door intention present;
- tutorial completed.

### 5.4 Example outputs

#### Example A — complete data

> **Trang's Story So Far**
>
> Trang entered the Courtyard as **the Observer**, led most strongly by **Empathy**.
>
> That strength was already visible in Trang's history:
>
> *"You learned early to notice who was left outside the group."*
>
> At Kessa's stall, Trang approached through Insight and equipped themself with a healer's kit and traveling cloak.
>
> At the gate, Trang intended to:
>
> *"I study the hinges and test whether the door can be lifted instead of forced."*
>
> Ra interrupted before the attempt resolved and introduced the Crown Bet.
>
> Trang has completed the guided Socio tutorial. These events are now part of their continuing history.

#### Example B — sparse data

> **Mara's Story So Far**
>
> Mara entered the Courtyard as **the Guardian**.
>
> At Kessa's stall, Mara prepared for what might come next.
>
> At the gate, Mara chose an approach of their own before Ra interrupted.
>
> Mara has completed the guided Socio tutorial. Their Character and progress are saved.

#### Example C — no purchase

> Kai spoke with Kessa but left the stall without taking new equipment. That choice is part of Kai's history; preparation did not require a purchase.

### 5.5 No editing of generated history

The Player cannot directly edit generated Story So Far text.

They may:

- edit their Character Face Sheet through normal Character tools;
- add their own journal/reflection afterward;
- control later public reveal of individual Story So Far entries.

---

## 6. Story So Far

### 6.1 Dedicated Character section

Add a dedicated Character page or subpage:

```text
Story So Far
```

Entries are chronological and durable.

Each entry should have:

- Character;
- event type;
- title;
- summary;
- Show Run;
- Show;
- Session where applicable;
- Scene where applicable;
- source event or journal reference;
- timestamp;
- privacy/reveal state;
- authored-by identity where relevant.

### 6.2 Initial event coverage

Audit and ensure meaningful events create entries or source feed lines for:

- Character created;
- archetype established;
- major life-stage line completed;
- Trailer Face completed;
- joined Show Run;
- selected for Show;
- Kessa stance attempted;
- Haggle result;
- equipment acquired;
- Kessa completed;
- door intention submitted;
- Ra introduction completed;
- tutorial completed;
- Aftercare submitted;
- Aftercare skipped;
- Director-authored journal moment.

Do not create a feed entry for every click.

### 6.3 Privacy

Default:

```text
private
```

Support a future-compatible reveal field or state.

Do not automatically expose generated play history on Trailer Face.

### 6.4 Director-authored entries

Reuse `/journal` where possible.

A Director-authored Character moment should preserve:

- Director author;
- Character target;
- Show/Session context;
- text;
- timestamp;
- private-by-default state;
- source=`director_journal`.

Do not allow Directors to silently rewrite Player-authored Face Sheet material.

---

## 7. Progression and recognition foundation

### 7.1 Required semantic milestones

Record:

#### Character

- `completed_locked_courtyard`
- `completed_guided_socio_tutorial`
- `first_show_completed`
- tutorial completion timestamp

#### Player

- `completed_first_socio_show`
- `completed_guided_socio_tutorial`
- participation count increment
- first-completion timestamp

### 7.2 Extensible awards

Provide a bounded extensible award model supporting future:

```text
points
badge
title
unlock
streak
```

For K75, seed or award only semantic MVP recognition.

Recommended initial recognition:

```text
Character milestone: Completed the Locked Courtyard
Player milestone: Completed First Socio Show
Title/badge: First Curtain
Unlock flag: Guided tutorial completed
```

The operator may rename the badge/title before deployment.

### 7.3 First-time versus repeat

- Character completion is per Character.
- Player first-show recognition is awarded once.
- A second Character can complete the tutorial and receive Character milestones.
- Repeated play must not duplicate one-time Player rewards.

### 7.4 No numeric economy yet

Do not assign arbitrary point values.

Schema may support numeric awards, but K75 does not define progression balance.

---

## 8. Aftercare

### 8.1 Form

Show three qualitative prompts:

1. **What were your favorite moments from the Show?**
2. **Who surprised you the most?**
3. **What do you hope to see in the next Session?**

Question two:

- selectable participating Player/Character where available;
- optional freeform explanation;
- freeform fallback if no suitable person list exists.

All fields optional.

### 8.2 Draft behavior

Drafts:

- autosave locally or server-side through the project's established safe form-draft pattern;
- survive closing and reopening;
- are not considered submitted;
- do not reset skip count until submitted.

Primary actions:

```text
Save and Close
Skip
```

### 8.3 Save and Close

`Save and Close`:

- accepts partial answers;
- creates or updates one Aftercare submission;
- records submitted time;
- resets consecutive skip count to zero;
- creates a Story So Far event;
- delivers the submission to Director's Chair;
- closes the panel;
- does not block later play.

### 8.4 Skip

`Skip`:

- opens a confirmation panel;
- displays prior consecutive-skip context;
- offers:
  - **Continue Without Aftercare**
  - **Go Back**
- increments the skip count only after confirmation;
- records skipped status;
- creates a minimal Story So Far event;
- closes the panel after confirmation.

### 8.5 Skip counter

Track consecutive explicit skips at the Player level.

Completing Aftercare resets it.

Do not count:

- closing the panel;
- refreshing;
- abandoning a draft;
- reaching tutorial completion without opening Aftercare.

### 8.6 Notes reminder

After Save or Skip, show:

> Did you take notes? Don't forget to update your notes on yourself and your fellow Players.

Actions:

```text
Update My People
Next
```

`Update My People` routes to the existing My People experience.

---

## 9. Director's Chair

### 9.1 Authority

Director+ means:

- Director;
- Producer;
- Operator.

### 9.2 Organization

Organize Aftercare as:

```text
Director's Chair
└── Show Run
    └── Show
        └── Session
            └── Player / Character
```

Provide a spreadsheet-style table with:

- Show Run;
- Show;
- Session;
- Player;
- Character;
- completion timestamp;
- Aftercare state;
- favorite moments;
- surprising person;
- surprise explanation;
- hopes for next Session;
- unresolved door intention;
- Story So Far summary;
- consecutive skip count;
- waiting-for-human-play state.

### 9.3 Read-only

Director's Chair may:

- view;
- filter;
- sort;
- search;
- export.

No:

- replies;
- scores;
- comments;
- Director edits to Player responses;
- threaded discussion.

### 9.4 Export

Provide a CSV export or the project's existing spreadsheet export format.

File organization and naming should be easy to understand, for example:

```text
Victory Exports/
  Show Run Name/
    Show Name/
      Session Date/
        aftercare.csv
```

If filesystem folder export is not appropriate in the web app, preserve equivalent metadata and download naming.

### 9.5 Empty and partial states

Clearly distinguish:

- not opened;
- draft;
- submitted;
- skipped;
- no Session;
- Player disconnected;
- tutorial completed but Aftercare pending.

---

## 10. Waiting for scheduled human play

### 10.1 Waiting message

After completion, the persistent handoff state should say:

> The automated beginning is complete.
>
> Your Character and progress are saved.
>
> The story continues with real people during a scheduled Session.

### 10.2 Explore Catharsis

The Player may close the completion panel and explore Catharsis.

Provide obvious routes to:

- Character;
- Story So Far;
- Inventory;
- My People;
- tutorial completion panel.

### 10.3 Shared Scene advancement

When a Director+ later flies a new shared Scene through the normal Scene/Cue system:

- clear the local tutorial projection;
- close or demote the waiting-state overlay;
- show the new shared Scene;
- preserve Story So Far and Aftercare access.

K75 does not add a special continuation GO.

---

## 11. MVP proof pass

Kernel 75 includes the MVP proof pass.

### 11.1 Golden journey

Walk the complete path with a clean account:

```text
landing page
→ signup
→ onboarding
→ Trailer Face
→ Third Place
→ Character creation
→ Show access
→ Character selection
→ /showtime
→ Catharsis
→ Kessa
→ equipment
→ door
→ intention
→ Ra
→ gate opening
→ tutorial completion
→ Story So Far
→ Aftercare
→ My People reminder
→ waiting for scheduled human play
```

### 11.2 Required roles

Test with:

- Player;
- Director;
- Producer/Operator where authority differs;
- Audience leakage check.

### 11.3 Refresh/reconnect points

Refresh at least after:

- Character creation;
- Show join;
- Kessa completion;
- equipment purchase;
- door intention;
- Ra topic progression;
- gate open;
- tutorial completion;
- Aftercare draft;
- Aftercare submission;
- waiting state.

### 11.4 Desktop presentation

Target the operator's normal desktop browser.

Verify:

- no dead controls;
- no invisible required actions;
- no UUID entry;
- no unexplained permission failures;
- copy fits;
- overlays can close and reopen;
- visual hierarchy is clear enough for a narrated video.

### 11.5 Advertising assets

Before recording, replace or approve:

- Ra portrait;
- tutorial-complete/handoff art;
- Ra prose;
- tutorial-complete copy;
- visible placeholder labels;
- obvious developer/test copy.

The Courtyard map and door art are already acceptable unless the operator changes them.

### 11.6 Clean demonstration state

Prepare:

- clean Show Run;
- clean Show;
- clean Session state;
- controlled test Player;
- no visible throwaway test records;
- no broken duplicate interactions;
- no stale active Sessions blocking `/showtime`.

### 11.7 Commit and deployment discipline

For MVP closure:

```text
tests green
→ operator browser walk
→ fix defects
→ commit exact reviewed state
→ deploy that commit
→ verify live
→ record commit SHA in reportback
```

Do not leave the final advertised deployment uncommitted.

### 11.8 Interest-signup readiness

K75 need not open public self-serve access.

Confirm:

- landing page can collect interest;
- contact/Discord path is clear;
- tours remain controlled;
- no claim that open enrollment is generally available unless explicitly enabled.

---

## 12. Authority and privacy

Prove:

- anonymous users cannot read Story So Far or Aftercare;
- Player can read only their own Character history and submissions;
- Director+ can read Show-scoped Aftercare;
- Audience cannot read Aftercare or private Story So Far;
- private-by-default entries do not leak to Trailer Face;
- Director-authored journal entries preserve author identity;
- Player cannot edit generated Story So Far;
- Player cannot award themself milestones;
- replay cannot duplicate one-time Player recognition;
- skip count cannot be client-forged;
- CSV/export obeys Director+ authority;
- Character switching never shows another Character's tutorial reflection;
- local completion projection remains viewer-scoped.

---

## 13. Tests

### 13.1 Backend tests

At minimum:

- Ra closing sequence gating;
- Continue idempotency;
- tutorial completion milestone;
- archetype-to-attribute selection;
- tied attributes resolved through archetype;
- deterministic summary generation;
- sparse-data summary;
- Story So Far chronological ordering;
- private-by-default behavior;
- reveal-state persistence;
- Director-authored journal entry provenance;
- one-time Player recognition;
- per-Character replay;
- Aftercare partial submission;
- draft persistence;
- Skip confirmation;
- consecutive skip count;
- submission resets skip count;
- closing without Skip does not increment;
- Director's Chair role filtering;
- export role filtering;
- local projection survives refresh;
- later shared Scene clears projection.

### 13.2 Frontend tests

At minimum:

- Ra final narration before Continue;
- completion Program opens only after Continue;
- panel close/reopen;
- Story So Far rendering;
- missing-data omission;
- Aftercare partial Save and Close;
- Skip warning states;
- Go Back behavior;
- My People routing;
- Director's Chair table states;
- CSV export control;
- waiting-state copy.

### 13.3 Manual browser test

Operator must personally verify:

1. Ra finishes unlocking and opening the gate.
2. Continue opens the completion projection.
3. personalized reflection matches the Character's real data.
4. no unsupported claims appear.
5. Story So Far is private.
6. Aftercare partial save works.
7. closing/reopening preserves draft.
8. Skip warning reflects prior skips.
9. Director's Chair receives submission read-only.
10. My People link works.
11. Player may close the completion panel and explore Catharsis.
12. waiting message clearly explains scheduled human continuation.
13. shared Scene advancement clears the local projection.
14. complete fresh-account journey works live.
15. recording environment is clean.

Run the existing alpha gate. Do not create a competing gate.

---

## 14. Explicit exclusions

Do not build:

- AI-written reflection;
- numeric ratings;
- Director replies to Aftercare;
- full Player-level economy;
- arbitrary point values;
- general achievement shop;
- full mobile venue navigation;
- public open-access onboarding;
- special tutorial continuation GO;
- broad renderer rewrite;
- new quest system;
- automatic campaign scheduling;
- new social feed replacing Story So Far;
- Director edits to Player reflections.

---

## 15. PASS standard

Kernel 75 passes only if:

1. Ra completes the gate-opening narration before tutorial completion.
2. Continue opens the participant-local completion projection.
3. Player may close and reopen the completion Program.
4. waiting copy clearly states that scheduled human play comes next.
5. deterministic Character reflection uses real recorded data.
6. archetype guides the primary attribute lens.
7. unsupported claims are omitted.
8. Story So Far exists as a separate private-by-default Character history.
9. meaningful tutorial events populate Story So Far.
10. `/journal` or its canonical storage supports Director-authored Character moments.
11. Character and Player completion milestones persist.
12. first-time Player recognition is not duplicated.
13. another Character may complete the tutorial separately.
14. Aftercare contains the three locked qualitative prompts.
15. partial Save and Close works.
16. Skip requires confirmation and tracks consecutive skips.
17. closing without Skip does not increment.
18. Aftercare is delivered read-only to Director's Chair.
19. Director's Chair supports filtering and export.
20. My People is offered as the default note follow-up.
21. Player remains in a participant-local waiting state.
22. future shared Scene advancement clears the waiting projection.
23. privacy and authority tests pass.
24. complete fresh-account golden journey passes.
25. visible MVP placeholders/copy are approved or replaced.
26. final reviewed deployment is committed and the SHA recorded.
27. operator completes the live browser/video-readiness walk.

---

## 16. Reportback requirements

Include:

- PASS/PARTIAL/FAIL;
- actual migration numbers;
- pre-implementation audit;
- Ra closing sequence;
- tutorial-complete Program behavior;
- deterministic reflection rules and template examples;
- archetype/attribute selection behavior;
- Story So Far model;
- `/journal` integration;
- privacy/reveal model;
- Character and Player milestone model;
- future gamification extension points;
- Aftercare schema and skip logic;
- Director's Chair organization and export;
- waiting-state behavior;
- shared Scene clear behavior;
- authority/security tests;
- automated test output;
- alpha-gate output;
- full manual golden-journey evidence;
- advertising/video-readiness findings;
- final commit SHA;
- live deployment status;
- known limitations;
- files changed;
- recommended post-MVP priorities, without starting them.
