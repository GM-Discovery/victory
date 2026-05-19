# Kernel 2 — Security Notes (Identity, Auth, Invites, Access Floor)

## Current Canon Note
This file is now partly historical.

Use [current-state.md](/opt/victory/Construction/current-state.md) for the live system view. Keep this file for security foundations, older kernel rationale, and historical implementation notes.

## Scope

This document records the **actual security posture** established during Kernel 2.

Kernel 2 is the **trust foundation layer**:
- identity
- authentication
- session control
- invite-based role assignment
- initial access gating primitives

This is not a complete security system. It is the first enforceable boundary.

---

## 1. Authentication & Session Model

### Method
- Server-side session model
- Session tokens stored **hashed** in database (`auth.sessions`)
- Cookie-based authentication (`victory_session`)

### Session Properties
- Tokens are:
  - randomly generated (32 bytes)
  - base64 encoded
  - SHA-256 hashed before storage
- Sessions include:
  - `user_id`
  - `expires_at`
  - `revoked_at`
  - `last_seen_at`
  - IP + user agent metadata

### Cookie Behavior
- HttpOnly
- SameSite=Lax
- Secure flag configurable via environment (`COOKIE_SECURE`)
- Expiration aligned to session TTL (24h currently)

### Security Guarantees
- No raw session tokens stored
- Sessions revocable server-side
- Expired sessions rejected at query time

---

## 2. Password Handling

### Hashing Algorithm
- Argon2id (via `golang.org/x/crypto/argon2`)

### Parameters
- time: 1
- memory: 64MB
- threads: 4
- key length: 32 bytes
- salt: 16 bytes (cryptographically random)

### Storage
- Stored in `auth.password_credentials`
- Format: encoded Argon2 string (`$argon2id$...`)

### Security Guarantees
- No plaintext passwords ever stored
- Salted hashing prevents rainbow table attacks
- Constant-time comparison used for verification

---

## 3. Password Reset Flow

### Token Behavior
- Random 32-byte token
- Base64 encoded for transport
- SHA-256 hashed before storage

### Storage
- `auth.password_reset_tokens`
- Includes:
  - `expires_at`
  - `consumed_at`
  - request metadata (IP, user agent)

### Flow
1. User requests reset
2. Token generated and stored (hashed)
3. Token logged (temporary dev behavior)
4. User submits token + new password
5. Token validated:
   - not expired
   - not consumed
6. Password updated
7. Token marked consumed
8. All existing sessions revoked

### Security Guarantees
- One-time-use tokens
- Expiration enforced
- Reset invalidates all prior sessions

---

## 4. Invite System (Access Control Backbone)

### Token Behavior
- Same pattern as reset tokens:
  - random
  - base64 encoded
  - SHA-256 hashed in DB

### Storage
- `invites` table
- Includes:
  - `target_role`
  - `location_id`
  - optional `production_id`
  - optional `venue_id`
  - `max_uses`
  - `uses_count`
  - `expires_at`

### Acceptance Rules
- Token must:
  - exist
  - not be expired
  - not be revoked
  - not exceed usage limit

### Role Assignment
- **Server assigns role**
- Role comes from invite record only
- Client cannot override role

### Membership Creation
- On acceptance:
  - `memberships` row created
  - scoped by:
    - location
    - optional production
    - optional venue

### Security Guarantees
- No client-side role elevation
- Role authority derived strictly from server state
- Invite tokens are single-use or limited-use

---

## 5. Role Authority Model (Current State)

### Roles
- producer
- director
- cast
- crew
- audience

### Enforcement
- Roles assigned via:
  - invite acceptance
  - initial producer bootstrap

### Important Rule
> Role is **never trusted from client payload**

### Current Behavior
- Authenticated users:
  - role derived from `memberships`
- Unauthenticated users:
  - currently default to `audience` (temporary)

---

## 6. Venue Visibility & Access (Initial Layer)

### Implemented
- `venues.is_public`
- `venues.is_workshop`

### Current Reality
Historical note:
The venue-access and map-visibility notes below describe the early access model and should be read as a foundation record, not a perfect reflection of every current venue/runtime surface.
- InfoBooth:
  - public
  - visible without authentication
- Workshop:
  - non-public
  - requires explicit access

### Intended Rule (Not Fully Enforced Yet)
- No ticket → no map visibility → no venue access

### Current Venue Exceptions
- Grant's Cabin stays invisible unless a venue grant exists
- Catharsis is signed-in only and remains ticketing-bound until the audience model is finalized

### Stage Element Control
- Context-menu actions still enforce server authority
- Reveal/hide is stage-surface scoped, not tray-scoped
- Audience cannot access the stage context menu

### Gap
- Map visibility endpoint not yet implemented
- Venue access still partially permissive

