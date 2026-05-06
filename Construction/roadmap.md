# Victory Roadmap

## Purpose
This roadmap reflects current priorities after Kernel 25.

Kernel numbers are labels, but from here they should remain stable once assigned so planning, reportbacks, and future agents all refer to the same landmarks.

## Near Roadmap: Kernels 26–35
- `26 — Director Console v1`
  - first explicit authority surface for live Cave management
- `27 — Character / Persona Polish`
  - tighten character flow after Kernel 24 behavior is stable
- `28 — Showing Review v1`
  - review chat, actions, reactions, and session/showing history
- `29 — Reaction Counts v1`
  - lightweight aggregate readback for audience/performer response
- `30 — Cave UI Organization Pass`
  - move proving-ground tools into cleaner drawers, overlays, and menus
- `31 — Template Venue Extraction v1`
  - extract a cleaner reusable venue shell from The Cave after organization work
- `32 — Stage Navigation / View Resolver v1`
  - better view/surface routing inside venue runtime
- `33 — Map/Object Movement v1`
  - more formal object movement and navigation behavior
- `34 — Venue Policy Console v1`
  - clearer venue-level policy toggles and controls
- `35 — Asset Browser / Library v1`
  - browse and select reusable assets/library elements

Ordering note:
- Showing Review comes before polish-only work
- Cave polish and organization belongs around Kernel 30
- template extraction follows that organization pass, not before it

## Medium Roadmap: Kernels 36–60
- strengthen showing/review surfaces and auditability
- richer director controls and venue policy tools
- asset pipeline improvements beyond first browser/library pass
- better movement, layout, and stage-state authoring tools
- cleaner audience segmentation and view resolution
- character-to-playable-sheet bridge once references stop being enough
- template venue hardening and first descendant venue builds
- production management and run coordination surfaces

## Long Roadmap: 60+
- multiple polished descendant venues built from the Cave-derived template
- deeper production tooling and institutional workflows
- more formal review, rehearsal, and run-state lifecycle support
- advanced asset systems and reusable venue packages
- optional future recording/capture work only after Showing Review is mature

## Strategy Notes
- The Cave remains the full-feature proving-ground venue.
- New runtime tools may appear visibly in The Cave first.
- Stable tools should later disappear into cleaner UI structures.
- Future venues should inherit from a cleaned template, not fork the current proving-ground clutter.

## Explicit Non-Goals Right Now
- no near-term video recording work
- no analytics implementation in this kernel range by default
- no premature template extraction before the Cave organization pass

## Recording Language
- **Showing Review** is near-term and means action/chat/reaction/log review.
- **Video Recording** is future-facing and not a current priority.
