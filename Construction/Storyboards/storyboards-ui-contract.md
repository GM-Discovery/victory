# Storyboards UI Contract (Kernel 80)

Implementation: `frontend/venues/storyboards/index.html`, `board.html`,
`socket.js`. No build step (static files, per repo convention) — plain
`<script>` blocks, `fetch(..., {credentials:"include"})` for REST,
`WebSocket` for live events.

## Surface map

- `index.html` — Venue landing: owned/shared tabs, archived-boards
  toggle, create-board form. `GET /api/storyboards` returns
  `{owned, shared}`; "recent" is not a separate query, the client just
  has both lists to work with.
- `board.html?id=<board_id>` — the grid: sticky-implicit column header
  row, band rows (label + Director+ edit/lock/remove controls), row
  cells with ordered cards. Every structural/card control is rendered or
  hidden based on `snapshot.viewer_tier`, which is always the
  server-resolved tier from the last `GET`/WS snapshot — the client never
  computes or trusts its own notion of role.
- `socket.js` — `watchStoryboard(boardId, {onSnapshot, onEvent, onError})`,
  modeled on `frontend/lib/player-profile-ws.js`. On any
  `storyboard/*` event, `board.html` simply re-fetches the full snapshot
  (`loadSnapshot()`) rather than patching state incrementally — correct
  by construction (always consistent with server truth) and adequate for
  a bounded kernel; a later kernel could optimize this into real
  incremental patching if board sizes make full-refetch too slow.

## Role gating in the UI (not just the API)

`tierAtLeastCrew`/`tierAtLeastDirector`/`tierIsOwner` in `board.html`
mirror the backend's tier ladder and gate: the "+ card" control (Crew+),
the structure toolbar and column/band/row rename/remove controls
(Director+), the hidden-from-audience and lock checkboxes in the card
editor (Director+ only — Crew never sees these fields at all, not merely
disabled ones, per spec 1.9), and the Sharing/Export/Archive buttons
(owner / Director+ as applicable). This is UX only — every one of these
actions is independently re-checked server-side; hiding a control here
is not a security boundary.

## Non-drag-and-drop card move (required fallback, spec 7.4)

The card editor's "Move to" field is a row `<select>` + column `<select>`
+ "Move" button — fully keyboard-operable (Tab, arrow keys to change the
selection, Enter to submit) with no pointer required. This was built and
verified first; no pointer-drag interaction exists in Kernel 80 at all —
adding one was treated as optional scope the guardrail would flag as
creep, not as something the fallback needs to coexist with.

## Occupied-removal resolution flow in the UI

Column/band/row "remove" buttons first attempt a plain `DELETE`. If the
server refuses with `column_occupied`/`band_occupied`/`row_occupied`, the
client prompts for a resolution: type `delete` to remove contents with
the parent, or the exact title/label of another sibling to move contents
there, or leave blank to cancel. This is deliberately a native `prompt()`
dialog rather than a custom modal — functional and fully keyboard
operable, kept simple for a first implementation; verified live against
a real board in `Removal Proof Board` (browser proof, see kernel
reportback).

## eWrite link mount point

The card editor calls `window.attachEwriteRuleLink(container, {objectType:
"storyboard_card", objectId: card.id})` verbatim — no new link UI was
built. Requires `frontend/lib/ewrite-rule-link.js` to be loaded, and the
backend's `idFieldFor` mapping (`ewrite-rule-link.js`) to include a
`"storyboard_card"` case, which Kernel 80 added alongside the object-type
plumbing in `ewrite/links.go`.

## My People is a suggestion source, never a picker that grants anything

The sharing panel's "People you know" list renders My People relationship
`stage_name`s as plain, non-interactive labels — **not** clickable
autofill chips. The My People API (`/api/player-relationships`) keys
relationships by `subject_profile_id`/`stage_name` (a player-profile
persona), not the Victory account `handle` that `storyboard_grants` is
keyed on; there is no safe field here to autofill a handle from without
risking filling in the wrong account. Sharing always requires the
deliberate handle-entry step, matching spec 1.8's requirement that a My
People relationship never itself grants access.

## Layout gotcha: floating nav widgets collide with page-edge content

`venue-account-badge.js` and `back-to-map.js` both float
`position: fixed` in a page corner (top-right / top-left respectively)
whenever the page has no `<header>` element for them to dock into
inline — neither Storyboards page uses a `<header>` tag. `board.html`
reserves `padding-right: 235px` on `.page-header` to clear the account
badge (it doesn't load `back-to-map.js` at all — its own "← Boards" link
in `header-actions` already covers that navigation, more usefully than a
generic map link would from inside a board); `index.html` reserves
`padding: 0 210px` to clear both. Found and fixed during the real-browser
proof (screenshots showed the page title and the "Sharing"/"Export"/
"Archive" buttons rendering underneath these widgets, intercepting
clicks) — worth remembering for any future Victory page that similarly
skips the `<header>` tag.
