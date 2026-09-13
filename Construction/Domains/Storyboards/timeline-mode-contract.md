# Timeline Mode Contract (Kernel 82)

What "Timeline" actually is: a built-in **instantiation configuration** for the existing Storyboards primitive, not a second board engine and not a Microscope implementation.

## What ships vs. what doesn't

Timeline reuses every piece of Storyboards unchanged: columns, bands, rows, cards, cell placement, drag-and-drop, color, images, locks, sharing, live sync, export. The only things Kernel 82 adds are:

- a code-defined starting configuration (3 columns with boundary roles, 1 band, 1 row, 4 Reference Panel fields) instantiated once at board-creation time;
- `column_role` on columns (`beginning`/`ordinary`/`ending`) and the server-side reorder/removal protection built on it;
- the Reference Panel itself (new tables/endpoints, generic — not Timeline-specific at the data-model level);
- middle-mouse panning on the board viewport.

Nothing about card rendering, drag-and-drop, hidden-card filtering, WebSocket events, sharing, or account lifecycle changed. A Timeline board and a Blank board are both just rows in `storyboards` — `mode='timeline'` vs `mode='blank'` is the only structural difference the backend or frontend ever branches on.

## Generic interpretation, not enforced meaning

The spec is explicit that Band/Row/Column/Card map to a *generic* interpretation (era/lane/chronological-position/moment) that the product never validates or enforces. Concretely: nothing in the schema, the API, or the UI ever reads a card's, row's, or band's *content* and treats it specially. A "Beginning" column is protected because of its `column_role`, not because of anything written in it — you could title it "Chapter 1" and it stays structurally the left boundary. This is what keeps the implementation honest to spec §1.4/§20's "do not hardcode Period/Event/Scene/Lens/Focus/tone/etc." — those are just words a user might type into an ordinary text field, indistinguishable to the system from any other word.

## No tone system, no placement assistant, no semantic card types (spec 1.6/1.7)

Explicitly not built, not because they were forgotten but because the spec forbids them: cards use the existing `color_token` whitelist unchanged (Kernel 81), with no meaning assigned to any color; card placement is exactly "pick a row × column cell," the same interaction Blank boards already have, with no parent-selection UI layered on top.

## Relationship to Kernel 81's presentation work

Timeline boards render through the exact same `board.html`/`grid-model.js` CSS Grid built in Kernel 81 — no separate Timeline template file, no alternate rendering path. The only rendering additions are conditional on data already in the snapshot (`column_role`, `reference_fields`), not on `mode` directly; a hypothetical future third mode that also wanted boundary columns or a Reference Panel would get both for free without touching Timeline's own code at all.