---

## 7. Join Flow Security (Current + Gap)

### Current State
- `JoinTheCave`:
  - uses membership role if authenticated
  - allows unauthenticated join as `audience`

Historical note:
- this section preserves an older security model discussion and should not be read as the live canonical state
- the current join code still creates audience users for unauthenticated joins, so any “remove anonymous audience join” note here is a pending recommendation, not a completed fact

### Problem
- Violates intended gate model:
  - “no ticket = no venue access”

### Required Correction
- Remove anonymous audience join
- Require:
  - valid session + membership
  - OR explicit venue admission policy

---

## 8. Data Integrity Protections

### Discord OAuth Security Notes
- Discord OAuth uses the authorization-code flow only
- OAuth state is random, hashed before storage, short-lived, and single-use
- callback rejects missing, invalid, expired, or reused state
- Discord access tokens are fetched server-side and are not stored for Kernel 32
- Discord client secrets and access tokens must not be logged
- Discord identity is only an authentication proof; Victory still authorizes roles, memberships, and venue access
- email is optional and must not be used as an automatic account-merge key

---

## 9. Presence + Attribution (Kernel 7)

### Presence Rules
- Presence is derived from authenticated WebSocket state
- Presence is in-memory only and is not canonical history
- Multiple open tabs for the same user are deduped in the visible roster
- The server broadcasts `presence/snapshot`, `presence/join`, and `presence/leave`

### Attribution Rules
- Speech, reaction, and reveal/hide actions are server-enriched with actor identity
- The client does not get to choose the actor for a stored action
- Outgoing action payloads should be treated as requests, not authority

### Security Guarantee
- A forged client payload cannot spoof another speaker or role in the stored/broadcast action stream

---

## 10. Identity Surface + Presence Repair (Kernel 8)

### Presence Rules
- The Cave does not allow anonymous presence
- Presence identity is resolved server-side from the authenticated session and session participant
- Connected users should always render with a non-blank label
- Multiple tabs for one user remain deduped in the visible roster

### Identity Rules
- `actor` is accountable identity
- `persona` is performed identity and is currently `null`

## Current Security Truth Addendum
- Auth remains cookie/session based with hashed session tokens in `auth.sessions`
- Performer profile editing is separated from character persona editing
- Character drafting currently follows performer-role access rather than separate draft-grant requirements
- Presence is still ephemeral connection state, not durable history
- Showing Review is about logs/actions/chat/reactions, not video capture
- Client-provided `actor` and `persona` values are ignored or overwritten
- Live and replayed actions use the same identity shape

### Security Guarantee
- A forged client payload cannot survive as trusted actor/persona data in the stored or broadcast action stream

### Constraints Enforced in DB
- role constraints (e.g., director requires production)
- invite usage limits
- uniqueness constraints on memberships
- hashed token uniqueness

### Transactional Safety
- signup, invite acceptance, password reset all use DB transactions
- partial writes prevented

---

## 9. Known Security Gaps (Must Be Addressed)

### High Priority
- [ ] Enforce venue access gates (no anonymous join)
- [ ] Implement map visibility endpoint
- [ ] Tie venue visibility to grants/memberships

### Medium Priority
- [ ] Rate limiting (auth + invites)
- [ ] Audit logging for:
  - login attempts
  - invite creation
  - invite acceptance

### Future (Workshop)
- [ ] File upload validation (type, size, signature)
- [ ] Image decode + re-encode (strip payloads)
- [ ] Storage path isolation per producer

---

## 10. Security Philosophy (As Implemented)

- Server is authoritative
- Client is untrusted
- Tokens are never stored raw
- Roles are never client-assigned
- Access is granted, never assumed
- Default stance: deny unless explicitly allowed

---

## 11. Kernel 9 Profile Surface Safety

### Safe Data Only
- Greenroom and Trailers store only performer-facing public fields
- Allowed fields are limited to:
  - stage name
  - pronouns
  - headshot URL or data URL
  - performance age range
  - bio
  - credits
  - skills
  - availability
  - public links

### Explicitly Excluded
- birthdate
- SSN
- credit card details
- bank details
- other sensitive identity records

### Access Rules
- Greenroom and Trailers are signed-in performer venues
- unauthenticated access is redirected away
- public profile API responses are not anonymous in this kernel

### Identity Rules
- `persona` remains `null`
- client identity claims are never trusted
- public profile state is server-resolved and draft/publish controlled

### Expressive Profile Fields
- The profile surface intentionally supports more than a handful of prompts
- Safe public fields may expand as long as they do not include birthdates, SSNs, card data, or other sensitive records
- Age is a performance range string, not a date of birth

