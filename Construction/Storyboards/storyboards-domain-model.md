# Storyboards Domain Model (Kernel 80, extended by Kernel 81)

Implementation: `backend/internal/storyboards/`. Schema:
`backend/migrations/090_kernel80_storyboards_core.sql` +
`091_kernel80_ewrite_object_link_storyboard_card.sql` +
`092_kernel81_storyboard_card_image.sql`.

## Kernel 81 additions (bounded — see the kernel spec's backend change budget)

- `storyboard_cards.image_asset_id UUID`, **no foreign key** — see
  `storyboards-card-image-contract.md` for the full reasoning (a real FK
  would be silently broken by either of two existing asset-lifecycle
  paths, defeating the gravestone requirement). Set/cleared via
  `cards.go`'s `SetCardImage`, same `canMutateCard` (Crew+-unless-locked)
  authority as every other card-content field.
- `cards.go`'s `SwapCards` — a new atomic function (one transaction,
  version-checked on both cards) exchanging two cards' cell placement in
  one step, backing the frontend's occupied-cell "Swap" resolution. See
  `storyboards-drag-and-drop.md`.
- Everything else in this document is unchanged from Kernel 80.

## Hierarchy

```
Storyboard
├── columns (ordered, board-scoped)
├── bands (ordered, board-scoped)
│   └── rows (ordered within their own band)
└── cards (placed at row × column, ordered within their cell)
```

## Two-level row order, not one global order

`storyboard_rows.sort_order_in_band` orders a row only within its own
band; `storyboard_bands.sort_order` orders the bands. Full visual row
order is "bands by `sort_order`, then each band's rows by
`sort_order_in_band`" — computed by `rows.go`'s `ListRows` (a single
`JOIN` + `ORDER BY b.sort_order, r.sort_order_in_band`), never stored as a
flat sequence.

This was a deliberate simplification during implementation over a single
global `sort_order` with an application-enforced contiguity invariant
(the shape an earlier design pass proposed). The two-level scheme makes
every one of the spec's band/row rules true **by construction**, with no
invariant-checking Go code required:

- "every row belongs to exactly one band" — `band_id NOT NULL`.
- "bands never overlap or share rows" — structurally impossible, since a
  row's position is only ever expressed relative to its own band, never
  as a board-wide coordinate a second band could collide with.
- reordering bands never touches row data.
- moving a row to another band is just `band_id` + a `sort_order_in_band`
  appended at the target's end, never a board-wide renumber.

## Ownership vs. grants

`storyboards.owner_user_id` is a plain column, checked separately from
`storyboard_grants` — ownership is **never** a sentinel grant row. The
spec is explicit that "the owner is not merely another grant": owner
holds capabilities (grant/revoke access, choose granted role, archive,
delete) no `granted_role` value expresses, and a sentinel-row design
would make ownership structurally revocable by deleting a grants row.
`owner_user_id` is `ON DELETE RESTRICT` (not `CASCADE`) so a board can
never silently vanish as a side effect of a user delete — the Kernel 77
account-deletion blocker is what clears ownership (see
`storyboards-permissions.md` and `account_deletion.go`).

`storyboard_grants.granted_role` reuses the existing `location_role`
enum (`001_init.sql`: `producer`/`director`/`cast`/`crew`/`audience`)
rather than inventing new role names — a DB-enforced reuse, not just a
convention. There is no `'owner'` value in that enum; owner status lives
only in `storyboards.owner_user_id`.

## `hidden_from_audience` is a plain boolean, not the JSONB visibility shape

`scene_stage_elements.visibility` uses `{"toRoles":[...],"privateTo":[...]}`
to express per-role visibility across a 5-role stage. Kernel 80's rule is
strictly binary (audience+cast vs. crew+), so `storyboard_cards.hidden_
from_audience BOOLEAN` is the more honest match for what the spec actually
asks for — a deliberate, explained deviation from the established
convention, not an oversight. See `storyboards-permissions.md` for who
may set it and `storyboards-live-events.md` for how it's kept from
leaking through live events.

## 200-column limit

Enforced in Go (`columns.go`'s `AddColumn`: counts existing columns,
rejects with `ErrColumnLimitExceeded` before insert at 200), not a DB
constraint — framed as a soft implementation safeguard with a specific
client-facing error (spec 1.4), not a hard schema rule.

## Card `version` — optimistic concurrency, not full revision history

`storyboard_cards.version INTEGER` bumps on every `UpdateCard`/`MoveCard`/
`ReorderCardsInCell`/`SetCardLock`. Mutations that take a `baseVersion`
(`UpdateCard`, `MoveCard`, `DeleteCard`) run `... AND version = $N`; zero
rows affected returns `ErrCardVersionConflict` (HTTP 409). This is
simpler than eWrite's revision-chain table since the spec only requires
"no silent overwrite," not stored history (spec 4.7). Because every
mutation is a single-row `UPDATE`/`DELETE` — never an insert-then-delete
— a card can never be duplicated by a race either; this is proven by
`TestConcurrentMoveCardNeverDuplicates`.

## Cross-board integrity

A card's `row_id` and `column_id` must belong to the same
`storyboard_id`. Enforced in Go (every card-mutating function loads the
row and column via `loadRow`/`loadColumn`, both scoped by `storyboard_id`
in their `WHERE` clause, so a cross-board ID simply 404s as
`ErrRowNotFound`/`ErrColumnNotFound` rather than silently succeeding) —
no cross-table CHECK trigger exists in this schema family; none was found
elsewhere in the repo as a precedent to follow, so this is the
lower-complexity choice.

## Occupied-removal resolution flow (spec 1.4, 1.5, 6.2-6.4)

Removing a column, band, or row that still holds cards (or rows, for a
band) never silently cascades. `RemoveColumn`/`RemoveBand`/`RemoveRow`
each take a `resolution` argument:

- `""` (empty) against an occupied target → `ErrColumnOccupied` /
  `ErrBandOccupied` / `ErrRowOccupied` (caller must choose).
- `"delete_cards"` / `"delete_rows"` → contents removed with the parent.
- `"move_cards"` / `"move_rows"` (+ a `target_*_id`) → contents relocated,
  appended to the end of their new home's order, before the parent is
  removed.

An unoccupied target deletes outright regardless of `resolution`.
