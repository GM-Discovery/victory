# Kernel 9 Greenroom + Trailers Public Profile Surface

## What is enforced

- `The Greenroom` is the signed-in public profile venue.
- `Trailers` is the signed-in edit venue for the same profile surface.
- Profiles use a server-owned `performer_profiles` table with draft and published state.
- The public projection excludes draft-only values.
- `persona` remains `null` for now and is reserved for future production characters.

## Safe profile fields

The profile table only stores safe public-facing performer data:

- stage name
- pronouns
- headshot URL or data URL
- performance age range
- favorite fun
- most relaxed
- favorite color
- favorite artist
- favorite food
- favorite song
- favorite place
- favorite movie or show
- hidden talent
- ideal day
- bio
- credits
- skills
- availability
- public links

## What is not stored

- birthdate
- SSN
- credit card data
- bank data
- social-security-adjacent secrets
- private backstage identity records

## Identity rules

- account identity remains the accountable surface
- `persona` stays `null`
- the profile page may render `persona` if a future kernel adds it, but Kernel 9 does not
- the Greenroom display uses:
  - persona name when present
  - otherwise display name
  - otherwise handle
  - otherwise shortened user id

## Access rules

- Greenroom and Trailers are not anonymous venues
- signed-in users only
- unauthenticated requests are redirected to the forbidden access screen
- public profile API responses still require authentication in this kernel

## Draft / publish rules

- edits save to draft first
- publish copies the current draft into the public projection
- published projection does not change until publish is pressed
- Greenroom renders the published projection only
- Trailers renders draft edits and published preview side by side
- the profile surface is intentionally expressive so performers can represent many parts of themselves safely

## Backend surfaces

- profile API: `backend/internal/profiles/profiles.go`
- venue visibility: `backend/internal/access/visibility.go`
- map routing: `frontend/app.js`

## Frontend surfaces

- Greenroom view: `frontend/venues/greenroom/index.html`
- Trailers view: `frontend/venues/trailers/index.html`
- shared identity helper: `frontend/lib/identity.js`

## Operator notes

- No new environment variables were added.
- No new dependencies were added.
- Backend restart is required after code changes.
- The profile image upload uses a data URL stored in the profile table for now.
- Age must stay a performance range, not a birthdate.
