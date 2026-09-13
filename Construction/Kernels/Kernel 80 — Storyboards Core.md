# Kernel 80 — Storyboards Core

**Status:** DEPLOYED LIVE 2026-08-06
**Type:** Bounded product-foundation kernel
**Primary tracks:** Victory Core
**Secondary tracks:** Socio, Anthology, Education
**Planning authority:** `Victory_Canonical_Roadmap_v2.md`
**Sequence position:** After Kernel 79
**Product surface:** Storyboards
**Primary Venue:** Storyboard Venue

---

## 0. Kernel contract

Kernel 80 creates Victory's reusable Storyboards foundation.

A Storyboard is a server-authoritative, shareable, grid-structured board composed of:

```text
Storyboard
├── ordered columns
├── ordered rows
├── bands containing adjacent rows
└── cells at row × column intersections
    └── zero or more ordered cards in each cell
```

This kernel must deliver one coherent vertical:

```text
Owner creates Storyboard
→ defines columns, rows, and bands
→ shares access with selected people
→ Crew+ participants create and move cards
→ Director+ changes board structure
→ connected users see authoritative updates
→ hidden cards respect established visibility behavior
→ board exports to structured JSON
```

Kernel 80 is intentionally smaller than Kernel 79.

It must not absorb Microscope rules, merged cells, cards spanning cells, nested bands, nested cards, printable image/PDF export, freeform canvas behavior, universal game-package support, or semantic narrative analysis.

The kernel succeeds when Victory has a stable generic Storyboard primitive that later kernels can extend.

---

# 1. Locked product decisions

## 1.1 Product name

The feature is called **Storyboards**. Use **Storyboard** for one board.

## 1.2 Structural model

Definitions:

- **Column** — one ordered vertical division of the board.
- **Row** — one ordered horizontal division of the board.
- **Cell** — the intersection of one row and one column.
- **Band** — a labeled semantic group containing one or more adjacent rows across all columns.
- **Card** — a content object placed inside one cell.

A band is not a row. A band contains rows. Each row belongs to exactly one band. Bands may not overlap and may not share rows.

## 1.3 Cards per cell

A cell may contain zero, one, or multiple cards. Cards in a cell have a stable explicit order.

Users may add a card to an empty or occupied cell, reorder cards within the same cell, and move a card to another cell. Cards do not span cells in Kernel 80.

## 1.4 Columns

Columns are ordered, user-labeled, addable, renameable, reorderable, and removable only through a safe resolution flow when occupied.

The current technical safety limit is 200 columns. This is an implementation safeguard, not a permanent product maximum.

## 1.5 Rows

Rows are ordered, user-labeled, assigned to exactly one band, addable, renameable, reorderable within or between bands, and removable only through a safe resolution flow when occupied.

Moving a row between bands must preserve card placement by column.

## 1.6 Bands

Bands span all columns, contain one or more adjacent rows, have custom labels and optional descriptions, and may be reordered, collapsed, expanded, locked, or hidden from ordinary display where appropriate.

Crew may edit band labels. Director+ may create, remove, reorder, lock, and otherwise structurally manage bands.

Bands do not nest, overlap, or share rows.

## 1.7 Ownership

A Storyboard belongs to the user who creates it. The creator is the board owner.

The owner has full structural editing, card editing, sharing and permissions management, export authority, and archival/deletion authority subject to lifecycle protections.

A board may later be linked to a Production, Show, Module, or other object, but personal ownership remains the Kernel 80 source of authority.

## 1.8 Sharing

The owner may explicitly grant Storyboard access to other users. Existing identity surfaces such as **My People** may help select a person.

A My People relationship does not automatically grant Storyboard access. Sharing requires a deliberate Storyboard access grant.

## 1.9 Established-role mapping

| Victory role | Effective Storyboard capability |
|---|---|
| Audience | View permitted content |
| Cast / Actor / Player | View permitted content |
| Crew | View; create, edit, move, and reorder cards; edit band labels |
| Director | Full structural editor |
| Producer | Full structural editor |
| Operator | Full structural editor |
| Owner | Full editor plus sharing and permission management |

Do not replace these with unrelated visible roles such as Viewer, Participant, or Editor. The backend may calculate effective capabilities internally.

