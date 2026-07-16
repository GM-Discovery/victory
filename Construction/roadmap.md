# Victory Roadmap

## Purpose

This is the current forward-looking roadmap after **Kernel 70**. The chronological record of shipped kernels lives in `Construction/OperatorLogs/operator-log.md`; older kernel specs and the large track roadmaps remain design history, not the source for the next unused kernel number.

## Shipped Foundation

- Kernels 1–52: identity, authority, actions, presence, venues, Showing review, Director Console, stage/Pixi proving ground, Discord, maps/grids/camera/cards/tokens, warehouse, and canonical dice.
- Kernels 53–60: Character Workbook and Socio progression, Face projection/overrides, commands, skills, and game-event foundation.
- Kernels 61–65: Trailer Player Workbook, My People, DB-test safety, and Third Place.
- Kernels 66–69: Show Runs, Shows, production onboarding, Scene Library, and Show Scene Placements.
- Kernel 70: location-scoped reusable Scenes, persistent Show-owned stage state, current Scene, Show variables, Cues, idempotent GO, rehearsal metadata, and curated player stage buttons.

## Recommended Near Horizon

### Visual Scene Composition And Capture

- Define one canonical mapping between a Base Scene and the existing stage-object system.
- Preserve a clear Base Scene versus This Show's Version override boundary.
- Capture/replay composition without copying state into a second element model.
- Keep Session as runtime window and Show as durable owner.
- Prove distinct Show overrides and persistence across Session replacement.

### Scene-Aware Cue Expansion

- Add `reveal_object`, `hide_object`, `enable_interaction`, and `disable_interaction` only after the visual object identity model exists.
- Keep action execution ordered, idempotent, auditable, and fail-stop.
- Preserve Audience exclusion and Crew's non-destructive boundary.

### Kernel 70 Evidence And Frontend Reliability

- **Closed by Kernel 70A**: shared contracts extracted from the parallel First Theater/Catharsis runtimes — `tests/catharsis/` mirrors `tests/first-theater/` (import-path/fixture changes only, identical 46-pass/9-known-fail shape) and `tests/contract/scene-nodes.contract.test.js` asserts the behavior both venues' `scene-nodes.js` must share, explicitly excluding Catharsis's extra token-aura layer as documented, accepted drift.
- Still open: browser acceptance/screenshots for rehearsal messaging, current-Scene changes, Start Show Session, and player Cue buttons in First Theater and Catharsis — Kernel 70A's manual checklist step covers this by hand where browser automation isn't available; capturing it as durable screenshot evidence remains unclaimed.
- Still open: repair the nine pre-existing `tests/first-theater/dice.test.js` failures — now duplicated (same 9, same cause) in `tests/catharsis/dice.test.js` too, since that suite is a faithful mirror. Both `scripts/test/alpha-gate.sh` and this roadmap track it as a named, non-blocking exception, not a silent gap.

### Product Journey Proof

- Prove account → Trailer Face → Production/roster → Show → staged Scene → linked Session → GO → durable aftermath.
- Observe non-developer users and record operator interventions, vocabulary confusion, and privacy misunderstandings.
- Turn that path into the short public demonstration of Victory.

## Medium Horizon

- Richer rehearsal editing and Scene version/diff presentation.
- Ruleset/module boundary so Socio is the first owned implementation rather than a permanent hardcode.
- Stronger director controls, admission/scheduling, and post-Show history.
- Accessibility: keyboard, touch, reduced motion, zoom, contrast, and non-spatial navigation.
- Security: CSRF, response headers/CSP, rate limits, upload limits, WebSocket origin policy, and authority audit logs.
- One-command release gate covering test DB setup, Go/frontend tests, migration replay, fresh install, privacy checks, and browser journeys.

## Long Horizon

- Multiple rulesets and anthology/integration proofs.
- Portable community/character archives and installation recovery.
- Convention, actual-play, school, or multi-director community workflows.
- Audiovisual capture only as a distinct future subsystem, never conflated with Showing Review or Scene capture.

## Current Non-Goals

- Microservices or a framework rewrite.
- A second stage-object authority model solely for Scenes.
- Victory-hosted Discord audio transport.
- Client-authored canonical dice, identity, authority, or durable stage truth.
- Broad marketplace work before one complete user journey is independently usable.

## Kernel Selection Rules

- Kernel numbers are stable labels; check the operator log before assigning one.
- Prefer the smallest end-to-end slice that proves a user outcome.
- Alternate capability work with integration, evidence, and consolidation work.
- A kernel is not complete until migrations, authority, privacy, projections, tests, fresh-install behavior, documentation, and known gaps agree.
