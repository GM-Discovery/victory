# Kernel 2 — Security Notes (Identity, Auth, Invites, Access Floor)

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
- InfoBooth:
  - public
  - visible without authentication
- Workshop:
  - non-public
  - requires explicit access

### Intended Rule (Not Fully Enforced Yet)
- No ticket → no map visibility → no venue access

### Gap
- Map visibility endpoint not yet implemented
- Venue access still partially permissive

---

## 7. Join Flow Security (Current + Gap)

### Current State
- `JoinTheCave`:
  - uses membership role if authenticated
  - allows unauthenticated join as `audience`

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

## Status

Kernel 2 establishes:
- identity trust
- session control
- role assignment
- invite-based access foundation

It does **not yet fully enforce**:
- map visibility
- venue gating