## 1.10 Venue admission

Not every user is admitted to the Storyboard Venue.

Venue admission remains the first access boundary. Users admitted to the Venue are generally there to participate.

Storyboard-specific sharing remains required for boards that are not broadly visible to all admitted users.

## 1.11 Hidden content

Reuse the established hidden-from-audience behavior where repository semantics fit.

At minimum:

- Audience and Cast-level viewers do not see audience-hidden cards;
- Crew+ users can see cards hidden from Audience/Cast;
- hidden content remains server-filtered;
- card counts, titles, positions, and metadata must not leak through client payloads.

Do not create a second nearly identical secrecy system without repository evidence that the established flag cannot serve this use.

## 1.12 Card content

A Kernel 80 card supports:

- title;
- short front text;
- longer back text;
- safe basic formatting;
- optional category or label metadata;
- optional color metadata;
- optional eWrite link;
- hidden-from-audience state;
- lock state;
- author and timestamps;
- cell position;
- order within cell.

Do not embed the full eWrite publication editor inside cards.

## 1.13 Export

Kernel 80 must provide structured JSON export including board identity and metadata, owner reference appropriate for export, visibility grants, columns, rows, bands, cards, ordering, front/back content, labels, locks, hidden state, eWrite links, format version, and integrity metadata where practical.

Visual, image, PDF, and print export are deferred.

---

# 2. Kernel goals

Kernel 80 has seven goals:

1. Create Storyboard ownership and sharing.
2. Create the structural board model.
3. Enforce established permissions.
4. Provide server-authoritative live state.
5. Provide a usable Storyboard Venue.
6. Preserve data portability through JSON export.
7. Remain bounded and avoid Kernel 81/Microscope scope.

---

# 3. Required investigation

Before implementation, inspect:

- current Venue architecture and admission logic;
- current role-resolution code;
- My People and user-selection surfaces;
- existing owner/shared-resource models;
- established hidden-from-audience flags and filtering;
- server-authoritative WebSocket patterns;
- card or index-card models already in Victory;
- eWrite object links;
- current export infrastructure;
- account export and deletion behavior;
- current asset/color systems;
- current board-like or grid-like UI code;
- current drag/drop libraries or repository-native patterns;
- migration numbering;
- backup/restore coverage;
- current uncommitted changes from Kernel 79.

Do not ask Grant for repository facts the code can answer.

Record whether existing models can be extended cleanly or whether Storyboards require a dedicated domain package.

---

# 4. Domain model

## 4.1 Storyboards

Required conceptual fields:

```text
id
owner_user_id
title
description
status
visibility
created_at
updated_at
archived_at
version
```

Status at minimum:

```text
active
archived
```

Deletion behavior must be explicit.

## 4.2 Access grants

Required conceptual fields:

```text
storyboard_id
user_id
granted_role
granted_by
created_at
updated_at
```

`granted_role` should use established Victory role names or a clearly documented mapping to them.

The owner is not merely another grant. The owner remains owner until an explicit transfer mechanism exists.

Ownership transfer is not required in Kernel 80 unless deletion lifecycle requires it.

## 4.3 Columns

Required conceptual fields:

```text
id
storyboard_id
label
description_optional
sort_order
created_at
updated_at
```

Columns must have stable IDs so reordering does not change card identity.

## 4.4 Bands

Required conceptual fields:

```text
id
storyboard_id
label
description_optional
sort_order
is_collapsed
is_locked
created_at
updated_at
```

## 4.5 Rows

Required conceptual fields:

```text
id
storyboard_id
band_id
label
description_optional
sort_order_within_band
created_at
updated_at
```

Every row must reference exactly one band.

Database constraints and service-layer validation must prevent unowned rows or cross-board membership.

## 4.6 Cards

Required conceptual fields:

```text
id
storyboard_id
row_id
column_id
sort_order_in_cell
title
front_text
back_text
category_optional
color_token_optional
ewrite_target_optional
hidden_from_audience
is_locked
created_by
updated_by
created_at
updated_at
version
```

A cell does not require a stored database row unless repository architecture benefits from it. The cell is logically `(row_id, column_id)`.

## 4.7 Activity/versioning

Kernel 80 does not require full revision history for every card.