### Operator Override
- `OPERATOR_HANDLE` or `OPERATOR_USER_ID` can mark an infrastructure operator who bypasses producer-gated admin surfaces
- The operator is outside the in-app production roster and is not represented as a producer membership row

### Operational Rule
- Age exposure is only a public performance range string
- Do not add a DOB field in later kernels unless the privacy model is rewritten intentionally

## 12. Kernel 10 Mailbox Boundaries

### Info Booth
- Public modal on the map
- No auth gate required
- Exists only as a front-door surface

### Mailbox
- Authenticated-only durable inbox
- Server enforces `to_user_id` ownership
- Users cannot read other users' mail
- `POST /api/messages` is operator/dev only

### Message Safety
- Body length is capped in the API
- `from_user_id` is server-resolved or null
- Client-provided identity fields are ignored

### Storage / Startup
- `messages` table is bootstrapped on server startup
- Migration file exists for durable setup
- Backend restart is required after code changes

## 13. Kernel 11 Note Card Boundaries

### Sender
- Must be authenticated
- Must be in the Cave session on the server
- Cannot spoof `from_user_id`

### Recipient
- Prefer a visible Cave participant
- May target a session participant by id
- Recipient role must be visible and allowed:
  - Director
  - Cast
  - Crew
- May fall back to the current director mailbox
- Users cannot target arbitrary inboxes

### Storage
- Note cards use `message_type = note_card`
- Context is server-owned
- `venue_slug` and `session_id` are stored for traceability

## Kernel 15 venue/permission notes

- `Producer's Office` is producer-only on the map.
- `The Director's Chair` is producer/director-only on the map.
- `GET /api/requests/incoming` is read-only and role-filtered by the server.
- `POST /api/requests/respond` is the only response path and writes reviewed_by/reviewed_at server-side.
- `GET /api/productions` is server-resolved and never trusts client-supplied production IDs as authority.
- `POST /api/invites` now validates venue slugs and production scope before storing a scoped invite.
- Office UIs show display names publicly; login handles are treated as internal identifiers.

### Limits
- Body capped at 250 characters
- No attachments
- No threading
- No realtime delivery

### Access
- Users can only read their own mailbox
- `POST /api/note-cards` is authenticated and session-gated
- `POST /api/messages` remains operator/dev only

## 14. Kernel 12 Index Card Boundaries

### Sender / Authority
- Index cards are created and edited by producer or director only
- Client identity claims are never trusted for card ownership
- The server decides whether `create/index_card` or `update/index_card` is allowed

### Storage
- Index cards are materialized as `elements.element_type = 'index_card'` and `elements.context_class = 'card'`
- Save actions are appended to the action log
- Card metadata is server-owned, including creator fields and timestamps

### Visibility
- Index cards are hidden from audience by default
- Director may reveal cards to cast/crew with the actor layer
- Director may reveal cards to audience with the audience layer
- Hiding uses the existing reveal/hide spine

### Limits
- Front + back text are capped at 2000 characters total
- No drag/drop
- No images
- No separate card universe

## 16. Context Class Distinction

### Storage
- `elements.context_class` is persisted in Postgres for later use
- `prop` means mobile
- `scenery` means fixed set piece / stage anchor
- `card` means index card
- future element classes should be stored explicitly rather than inferred only in the UI

### Current Seed
- `first-fire` is treated as fixed `scenery`

### Chat Boundary
- Venue chat must remain separate from `perform/speak`
- Kernel 21 chat should use `chat/message` and not stage speech semantics
- Chat is session-scoped and stored in the action log, not in a separate chat transport

## Kernel 22 Showing Model

- Presence is ephemeral and tells us who is connected right now.
- Showing is durable and records the theatrical event.
- `showing_id` is server-owned and now attaches to Cave actions.
- A closed showing must reject new chat, reveal/hide, overlay, and card placement.
- Reconnects should not create a new showing.
- The Cave chat lane stays in the bottom panel; `perform/speak` remains the stage speech lane.

## 15. Kernel 13 Workshop Placement Boundaries

### Authority
- `act/place_element` is server-authoritative.
- Producer and director may place index cards.
- Audience cannot create, send, or place cards.
- Client-provided authority and ownership claims are ignored.

### Venue Validation
- Target venues must have `config.index_cards_enabled = true`.
- The server rejects disabled or unknown venues.
- Placement is tied to the authenticated session participant and server-known venue access.

### Visibility
- Workshop is the source surface.
- Venue tray/backstage remains hidden from audience until the existing reveal system exposes it.
- Stage/worldspace uses the same reveal/hide spine.

---

## Status

Kernel 2 establishes:
- identity trust
- session control
- role assignment
- invite-based access foundation

It does **not yet fully enforce**:
- map visibility
- venue gating
