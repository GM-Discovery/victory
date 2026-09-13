# Stage Object Visibility — Operator Guide

Kernel 90. Who can currently perceive or interact with a thing on your stage,
how you change that, and what Victory deliberately will not do.

The shape of the whole thing in one line:

> **A Scene describes the stage. Visibility describes who can currently
> perceive what is on that stage. Visibility never creates a second Scene.**

Before Kernel 90 this question had three unrelated answers that could not agree
with each other. It now has one.

---

## 1. The four words that matter

Four distinctions do most of the work. Getting them straight is most of
learning the feature.

**Hidden is not deleted.**
A hidden object still exists. It keeps its name, its position, its artwork and
its identity. Reveal it and it comes back exactly as it was. Deleting is still
a separate operation ("Remove from Stage") and still permanent.

**Disabled is not hidden.**
A disabled interaction stays *visible*. Players can see the weapon cabinet;
they just cannot open it. That is usually what you want — "the cabinet is
locked" should read as a locked cabinet, not as a cabinet that vanished.

**Hidden is not disabled.**
Hiding an object does not switch its interaction off, and revealing it does not
switch one back on. Independent on purpose, so you can hide something that is
still armed for when it reappears.

**Visible is not "visible to the people I meant".**
An object is either visible to all ordinary viewers, or hidden — and a hidden
object can be *granted* to specific audiences. Grants do nothing at all to a
visible object. If you set a scope and nothing changed, this is almost always
why: you set the scope but never hid the object.

---

## 2. Where the controls are

**Right-click any durable stage object** — a token, an index card, a drawing, a
Scene-authored prop. Two grouped entries appear:

```
Visibility
  Visible ✓
  Hidden
  Scope…
Interaction
  Enabled ✓
  Disabled
```

The tick marks the current state, so you read it rather than infer it from
which verb is on offer.

**Interaction appears only when there is something to interact with.** A plain
prop shows Visibility alone. A token bound to a merchant, a door or a dialogue
shows both.

Nothing was added to the toolbar. Kernel 89 already spent the toolbar budget,
and these controls belong on the object they affect.

> **Keyboard access is a known gap.** The stage context menu is right-click
> only, inherited from Kernel 83. The scope panel is keyboard-navigable and
> closes on Escape once open; the menu that opens it is not. See §8.

---

## 3. The audiences you can scope to

Choose **Scope…** on a hidden object. You are picking who may perceive it
*while it is hidden*.

| Scope | Who that is |
|---|---|
| *(nothing selected)* | **Director-only.** Backstage sees it, nobody else. |
| **Cast** | Every Player in the Show — but **not** the house. |
| **Audience** | The house — but **not** the Cast. |
| **Cohort: …** | One Cohort. Cohort A has reached the boat; Cohort B has not. |
| **Character: …** | One specific Character — only the Player currently *playing* it, not merely someone who owns it. |

**You can pick several at once.** "Cohort A and the Audience" is one object with
two grants. Not two objects, and not two Scenes.

Cast and Audience are separate deliberately. It is genuinely useful to show the
players a map pin the house cannot see, and equally useful to show the house a
looming shape the players have not noticed. Neither implies the other.

**Save & hide** does both steps in one press, because setting a scope on an
object you forgot to hide is the easiest mistake to make here.

---

## 4. What you see as Director

Hidden objects **stay on your working stage**. They are dimmed and carry a
dashed outline; objects showing a nameplate also read `HIDDEN`.

That is the whole treatment, and it is restrained on purpose. Your stage should
still look like your stage, not like a debugging session.

You also see each object's grant list. Players never do — telling Cohort B that
something is revealed to Cohort A leaks the shape of the secret even while
correctly withholding the thing itself.

---

## 5. Cues do exactly what you do

Four Cue actions are now real:

```
reveal_object       hide_object
enable_interaction  disable_interaction
```

They call the **same** operation your right-click menu calls, on the same
record. So:

- A Cue hiding object X and you hiding object X produce identical state.
- After a Cue fires, your menu shows the new state as current.
- There is no separate "Cue visibility" to drift out of step.

A Cue targets an object by its identity, never by its label or its position on
screen. Rename a token, move it, re-lay-out the stage — the Cue still finds it.

**A Cue does not change scope.** It reveals to whoever you already scoped the
object for. That keeps "who is this for" in one place — your deliberate choice —
rather than scattered across every Cue that touches the object.

---

## 6. What stays outside this system

**The map.** The map keeps its own existing behaviour. There is no per-Player or
per-Cohort map hiding, and there will not be: when you want to change the stage
radically, **change Scene**. That is the theatrical operation, and it is better
than hiding a map.

**Scenes.** Nothing here is written into a Scene. Staging the same Scene into a
*different* Show starts clean, with nothing hidden — so a reusable Scene never
carries another Show's secrets. Within one Show, what you hid stays hidden.

**Persistent Character state.** Fate, pools, statuses, inventory and Story So Far
are keyed to the Character, never to a Scene or to visibility. Changing Scene
does not touch them.

**Announcements and dice.** Ephemeral presentation, not durable objects. Not part
of this system.

**No fog-of-war.** No line of sight, vision cones, lighting, or explored areas.
This is per-object visibility, which is a different thing.

**No rules engine.** Enable and disable are *states you set*. Nothing watches for
a condition and flips them for you.

---

## 7. State survives

Visibility and interaction state persist across page reload, WebSocket
reconnect, Session resume, and you switching views. It is stored against the
**Show**, so restarting a Session does not silently reset what you hid.

---

## 8. Known limits, stated plainly

**No backfill from the old system.** Anything hidden before Kernel 90 reads as
visible now. A deliberate call: the old mechanism never recorded *who* something
was hidden from, so there was nothing honest to convert. Re-hide the handful of
things that matter through the new controls.

**Pre-Show-era sessions lose reveal/hide.** A legacy session with no Show
attached has no Show to store state against, so the old Cave-style hide does
nothing there. Sessions attached to a Show are unaffected.

**Right-click only.** No keyboard path to the stage context menu yet.

**The Audience cannot reach Catharsis at all.** Venue access requires a cast,
crew, director or producer membership, so a pure audience-role account cannot
open the venue. Audience *scoping* works and is proven; whether the Audience
should be able to walk into Catharsis is a separate product decision. Same shape
as Kernel 89's First Theater finding.

**Roster Players are labelled "audience" by the venue role lookup.** A
pre-existing gap upstream of this kernel: that resolver is called without a Show
Run hint, so it never consults the roster. Kernel 90 is correct regardless —
Cast is resolved from Show participation, not from that label — but anything else
trusting the label is getting a wrong answer.

---

## 9. Walkthrough

Two accounts, one Director and one Player, both in the same Show.

1. Open Catharsis as Director. Confirm the Player sees the same objects.
2. Right-click a token → **Visibility → Hidden**.
3. The token dims and gains a dashed outline on your stage, and disappears
   entirely from the Player's.
4. Right-click again → **Visibility → Visible**. The Player has it back.
5. Right-click → **Visibility → Scope…**, tick **Cohort A**, press
   **Save & hide**. A Player in Cohort A keeps it; one in Cohort B loses it.
6. On a token with a bound interaction, choose **Interaction → Disabled**. The
   Player still sees the object and can no longer use it.
7. **Interaction → Enabled**. They can use it again.
8. Reload both browsers. Everything is exactly as you left it.
9. Author a Cue with `reveal_object` on an object you have hidden and press GO.
   The Player gains it, and your own menu now shows **Visible ✓**.
