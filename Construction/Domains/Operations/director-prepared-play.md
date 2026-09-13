# Director Prepared Play — Operator Guide

Kernel 89. What a Director can prepare ahead of a Show, what they can reach
during one, and — just as importantly — what Victory deliberately will not do
for them.

The shape of the whole thing in one line:

> **Prepared where useful, open where play should remain open.**

---

## 1. Where preparation is authored

**Director's Chair → Preparations.**

Pick a Show Run and a Show, then save:

- **Target complexities** — a label and a number. "Climb Training Wall, 14."
- **Announcements** — a preset style plus your own words, kept so a recurring
  beat is one click instead of retyping.

Both are Director-only. There is no Player-facing route that can read them,
so a saved target complexity is not a spoiler waiting to leak; it is not
visible to a Player at all, at any tier, ever.

You do not have to author here. Every one of these can also be created,
edited, and deleted from inside the live venue — the Chair exists for the
half hour before the room fills up, not as a required stop.

---

## 2. Where prepared material is used

**In the live venue (Catharsis).**

> Announcement support is also wired into the First Theater, but that venue
> currently has no participant-facing visibility — no ordinary Director or
> Player can reach it. If that changes, announcements there will work with no
> further code.

Two buttons sit in the upper right:

| Button | What it is |
|---|---|
| **Director Tools ▾** | One selector holding every tool family |
| **Send Aftercare** | Top-level on purpose — see §6 |

The families behind the selector:

| Family | What it opens |
|---|---|
| Announce… | The preset palette, an open-text field, and your saved announcements |
| Roll Prep… | Prepared target complexities: recall, change live, add, delete |
| Merchant… | Author a merchant from the real equipment catalog, then expose them |
| State… | Kernel 85/88's Game Status card — Fate, statuses, pools, blank flags |
| Stage… | Scene Configuration — activate, update, save-as-new |
| Cohorts… | Who plays together |
| Turn & Help… | Current Turn and the Help/Interrupt stack |

The same families are on the **stage right-click menu** and on a **token's
right-click menu**, nested under "Director Tools ▸" so they cost one line of
menu rather than six. They open the same panels; there is no second copy of
anything.

---

## 3. Recalling a target complexity

Director Tools → Roll Prep… → **Recall**.

The number appears in a chip next to the toolbar, and it prefills the target
complexity field the next time you open an Interrupt in the Help Stack.

What recall does **not** do, on purpose:

- it does not roll;
- it does not choose which Character skill applies;
- it does not decide whether the fiction permits the attempt;
- it does not stop a Player attempting something you never prepared.

You can type over the number in the panel and it saves. That is the whole of
"change the value live."

---

## 4. Exposing a merchant

Director Tools → **Merchant…**

1. **+ New**, or pick an existing merchant.
2. Edit the name and the opening line.
3. Tick stock from the catalog. This is the one canonical equipment corpus —
   the same 145-item Socio catalog everything else uses. There is no second
   list and no way to invent an item here; if something is missing, add it to
   the catalog itself (`/api/venues/{venue_slug}/equipment`).
4. **Save merchant.**
5. Under *Expose on the current Scene*, choose **Everyone on this Scene** or a
   single **Cohort**, then **Expose**.

That creates a stage entry point on the Show's current Scene. Cohort targeting
is enforced on the server at the one Player-eligibility gate, so a Player who
guesses the URL still cannot open a merchant aimed at another Cohort.

Talking to the merchant is roleplay. The panel handles stock and the Haggle
check; everything the merchant *says* beyond the authored opening line is you.

---

## 5. Announcements

Director Tools → **Announce…**

- **Choose who sees it**: this Cohort (or the room), everyone in the Show, or
  Directors only.
- **Press a preset** — Success, Triumph, Explosion, Consequences, Failure,
  Danger, Warning, Revelation, Discovery — and it goes out with its own words
  and its own theatrical treatment.
- Or **write your own** and pick the style it wears.
- **Save as preset** keeps it for next time.

Each style is distinguished by more than colour: a glyph, a frame shape, an
entrance, and its own name printed on the banner. A viewer who cannot
distinguish the accent still reads "EXPLOSION."

