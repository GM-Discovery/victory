# Kernel 30.2 Audit Report

## Purpose
This report consolidates scraps, drift, and unfinished work so the next kernel-maker can work from a clean board.

Kernel 30.2 is not a new architecture pass. It is a consolidation pass:
- verify what exists
- fix only what is small and obvious
- document what is partial or deferred

## Executive Summary
The system is in a much healthier state than the older notes suggest.

The biggest live truths are:
- Server truth is authoritative.
- The Cave is still the proving ground and can remain cluttered for now.
- PixiJS is live in First Theater, not in The Cave.
- The Cave uses server snapshot/actions plus projected client rendering, not client-authored truth.
- Showing review, director console control, overlays, lock state, nameplate toggles, persona equip/unequip, and index card lifecycle all exist in real code.

What remains unfinished is mostly consolidation, documentation, and a few intentionally deferred surfaces.

## 1. Pixi / Renderer Containment
### Confirmed
- PixiJS is installed in First Theater via CDN.
- The renderer is client-side only.
- First Theater can fall back to DOM-only readability if Pixi fails.
- The Pixi scene is built from snapshot data, including live cards and the `first-fire` object.
- DOM overlays remain separate from the Pixi canvas.

### Status
- COMPLETE

### Notes
- Pixi is still experimental unless a later kernel proves otherwise.
- The current architecture is:
  - Server truth -> snapshot/actions -> projected elements -> renderer/DOM/Pixi

### Deferred
- a full renderer migration
- renderer authority on the client
- any architecture where Pixi state becomes app truth

## 2. Context Menus
### Confirmed
- The Cave resolves context menus by `context_class` where possible.
- Known classes include:
  - `card`
  - `prop`
  - `scenery`
  - `media`
  - `surface`
  - `actor`
  - `system`
- Audience has no privileged context menu.
- Locked items collapse to unlock/info behavior if authority allows.
- Unknown/future items safely fall back to info-only behavior.

### Status
- COMPLETE

### Notes
- Fire is lightly special-cased by slug fallback, but it does not appear to be a dangerous architectural exception.
- Index cards are not wildly special-cased beyond card behavior.