It does require optimistic or authoritative version checks, author/timestamp audit data, no silent overwrite of a locked or concurrently changed card, and structured events for live updates.

---

# 5. Authority model

## 5.1 Access sequence

Every request must resolve:

1. authenticated user;
2. Storyboard Venue admission where applicable;
3. Storyboard ownership or explicit grant;
4. established role;
5. requested capability;
6. card/band/board lock state;
7. hidden-content visibility.

Client-submitted role claims are not authority.

## 5.2 Audience and Cast

Audience, Actor, Player, and other Cast-equivalent roles may list shared boards, open boards, view permitted structure and cards, open permitted card backs, and follow permitted eWrite links.

They may not create, move, edit, or delete cards; rename bands; alter structure; or manage sharing.

## 5.3 Crew

Crew may create, edit, move, reorder, and delete cards; change card labels/categories/colors; set audience-hidden state where existing policy allows; and edit unlocked band labels and descriptions.

Crew may not add/remove/reorder columns, rows, or bands; lock/unlock structure; manage sharing; or delete the board.

## 5.4 Director+

Director, Producer, and Operator may perform Crew actions and add/remove/reorder columns, rows, and bands; move rows between bands; lock/unlock bands and cards; change board metadata; and export where permitted.

Default: only the owner manages Storyboard access grants.

## 5.5 Owner

Owner may perform all editor actions, grant/revoke access, choose the granted established role, archive, export, and delete subject to safe lifecycle behavior.

## 5.6 Lock behavior

### Locked card

- Audience/Cast view normally.
- Crew cannot edit or move it.
- Director+ or owner may unlock it.
- Hidden behavior still applies.

### Locked band

Recommended Kernel 80 behavior:

> A locked band prevents card creation, editing, movement, row movement, and label edits within the band until unlocked.

Director+ or owner may unlock it.

---

# 6. Structural operations

## 6.1 Board creation

Owner provides title, optional description, initial columns, initial band, and at least one row.

A valid board must have at least one column, one band, and one row assigned to a band.

## 6.2 Column operations

Director+ may add, rename, reorder, and remove columns.

Removing an occupied column must require moving cards, explicitly deleting contained cards, or canceling. Do not silently delete cards.

## 6.3 Band operations

Director+ may add, rename, reorder, collapse, expand, lock, unlock, and remove bands.

Crew may rename/edit labels when unlocked.

Removing a band requires resolving all rows and cards through moving, explicit deletion, or canceling.

## 6.4 Row operations

Director+ may add a row to a band, rename, reorder within a band, move to another band, and remove.

Moving a row preserves row identity, cards, card column positions, and within-cell ordering.

Removing an occupied row requires explicit resolution.

## 6.5 Card operations

Crew+ may, subject to locks, create, edit, move, reorder, delete, flip front/back in UI, and follow eWrite links.

Multiple cards in one cell must remain visibly ordered. The system must not rely on DOM order as authority.

---

# 7. Storyboard Venue

## 7.1 Venue entry

The Storyboard Venue should provide:

- boards owned by user;
- boards shared with user;
- recent boards;
- create board;
- export entry points;
- board search/filter;
- archived boards if authorized.

Do not overload the landing page with every board's full grid.

## 7.2 Board view

The primary board view must show board title, ownership/role indicator, columns, band labels, row labels, cells, ordered cards, hidden/locked states where authorized, owner sharing controls, Director+ structural controls, and Crew+ card controls.

## 7.3 Scale and scrolling

The UI must remain usable with many columns, rows, cards, and collapsed bands.

Requirements:

- persistent row/column labels;
- horizontal and vertical scrolling;
- no full-page failure near the 200-column technical limit;
- virtualization or bounded rendering if repository evidence requires it.

Kernel 80 need not make a 200-column board beautiful, but it must not corrupt or crash.

## 7.4 Card interaction

Prefer:

- click to open;
- explicit edit state;
- drag/drop or move controls;
- reorder within cell;
- keyboard-accessible fallback;
- front/back display.

Drag-and-drop must not be the only way to operate the board.

## 7.5 Band display

Bands must be visually distinct from rows and clearly show membership, label, collapsed/locked state, and boundaries.

---

# 8. Live multi-user behavior

## 8.1 Server authority

