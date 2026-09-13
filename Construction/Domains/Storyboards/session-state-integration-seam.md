# Session-State Integration Seam (Kernel 82) — wired by Kernel 83

**Status: implemented.** Everything below this line describes the seam as
it existed when Kernel 82 shipped, deliberately unwired. Kernel 83 wired
it: `board.html`'s `buildReferenceCoordinationSlot` now renders a live
Group Leader/Current Turn slot at the top of the Reference Panel body
whenever `snapshot.coordination.active` is true, and is absent entirely
(not disabled) otherwise — exactly the shape this document predicted.
See `Construction/Domains/Venues/collaborative-venue-coordination-contract.md` and
`Construction/Domains/Venues/presence-tray-coordination-actions.md` for the actual
implementation; the rest of this file is kept as the historical record of
the seam's design rationale, which turned out to need no changes once the
real feature existed.

Spec §8's requirement, verbatim: Kernel 82 must not implement Group Leader, Current Turn, participant ordering, or Presence Tray right-click behavior, but must "document a clean integration point so Timeline can later consume them."

## What actually exists today: nothing, on purpose

There is no `group_leader`/`current_turn` table, column, or field anywhere in the Kernel 82 schema. The Reference Panel has no reserved "session state" slot type among its four field types (`short_text`/`long_text`/`list`/`paired_list`) — none of them represent live session/turn state, and none should; they're all plain user-authored content. This is deliberate, not an oversight: a placeholder table or field for a feature that doesn't exist yet would itself be the kind of premature, half-built abstraction this project's own conventions warn against, and it would risk becoming the *wrong* shape once a real platform-wide Group Leader/Current Turn feature is actually designed.

## Where the seam goes when that feature exists

The natural integration point is the Reference Panel's rendering layer, not its data model:

```text
Reference Panel (board.html's renderReferencePanel)
  → checks whether a session-state provider is available
      (however that future kernel exposes availability —
       e.g. a snapshot field, a capability flag, a separate
       lightweight API call)
  → provider available  → render a small slot showing
                           Group Leader / Current Turn info
  → provider unavailable → slot is absent entirely, not a
                           disabled/greyed-out placeholder
```

This mirrors a pattern already used elsewhere in the frontend for optional cross-feature integration (e.g. `window.attachEwriteRuleLink` is called conditionally — `if (window.attachEwriteRuleLink)` — so a card editor never breaks if that script happens not to be loaded). A future Group Leader/Current Turn kernel would similarly expose something Timeline's `board.html` can check for and call *if present*, without Storyboards ever needing to know that feature's internal implementation, and without Storyboards ever being a blocking dependency for building it.

## Why the Reference Panel and not the board grid itself

Group Leader/Current Turn is inherently *session*-scoped context (who's running this sitting, whose turn is it right now), the same conceptual category as the Reference Panel's existing Premise/Beginning/Ending fields (shared setup/context outside the grid) — not something that belongs attached to a specific card, row, or column. Slotting it into the Reference Panel's existing "persistently reachable while viewing the board" UI real estate (spec §5.1) avoids needing a second persistent-panel concept later.

## What this document is not

Not a specification for the future feature itself — no schema, no API shape, no UI mockup for Group Leader/Current Turn is proposed here, because designing that is explicitly out of Kernel 82's scope (spec §20's non-goals list it by name). This document exists solely so a future kernel author knows where Timeline expects to plug in, without needing to re-derive it from scratch or retrofit the Reference Panel's rendering code to make room.
