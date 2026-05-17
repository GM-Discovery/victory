# Kernel 31 — Middle School Stage + Edge Drawer Layout v1

## Purpose
Create the first clean stage-shell venue pattern without turning The Cave into the final template.

The Cave remains the proving ground. It may stay cluttered.

Kernel 31 creates a new producer-only venue:

> Middle School Stage

This venue demonstrates the future layout grammar:

- stage/content area in the center
- top health/status/target bar
- left edge drawer for network / presence / future tools
- right edge drawer for character / game / handout surfaces
- bottom collapsed chat drawer
- right-click context menu as the primary object-control surface
- all controls hidden until needed

## Core Rule
Do not break working Cave behavior.

Do not migrate The Cave into the new shell.

Do not delete existing Cave panels.

Do not move Pixi into The Cave.

Do not make Middle School Stage into a full production venue yet.

## Product Intent
The Cave is not the copyable shell.

Middle School Stage is the first clean copyable stage shell.

Reasoning:

- The Cave is primitive, proving-ground, and tool-cluttered.
- Middle School Stage is a cleaner ordinary stage metaphor.
- Pixi is not a primitive and should not automatically exist everywhere.
- Tools proven in The Cave should become portable primitives.

## Venue: Middle School Stage
Required:
- slug: `middle-school-stage`
- display name: `Middle School Stage`
- visibility: producer-only for now
- not anonymous
- not general signed-in
- not audience/cast/crew by default

It should have:
- a real frontend page
- clean layout structure
- placeholders/drawers for future content
- access enforcement

Do not:
- build a full new experience
- add Pixi
- turn it into a full theater product
- delete The Cave clutter

## Status
- Stage shell exists as a clean layout prototype
- Access is producer-only
- The map exposes it to producers
- Backend truth still comes from server authority and visibility