All mutations must be accepted or rejected by the server.

The client must not be authoritative for role, ownership, ordering, hidden state, locks, card position, or structural state.

## 8.2 Events

Define structured events for:

- board metadata changed;
- grant changed;
- column added/updated/reordered/removed;
- band added/updated/reordered/removed;
- row added/updated/reordered/moved/removed;
- card added/updated/moved/reordered/removed;
- lock changed;
- board archived.

## 8.3 Concurrency

At minimum:

- stale structural mutations fail safely;
- simultaneous card moves cannot duplicate one card;
- within-cell ordering is deterministic;
- lock changes apply immediately;
- unauthorized clients cannot continue mutating after grant revocation.

A full operational-transform system is not required.

## 8.4 Reconnect

On reconnect, the client requests an authoritative snapshot, replaces stale local state, handles unsent edits explicitly where practical, and continues filtering hidden cards.

---

# 9. Hidden-from-audience behavior

Inspect the established implementation before coding.

Kernel 80 must prove:

- Audience/Cast payloads omit hidden cards;
- hidden cards do not leave metadata revealing title or count unnecessarily;
- Crew+ sees hidden cards;
- moving a hidden card does not broadcast content to viewers;
- card deletion does not reveal prior hidden content;
- export includes hidden cards only for authorized exporter;
- account export respects ownership and authority.

If existing semantics distinguish Audience from Cast differently, document the adopted behavior while preserving the locked requirement that both are view-only.

---

# 10. eWrite links

Cards may optionally link to an eWrite Publication, Section, Subsection, or Directory entry.

Opening a link must preserve access control, open the exact target, avoid leaking hidden titles, provide a safe missing/inaccessible state, and preserve a return path to the Storyboard where practical.

Storyboards do not duplicate canonical eWrite content.

---

# 11. JSON export

## 11.1 Export authority

Owner and authorized Director+ may export according to current role policy.

Hidden cards are included only when the exporter is authorized to see them.

## 11.2 Format

Use a versioned format such as:

```json
{
  "format": "victory-storyboard",
  "format_version": 1,
  "exported_at": "...",
  "storyboard": {},
  "bands": [],
  "rows": [],
  "columns": [],
  "cards": [],
  "grants": [],
  "links": [],
  "integrity": {}
}
```

## 11.3 Preservation

Export must preserve stable or portable IDs, ordering, band-row membership, cell placement, card stack order, front/back content, hidden state, locks, labels, eWrite references, and visibility grants without exposing unnecessary personal data.

## 11.4 Security

Do not export passwords, sessions, private My People metadata, hidden profile details, inaccessible eWrite content, or server secrets.

## 11.5 Import

Full import is not required in Kernel 80.

The format must be designed so a later kernel can import without reconstructing meaning from visual coordinates.

---

# 12. Account lifecycle

## 12.1 Account export

Kernel 77 account export must include owned Storyboards, cards authored by the user where appropriate, access grants, and structured board data.

## 12.2 Account deletion

When an owner requests deletion:

- owned active Storyboards appear in the deletion plan;
- the owner may delete boards;
- future ownership transfer may be offered if repository patterns support it;
- shared users do not silently become owners;
- no ownerless boards remain.

When a non-owner shared user deletes their account:

- grants are removed;
- authored cards remain with anonymized authorship where continuity requires;
- no ghost grant remains.

---

# 13. Required migrations

Likely structures:

- storyboards;
- storyboard_grants;
- storyboard_columns;
- storyboard_bands;
- storyboard_rows;
- storyboard_cards;
- optional storyboard activity/version table;
- indexes and constraints.

Requirements:

- next verified migration numbers;
- clean-install safe;
- additive;
- no manual production rows required;
- explicit FK behavior;
- owner deletion behavior documented;
- row belongs to one band;
- card row and column belong to same board;
- stable ordering indexes;
- 200-column service limit enforced safely;
- backup/restore covers all board data.

---

# 14. Required tests

## 14.1 Ownership and sharing

- creator becomes owner;
- My People relationship alone grants nothing;
- owner grants Audience, Cast, Crew, and Director;
- owner revokes access;
- non-owner cannot manage grants;
- revoked live client loses mutation authority.

## 14.2 Role permissions

