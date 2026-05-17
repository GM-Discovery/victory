# Victory Roadmap

## Purpose
This roadmap reflects current priorities after Kernel 31.

Kernel numbers are labels, but from here they should remain stable once assigned so planning, reportbacks, and future agents all refer to the same landmarks.

## Near Roadmap: Kernels 26–41
- `27 — Showing Review v1`
  - review chat, stage speech, reactions, reveal/hide, overlays, cards, and persona changes
- `28 — Director Console v1`
  - live control surface for the current showing
- `29 — PixiJS Stage Spike v1`
  - complete proving-ground spike for PixiJS as the stage/worldspace renderer
- `30 — Cave Organization + Renderer Containment`
  - keep Pixi experimental but usable, and organize Cave tools into clearer containers
- `31 — Middle School Stage + Edge Drawer Layout v1`
  - build the first clean copyable stage shell without turning The Cave into the template
- `32 — Template Venue Extraction v1`
  - extract a cleaner reusable venue shell after the stage-shell pass
- `33 — Stage Navigation / View Resolver v1`
  - better view/surface routing inside venue runtime
- `34 — Map/Object Movement v1`
  - more formal object movement and navigation behavior
- `35 — Venue Policy Console v1`
  - clearer venue-level policy toggles and controls
- `36 — Asset Browser / Library v1`
  - browse and select reusable assets/library elements
- `37 — Warehouse / Asset Storage Surface v1`
  - durable storage and browsing surface for reusable assets
- `38 — Library / Script Surface v1`
  - browse scripts and reusable reference material by ruleset or sheet type
- `39 — Catharsis Prep Kernel`
  - prep surfaces for the catharsis flow without building the full experience yet
- `40 — Socio/Catharsis v1`
  - first public-facing catharsis experience surface
- `41 — Documentation + Hardening Pass`
  - reconcile docs, tighten errors, and stabilize the kernel trail

Ordering note:
- Showing Review comes before polish-only work
- Director Console lands immediately after review
- Cave polish and organization belongs around Kernel 30
- Middle School Stage follows the organization pass and proves the first clean shell
- template extraction follows that stage-shell pass, not before it

## Medium Roadmap: Kernels 37–61
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