**Victory never decides what a roll meant.** There is no rule anywhere that
turns `roll >= target` into SUCCESS. You saw the table, you saw the fiction,
you press the button. That is not an omission to be fixed later; it is the
design.

---

## 6. Sending Aftercare

**Send Aftercare** is its own button, not buried in the selector, because
finding it should never take thought at the end of a Show.

It opens **each Player's own** Aftercare form — the same Kernel 75 form, the
same three prompts, the same drafts. It does not answer anything for them and
it does not create a record on their behalf.

You get a receipt naming who it reached. A Player who is not connected right
now is reported as offline rather than silently counted: their Aftercare is
still waiting for them, because it belongs to the Show, not to the Session.

Who it reaches: Players on the roster with a selected Character, optionally
narrowed to one Cohort. Directors, Crew, and Audience are never targeted —
they have no Aftercare record to write.

**Nothing sends it but you.** `/showtime end` closes the technical Session and
does not touch Aftercare, and it will not start doing so. Whether a
performance is ready for reflection is a human judgement.

---

## 7. Scenes versus persistent state

A **Scene** is stage composition: the map, the fit, the camera, the grid, and
which tokens are where. Save it, recall it, switch to the next one.

A Scene is **not** where Fate, health, statuses, Story So Far, inventory, or
anything else about a Character lives. Those persist independently and survive
a Scene change untouched. If you ever find persistent Player state riding
along inside a Scene save, that is a bug worth reporting, not a feature.

---

## 8. Running the Training Arena

The sequence Kernel 89 was built to make runnable without a coding agent:

1. `/showtime <code>` and `/mic hot`.
2. Players arrive at the Training Arena Scene.
3. **Russel is you.** His canonical self — Russel the Iron-Handed,
   Master-at-Arms, late 40s, "a good man trying to do a nearly impossible
   job" — lives in the **Niava Setting Supplement, Chapter V**, readable in
   the Library while you play. Victory deliberately holds no second Russel:
   there is no NPC record to drift out of agreement with the book.
4. Expose the Arena merchant to whichever Cohort reaches them.
5. Recall prepared target complexities as training attempts come up.
6. Players roll, help each other, spend Fate — all Kernel 88, unchanged.
7. Announce what happened.
8. Apply any state change by hand from the State family.
9. Russel points them at Hervia. That is the main quest: a name, a direction,
   and your delivery. There is no quest log, no objective tracker, and no
   completion trigger, because a campaign is not a checklist.
10. Switch Scenes. Character state comes along.
11. **Send Aftercare.**
12. `/showtime end`.

---

## 9. What is intentionally not automated

Listed plainly, because the absence is the design and someone will eventually
mistake it for a gap:

- **No macros.** One preparation is one understandable thing. Nothing chains,
  and a saved target complexity cannot also award Fate, move a Cohort, or
  change a Scene. Those are separate decisions you make.
- **No dialogue trees.** Merchants have five fixed stances and one Haggle
  slot. Conversation is roleplay.
- **No quest engine.** No objectives, no markers, no completion triggers.
- **No inferred outcomes.** Victory does not read dice and decide meaning.
- **No economy simulator.** No currency was invented.
- **No prepared list of legal actions.** A Player may attempt something you
  never authored, and Victory will never answer "that action is unavailable"
  because it is missing from a preset.

---

## 10. Proving it still works

```bash
# Backend + acceptance proof (test database only; refuses anything else)
bash scripts/smoke/kernel89-run.sh

# The First Theater announcement harness. Currently reports a PRECONDITION
# blocker rather than passing -- that venue has no participant visibility.
bash scripts/smoke/kernel89-run.sh --first-theater

# UI proof: grouped tools, nested menus, announcement rendering
OUT_DIR=Construction/OperatorLogs/evidence/kernel-89 \
  node scripts/smoke/kernel89-director-tools-browser.js

# Unit tests
node --test tests/stage-runtime/kernel89-director-tools.test.js
cd backend && go test ./internal/directorprep/... ./internal/announcements/... ./internal/merchant/...
```

`scripts/smoke/kernel89-run.sh` runs on the real `catharsis` venue row (the
`/ws/*` routes only exist for named venues), closes any leftover session
first, and closes its own when it finishes. Do not run it at the same time as
`go test ./internal/shows/...`.