- Audience view only;
- Cast/Actor/Player view only;
- Crew creates/edits/moves/reorders cards;
- Crew edits unlocked band label;
- Crew cannot change columns, rows, or band structure;
- Director+ edits structure;
- owner edits structure and permissions;
- client role forgery rejected.

## 14.3 Structural integrity

- board requires column/band/row;
- row belongs to exactly one band;
- bands do not share rows;
- column, row, and band ordering are stable;
- moving row preserves cards;
- occupied row/column/band cannot be silently deleted;
- 201st column rejected with clear error.

## 14.4 Card behavior

- zero, one, and multiple cards in a cell;
- reorder cards;
- move card between cells;
- concurrent move does not duplicate;
- locked card rejects Crew mutation;
- locked band rejects contained mutations;
- Director+ unlocks;
- front/back preserved;
- eWrite link resolves safely.

## 14.5 Hidden behavior

- Audience and Cast cannot see hidden cards;
- Crew+ sees hidden cards;
- hidden metadata does not leak;
- hidden move event filtered;
- unauthorized export omits hidden cards;
- authorized export includes them.

## 14.6 Live behavior

- structured WebSocket events;
- snapshot on connect;
- reconnect consistency;
- stale version rejected;
- grant revocation enforced;
- deterministic within-cell order.

## 14.7 Export

- versioned JSON generated;
- hierarchy and ordering preserved;
- hidden authority enforced;
- no secrets;
- integrity valid;
- malformed board cannot export silently.

## 14.8 Lifecycle

- account export includes owned Storyboards;
- owner deletion plan includes boards;
- no ownerless board;
- shared-user deletion removes grant;
- authorship anonymized appropriately.

## 14.9 Regression

- full Go suite;
- frontend tests;
- fresh-install smoke;
- Kernel 79 eWrite links remain;
- Kernel 77A venues remain;
- account recovery remains;
- backup/restore remains;
- `straturli` remains usable;
- no public-signup regression.

---

# 15. Required browser proof

Manually prove:

- admitted user opens Storyboard Venue;
- unadmitted user is denied;
- owner creates board;
- columns, rows, bands, and cells render correctly;
- band visibly groups rows;
- cell supports multiple ordered cards;
- Crew creates and moves cards;
- Crew changes band label;
- Crew cannot alter structure;
- Director changes structure;
- owner grants and revokes access;
- Audience/Cast are view-only;
- hidden card absent for Audience/Cast;
- hidden card visible for Crew+;
- locked band blocks Crew;
- multiple connected users see changes;
- reconnect restores authoritative state;
- eWrite link opens exact target;
- JSON export downloads and validates;
- 200-column safeguard behaves safely;
- keyboard fallback exists;
- scrolling remains usable.

---

# 16. Required artifacts

```text
Construction/Kernels/Kernel 80 — Storyboards Core.md
Construction/Domains/Storyboards/storyboards-domain-model.md
Construction/Domains/Storyboards/storyboards-permissions.md
Construction/Domains/Storyboards/storyboards-live-events.md
Construction/Domains/Storyboards/storyboards-export-format.md
Construction/Domains/Storyboards/storyboards-ui-contract.md
```

Use the standard reportback template.

---

# 17. Required evidence

## Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

