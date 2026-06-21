# Victory Roadmap

## Purpose
This roadmap reflects current priorities after Kernel 47.

Kernel numbers are labels, but from here they should remain stable once assigned so planning, reportbacks, and future agents all refer to the same landmarks.

## Near Roadmap: Kernels 26–47
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
- `32 — Discord OAuth Primary Login v1`
  - add Discord OAuth as the primary login path while preserving Victory identity and operator fallback
- `33 — Operator Bootstrap + Canon Capture v1`
  - add a safe operator bootstrap path for producer authority and capture Kernel 32 plus follow-on live-site work into canon
- `34 — Stage Navigation / View Resolver v1`
  - better view/surface routing inside venue runtime
- `35 — Map/Object Movement v1`
  - more formal object movement and navigation behavior
- `36 — Venue Policy Console v1`
  - clearer venue-level policy toggles and controls
- `37 — Asset Browser / Library v1`
  - browse and select reusable assets/library elements
- `38 — Warehouse / Asset Storage Surface v1`
  - durable storage and browsing surface for reusable assets
- `39 — Library / Script Surface v1`
  - browse scripts and reusable reference material by ruleset or sheet type
- `40 — Catharsis Prep Kernel`
  - prep surfaces for the catharsis flow without building the full experience yet
- `41 — Socio/Catharsis v1`
  - first public-facing catharsis experience surface
- `42 — Documentation + Hardening Pass`
  - reconcile docs, tighten errors, and stabilize the kernel trail
- `43 — Discord Audio Left Tray Foundation v1`
  - add shared Discord audio status/control surface without building Victory audio transport
- `44 — Discord Audio Presence / Speaker Feasibility v1`
  - show live Discord voice-channel participants in the tray, prove speaker-indicator feasibility, and document why volume controls are deferred in the current architecture
- `45 — Venue Shell Rebase + Shared Helper Extraction v1`
  - extract reusable venue shell helpers, tighten the map shell, and make First Theater / Middle School Stage cleaner source patterns for future venues
- `46 — Pixi Map Layer + Workshop Map Upload v1`
  - First Theater now uploads/replaces a single active map as a Workshop asset, stores the crop/fit state server-side, and renders the map on its own Pixi layer without replacing the older stage façade
- `47 — Pixi Grid Primitive + Map Alignment v1`
  - First Theater now supports a persistent, server-backed square or hex grid rendered above the active map and below stage elements, with a Configure Grid context-menu surface for live-preview alignment, save/cancel, hide/show, and reset; visual-only, no snapping or pan/zoom yet

Ordering note:
- Showing Review comes before polish-only work
- Director Console lands immediately after review
- Cave polish and organization belongs around Kernel 30
- Middle School Stage follows the organization pass and proves the first clean shell
- Discord OAuth now lands immediately after the shell work so identity work can become the primary login path without replacing Victory authorization
- Operator bootstrap lands immediately after Discord OAuth so producers still come from Victory authority, not the Discord login itself
- Discord audio lands as a shared surface after the hardening pass, while actual audio remains in Discord
- Voice-state presence can follow the user/session without turning Victory into an audio router
- Venue shell extraction follows the audio pass so the reusable shell contract can capture the live patterns already proved in First Theater, The Cave, and Middle School Stage
- The First Theater Pixi pass now follows shell extraction as a dedicated map-layer workflow instead of a backdrop swap, and the older stage composition remains the preferred live presentation around that layer
- The grid primitive follows the map-layer pass directly so alignment tooling has a real map to align against; stage navigation (pan/zoom) is deferred to a later kernel so the grid first ships as a fixed-viewport visual layer

## Medium Roadmap: Kernels 38–62
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
