# Kernel 15 Producer's Office + Director's Chair

## What shipped

- `Producer's Office` is visible to producers and operators.
- `The Director's Chair` is visible to producers, directors, and operators.
- The map now shows notification counts for pending request queues.
- Producers can review pending director requests from the office.
- Producers and directors can issue scoped invites through the office pages.
- `GET /api/requests/incoming` returns the review queue for the current authority tier.
- `GET /api/productions` returns productions for invite scoping.

## Permission flow

- Producers can invite director, cast, crew, or audience scopes.
- Directors can invite cast, crew, or audience scopes below them.
- Audience invites are venue-scoped and are meant for simple look-around access.
- Director/cast/crew invites are production-scoped and use the existing invite/membership seam.
- Incoming requests can now be approved or denied directly from the office queues.
- Office request cards show the public display name first. Handles are treated as internal login identifiers.

## Live notes

- The live backend on port `8081` was restarted with the new host process so the browser sees the updated handlers.
- The new venue rows were seeded into the live database after the first map check revealed they were missing.
- The current production for scoping is `Main Production` in `amurray-family`.
