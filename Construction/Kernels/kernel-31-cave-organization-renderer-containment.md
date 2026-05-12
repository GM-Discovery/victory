# Kernel 31 - Cave Organization + Renderer Containment

## Purpose
Clean the proving ground without changing the spine.

## Scope
- Keep the Pixi stage experimental but usable
- Organize Cave tools into clearer panels, overlays, and drawers
- Reduce visual clutter around the stage
- Preserve DOM chat, context menus, and forms
- Preserve server-authoritative actions
- Ensure Pixi only renders projected stage state
- Fix obvious interaction rough spots:
  - nameplates
  - hide/show behavior
  - lock/unlock
  - context menu consistency
  - renderer fallback behavior

## Non-Goals
- No full Pixi migration
- No 3D
- No new venue clone yet
- No template extraction yet
- No rules engine
- No major new features

## Pass Condition
The Cave still contains all the tools, but the stage view becomes readable and the tools feel organized instead of piled on.

## Short Blurb
Renderer decision: Pixi is adopted experimentally, but not yet promoted to final/default stage truth. Kernel 31 will focus on organization, containment, and interaction cleanup so the proving ground stays usable while the spine remains server-authoritative.
