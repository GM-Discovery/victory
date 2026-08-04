# eWrite Permissions Matrix (Kernel 78)

Implementation: `backend/internal/ewrite/authority.go`. Every helper:
Operator short-circuit first, then `access.CurrentLocationRoleForLocation`
(never the global-best `CurrentLocationRole`), then named grants. Client
payloads select rows; they never carry authority.

## Matrix

| Capability | Operator | Producer/Director (in location) | Crew (in location) | Cast | Audience | Anonymous |
|---|---|---|---|---|---|---|
| See Writer's Room map tile | ✓ | ✓ | ✓ | — | — | — |
| Create collections/publications | ✓ | ✓ | ✓ | — | — | — |
| Edit a publication | ✓ | ✓ (all in location) | own / named grant only | — | — | — |
| Read a draft | ✓ | ✓ | own / named grant only | — | — | — |
| Publish/unpublish | ✓ | ✓ | own / `publish` grant | — | — | — |
| Delete a (non-published) publication | ✓ | ✓ | own only | — | — | — |
| Manage named editors | ✓ | ✓ | `manage_editors` grant | — | — | — |
| Create object links | ✓ | ✓ | edit authority on pub AND Crew+ at item's location | — | — | — |
| Read published `authenticated`/`public` | ✓ | ✓ | ✓ | ✓ | ✓ | — (deferral) |
| Read published `production` | ✓ | active membership at the publication's location, any role | ← | ← | ← | — |
| Search results | same predicates as reading; drafts structurally excluded (`status='published'` in SQL) | | | | | |

## Named grants (`ewrite_editors`)

`edit` < `publish` < `manage_editors` are independent kinds, not a ladder —
an `edit` grant cannot publish (tested). Grants are per-publication.

## Two recorded contrasts

1. **Draft read = edit authority**, deliberately NOT the storysofar
   owner-only precedent (`storysofar/store.go:210`): eWrite drafts are
   production work product shared within the authoring team, not private
   reflections. Producers/Directors in scope can read Crew drafts.
2. **Creator authority requires a live authoring role.** A creator demoted
   to audience keeps nothing — `CanEditPublication` re-checks the role
   before honoring `created_by`.

## Visibility deferral

`public` visibility exists in schema and API but is served to authenticated
readers only in Kernel 78. The first anonymous content API is deliberately
deferred (operator decision in the kernel spec amendments); when it ships,
only `CanReadPublication`'s `public` arm and a new unauthenticated route
change.

## Error-shape rule

A reader denied a publication gets `publication_not_found`, not
`forbidden` — a denial must not confirm a hidden title exists (spec 6.3).
