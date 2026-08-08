# Presence Tray Coordination Actions (Kernel 83)

## Where this lives

Storyboards had no Presence Tray at all before this kernel — `backend/internal/storyboards/ws.go`'s own package comment (Kernel 80) says so explicitly: "Storyboards has no location/session/presence concept." Kernel 83 builds the first one, in `frontend/venues/storyboards/board.html` (`#presence-tray`), directly reusing the `network.Hub.BoardWatcherUserIDs` primitive that already existed for a different purpose (per-viewer event fan-out). This document describes the interaction contract so a future venue's tray — reusing this one's markup/CSS pattern or an entirely different tray — implements the same right-click behavior.

## Right-click is the only mutation surface

`.presence-chip` elements carry a `contextmenu` listener (`openPresenceContextMenu`). There is no separate "manage participants" panel, dropdown, or modal — per spec §3.5, that was explicitly out of scope for this kernel.

## Menu construction is server-truth-driven, client-decided

The client decides which of the two possible actions to show using data the server already sent (`snapshot.coordination`, `snapshot.viewer_user_id`, `affordances.canEditStructure`):

```text
Make Group Leader shown iff:
  a coordination session is active, AND
  (actor has Director+ authority OR actor is the current Group Leader), AND
  the right-clicked target is not already Group Leader

Give Turn shown iff:
  a coordination session is active, AND
  (actor has Director+ authority OR actor is Group Leader OR actor holds Current Turn), AND
  the right-clicked target does not already hold Current Turn
```

If neither condition holds, `openPresenceContextMenu` returns before ever showing the menu element (spec §3.1: "Only show actions the acting user is authorized to perform" — not a disabled button, an absent one). This is a UX convenience only. The server (`coordination_http.go`'s `coordinationHandler` → `AssignGroupLeader`/`AssignCurrentTurn`) re-derives authority from scratch on every POST and returns `403 not_authorized` for anything the client shouldn't have been able to send anyway — see `TestCoordinationUnauthorizedTiersCannotSeizeLeader` and the Kernel 83 Playwright proof's scenario 10b/14b, which both drive a forged direct API call from an unauthorized account to prove the server-side gate independent of what the UI shows.

## Wording

Exactly two labels exist, matching spec §3.2 verbatim: **Make Group Leader** and **Give Turn**. No "Pass Leadership To," "Set Leader," "Transfer Leadership," or per-target variants — the right-clicked chip is already the target, so the label never needs to repeat it.

## Indicators

Each `.presence-chip` renders:

- a small green presence dot (this user is currently connected/watching);
- the display name (falls back to handle, then "Someone");
- a `Leader` badge (`.presence-badge-leader`) if `coordination.group_leader_user_id` matches this entry;
- a `Turn` badge (`.presence-badge-turn`) if `coordination.current_turn_user_id` matches this entry.

Both badges can appear on the same chip simultaneously (spec §3.4's "someone holding both" case) — they are independent conditionals, not a mutually-exclusive state machine.

## Ordering is stable, never turn-driven

`renderPresenceTray` renders `snapshot.presence` in the order the server sent it. The server (`BuildPresenceRoster` in `backend/internal/storyboards/coordination.go`) sorts that list by handle, then user ID, as a tiebreak — deliberately not by connection time, leader/turn status, or any other field that could make the tray appear to reorder itself as coordination state changes (spec §3.4: "Do not reorder the Presence Tray based on these states"). Sorting by a fact (handle) that Group Leader/Current Turn assignment can never change is what makes that guarantee hold structurally, not just by convention. The Kernel 83 Playwright proof's scenario 17 captures chip name order before and after a Give Turn action and asserts byte-for-byte equality.

## Absent holders

Presence entries only exist for currently-connected watchers (per this venue's chosen session model — see `venue-session-state-lifecycle.md`), so a disconnected Group Leader/Current Turn holder has no chip to badge at all; the Presence Tray silently omitting them from the roster is itself a truthful "not here right now" signal. Where a stronger explicit "(absent)" note is wanted, it lives in the Reference Panel's read-only coordination slot instead (`buildReferenceCoordinationSlot` in `board.html`), which renders per-holder `*_present` booleans the server computes regardless of whether that holder is currently in the presence roster.

## Accessibility (spec §10.3)

Right-click is the only interaction surface in this kernel, as specified. The menu items are real `<button>` elements inside a normal DOM subtree (not canvas-drawn or otherwise inaccessible to a screen reader once open), and closing via `Escape` or an outside click both work, matching normal menu conventions. There is no keyboard/touch trigger to *open* the menu — this is a known, spec-acknowledged limitation (§10.3: "Log any right-click-only accessibility limitation rather than inflating scope"), in the same category already logged for Kernel 81's occupied-cell picker and Kernel 82's middle-mouse pan in `Construction/Storyboards/storyboards-accessibility.md`. A future kernel adding a keyboard-reachable trigger (e.g. a focusable "⋮" button per chip that opens the same menu) would not need to change anything about the menu's authority logic or wording, only its entry point.