## Tests

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 ./...
```

Run all frontend/node tests and fresh-install smoke.

## Static checks

```bash
git diff --check
```

Run `node --check` on changed scripts.

## Multi-user evidence

Record users/effective roles, accepted and denied operations, event sequence, reconnect result, grant revocation result, and hidden-card filtering result without sensitive account data.

## Export evidence

Record file size, board dimensions, band/row/column/card counts, maximum cards in one cell, integrity result, and reconstruction-readiness assessment.

---

# 18. Pass criteria

Kernel 80 passes when:

- Storyboard Venue exists;
- ownership and explicit sharing work;
- established Victory roles control effective capabilities;
- columns, rows, bands, cells, and cards persist;
- each row belongs to exactly one band;
- bands do not overlap or share rows;
- cells support multiple ordered cards;
- Crew manages cards and band labels;
- Director+ manages structure;
- owner manages permissions;
- Audience/Cast remain view-only;
- hidden cards remain hidden from Audience/Cast;
- locks work;
- live state is server-authoritative;
- reconnect works;
- JSON export preserves board meaning;
- account export/deletion integrates;
- full tests and browser proof pass;
- no Kernel 76-79 regression occurs.

---

# 19. Partial and fail rules

## PARTIAL

Use when static boards work but live sync does not, permissions depend on client trust, sharing is incomplete, multiple cards per cell are unstable, export flattens structure, hidden filtering is incomplete, or ownership lifecycle is unresolved.

## FAIL

Use when hidden cards leak, Audience/Cast can mutate, Crew bypasses structural permissions, one row belongs to multiple bands, card movement duplicates or loses cards, occupied structural deletion silently destroys content, revoked users retain authority, export includes inaccessible content, Grant is locked out, or Kernel 79/security protections regress.

---

# 20. Non-goals

Kernel 80 does not:

- implement Microscope;
- define Lens or Palette;
- create game-specific turns;
- merge cells;
- allow cards to span cells;
- create nested cards or bands;
- provide freeform positioning;
- create image/PDF/print export;
- perform narrative analysis;
- import Storyboard JSON;
- build public sharing links;
- create anonymous access;
- replace eWrite.

---

# 21. Kernel-size guardrails

Kernel 79 was materially larger than intended.

Kernel 80 must remain bounded:

- one Storyboards backend domain package;
- one coherent migration family;
- one primary Venue;
- one live-event contract;
- one JSON export format;
- one complete multi-user browser journey.

When implementation reveals an attractive extension, defer it unless required for pass criteria.

Do not pull forward merged regions, card nesting, visual export, Microscope rules, extensive templates, semantic analysis, or public links.

A passing small core is preferable to another four-agent mega-kernel.

---

# 22. Immediate operator outcome

At completion, Grant must be able to answer:

1. Can I create and own a Storyboard?
2. Can I share it deliberately with someone selected through existing people tools?
3. Do Audience and Cast remain view-only?
4. Can Crew create, edit, move, and reorder cards?
5. Can Crew rename bands without restructuring the board?
6. Can Director+ change columns, rows, and bands?
7. Can only the owner manage access?
8. Can one cell contain multiple ordered cards?
9. Does every row belong to one and only one band?
10. Can bands visually and structurally group adjacent rows?
11. Do hidden cards remain hidden from Audience and Cast?
12. Do locks stop unauthorized changes?
13. Do connected users see authoritative updates?
14. Does revoked access stop immediately?
15. Can I export the board as meaningful JSON?
16. Does account deletion avoid ownerless Storyboards?
17. Did the system preserve Kernels 76-79?

Kernel 80 succeeds when Storyboards are a stable reusable Victory primitive, ready for composition and game-specific packages without prematurely building those later layers.

---

# 23. Implementation record (added post-implementation)

Delivered 2026-08-06. Backend: `backend/internal/storyboards/` (types,
authority, boards, grants, columns, bands, rows, cards, snapshot, http,
events, ws, export — 39 passing tests). Migrations 090-091. Frontend:
`frontend/venues/storyboards/` (index.html, board.html, socket.js).
Account lifecycle integrated into `backend/internal/identity/`
(`account_export.go`, `account_deletion.go`).

Two decisions were confirmed with Grant before implementation (neither
contradicts this spec, both resolve locked-open questions):

- **No Production/Show/Module scoping in Kernel 80** — boards are
  personal (creator-owned) only, consistent with §1.7's "personal
  ownership remains the Kernel 80 source of authority."
- **Boards are private-to-owner by default** — no "broadly visible to
  all venue-admitted users" mode, consistent with §1.10's "Storyboard-
  specific sharing remains required for boards that are not broadly
  visible."

One further decision was confirmed during implementation: **archived
boards block account deletion the same as active ones** (not only
"active" ones as §12.2's literal wording might suggest) — see
`storyboards-domain-model.md`'s ownership section for why treating only
"active" boards as blocking would leave a real path to an unresolvable
`RESTRICT` foreign-key violation, since Kernel 80 has no archived-board
reassignment mechanism.

Full design detail, rationale, and test-by-test mapping to this spec's
§14/§15 requirements live in the five sibling documents in
`Construction/Domains/Storyboards/` and in the reportback filed for this kernel.