## 3. Element Action Matrix
### Confirmed
| Action | Backend handler | Server authority | Frontend trigger | Persists | Snapshot reflects | Audience projection | Status |
|---|---|---|---|---|---|---|---|
| `act/reveal_element` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/hide_element` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/place_element` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/remove_element` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/set_element_lock` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/set_nameplate_visibility` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `act/show_overlay` | yes | yes | yes | yes | yes | yes | COMPLETE, single-active only |
| `act/hide_overlay` | yes | yes | yes | yes | yes | yes | COMPLETE, single-active only |
| `create/index_card` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `update/index_card` | yes | yes | yes | yes | yes | yes | COMPLETE |
| `delete/index_card` | yes | yes | yes | yes | yes | yes | COMPLETE |

### Notes
- The action stream is durable.
- The UI is a projection of the action stream, not the source of truth.

## 4. Lock / Unlock
### Confirmed
- Lock state exists and is enforced server-side.
- Locked elements cannot be moved or removed.
- Locked elements can still be inspected.
- Authorized users can unlock from the context menu.
- Audience has no lock controls.

### Status
- COMPLETE

## 5. Nameplate Visibility
### Confirmed
- `act/set_nameplate_visibility` exists.
- The state is element-level visibility data, not a separate persona nameplate model.
- The UI can hide/show nameplates where permitted.

### Status
- COMPLETE for element nameplates
- DEFERRED for a formal persona nameplate system

## 6. Overlay Layer
### Confirmed
- `act/show_overlay` and `act/hide_overlay` exist.
- The active overlay is derived from the action stream.
- Only one active overlay is currently represented.
- The overlay survives in snapshot form and can be hidden cleanly.
- Director Console can hide the active overlay.

### Status
- PARTIAL

### Deferred
- multiple overlays
- timed overlays
- overlay editor
- animation
- audience interaction
- world-anchored overlay authoring

## 7. Venue Chat
### Confirmed
- `chat/message` is distinct from `perform/speak`.
- Venue chat is bottom-panel conversation.
- Stage speech stays in the stage speech lane.
- Chat auto-collapses after inactivity.
- Chat respects `chat_enabled`.
- Stage speech respects `talking_enabled`.

### Status
- COMPLETE for the separation
- PARTIAL for long-window polish and interaction details

### Deferred
- Discord/bot/webhook integration
- richer chat history UX if it is not already present elsewhere

## 8. Stage Speech
### Confirmed
- `perform/speak` remains a distinct staged-speech lane.
- Speech is reviewed distinctly from chat.

### Status
- COMPLETE

## 9. Showing Model
### Confirmed
- Showings exist.
- Actions carry showing association.
- Review exists for closed showings.
- Director’s Chair reads action history rather than video playback.
- Current showing can be controlled from the Director Console.
- New showing startup is still deferred.

### Status
- PARTIAL

### Deferred
- live-starting a brand-new showing
- accidental reconnect-based showing creation
- a video recording model

## 10. Director Console
### Confirmed
- current showing is visible
- audience view toggle works
- close showing works
- chat policy toggle works
- overlay status and hide overlay work
- presence roster is visible
- unauthorized users are blocked

### Status
- PARTIAL

### Deferred
- start new showing
- reopen showing
- production/run selection
- richer element tools
- live auto-refresh beyond the current loop

## 11. Character / Persona
### Confirmed
- Greenroom owns character creation/editing.
- The Cave only equips or unequips existing character cards.
- `persona/equip` and `persona/unequip` work.
- Presence updates after persona changes.
- Sheet links are metadata only.

### Status
- COMPLETE for the equip/unequip flow
- DEFERRED for playable sheets and game mechanics

### Deferred
- dice
- stats
- macros
- rules execution
- initiative
- HP
- tokens

## 12. Greenroom / Trailers Split
### Confirmed
- Trailers is the performer profile drafting/publishing surface.
- Greenroom is the public profile + dressing room surface.
- The split is documented and reflected in current UI behavior.

### Status
- COMPLETE

## 13. Map Placeholder / Venue Registry
### Confirmed
- The venue registry contains several surfaces that are not yet full experiences.
- Public visibility and role visibility already exist as live policy logic.
- Some listed venues are frontend shells or redirects only.

### Status
- PARTIAL

### Deferred
- new venue experiences
- policy cleanup that would require broader access redesign

## 14. Workshop / Library / Warehouse
### Confirmed
- Workshop exists, but it routes through Cave workshop mode.
- Library and Warehouse are still placeholder-ish from a frontend perspective.
- Asset management is not yet a full browser experience.

### Status
- PARTIAL / DEFERRED

### Deferred
- Asset Browser / Library v1
- Warehouse storage surface
- Library script/ruleset surface
- promote element to library
- producer asset approval
- storage quotas
- ownership/lending UI

## 15. Index Cards
### Confirmed
- Index cards are elements.
- Create/update/delete exist.
- Placement exists.
- Cards can be moved, revealed, hidden, locked, and nameplate toggled.
- Cards appear in review/history.

### Status
- COMPLETE for core lifecycle

### Deferred
- timeline ordering
- stacks
- Microscope-specific lanes
- stamps
- drawing
- images on cards
- rich text

## 16. Cave Clutter / UI Organization
### Confirmed
- The Cave still contains a lot of proving-ground surface area.
- That is acceptable for now as long as it is documented honestly.
- Later kernels should hide stable tools into overlays, drawers, context menus, or cleaner venue surfaces.

### Status
- PARTIAL

### Required Statement
The Cave is allowed to contain everything while tools are being proven. A later template venue will hide tools into overlays/drawers/context menus. Future venues descend from the cleaned template.

## Documentation Drift Found
The following items were the biggest documentation mismatches against live code:
- PixiJS is live in First Theater, not The Cave.
- Pixi is experimental and renderer-only.
- Vendor acknowledgements did not list PixiJS.
- Security notes still treated anonymous audience join as if it were already removed.
- Director Console still defers start-showing.
- Some venue surfaces exist only as registry entries or shells.
- The Cave remains cluttered on purpose, but older notes sometimes read like that clutter should already be gone.

## Practical Next Steps
- Keep Kernel 30 stable.
- Treat Kernel 30.2 as documentation and consolidation, not invention.
- Update stale notes when they conflict with live code.
- Do only small bounded fixes if they are obvious and testable.
- Leave larger architecture changes for later kernels.

## Validation
The live repo state already passes the basic checks used during this audit:
- `go test ./...`
- `node --check` for edited inline scripts
- `git diff --check`

