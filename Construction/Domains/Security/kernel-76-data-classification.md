# Kernel 76 — Data Classification, Deletion, and Export Map

Covers the 92 tables present in the `victory` database as of the Kernel 76 rebuild.
Classes follow Kernel 76 §7.6.

---

## 1. Classes

| Class | Meaning | Who may read |
|---|---|---|
| **P** Public | safe for anonymous | anyone |
| **A** Authenticated | any signed-in account | members |
| **C** Production-confidential | scoped to a Production/Location | its staff and cast |
| **U** Private to user | the owner alone | owner only |
| **S** Authentication secret | never rendered to anyone | nobody |
| **O** Operator/deployment secret | infrastructure | operator shell |
| **T** Disposable test data | may be destroyed | — |

---

## 2. Classification

### Authentication (S)

| Table | Class | Note |
|---|---|---|
| `auth.sessions` | S | SHA-256 hashed tokens; raw never stored |
| `auth.password_credentials` | S | Argon2id — t=1, m=64 MiB, p=4, 16-byte salt, 32-byte key |
| `auth.password_reset_tokens` | S | hashed, 1h TTL, single-use |
| `auth.oauth_states` | S | hashed, expiring, consumed atomically |
| `auth.discord_identities` | U | stable Discord id is the external identifier |
| `auth.discord_gateway_settings` / `_state` | O | closed to non-operators by K76-M02 |

### Private to the user (U)

`character_journals`, `player_relationships` and its five satellite tables
(`_facts`, `_events`, `_followups`, `_journal_entries`, `_categories`), `messages` (direct),
`player_profile_workbooks` and `_facts` while unpublished, `aftercare_submissions`,
`aftercare_response_drafts`, `participant_freeform_submissions`,
`participant_local_projections`.

This is the material Kernel 76 §1.5 says Grant must not see in ordinary operation. Verified:
no operator-facing route renders journals, relationship notes, or message bodies, and request
logging records method, path, and duration only.

### Production-confidential (C)

`productions`, `shows`, `show_runs`, `show_run_roster_members`, `show_run_tickets`,
`show_run_audience_blocks`, `scenes`, `show_scene_placements`, `scene_stage_elements`,
`stage_element_bindings`, `cues`, `cue_executions`, `actions`, `sessions`,
`session_participants`, `showings`, `elements`, `libraries`, `venue_layout_elements`,
`venue_active_maps`, `venue_grid_configs`, `assets`, `asset_derivatives`,
`command_execution_receipts`, `dialogue_packets` and topics, `merchant_packets`,
`equipment_items`, `character_inventory_items`, `backstage-notes`.

`productions` was disclosed to every Location member until K76-M01.

### Authenticated (A)

`users` (handle, display name — not email), `third_place_headshots` **once admitted**
(K76-H02), published `performer_profiles`, `locations`, `venues`, `lots`.

### Public (P)

`venues` slugs and names for the map fog resolver, and the static frontend. Nothing else.

### Operator (O)

`/opt/victory/.env`, PostgreSQL role credential, Discord client secret and bot token,
`warehouse_storage_settings`.

---

## 3. Deletion dependency map (§5.17)

**Account deletion does not currently work.** Attempting it during the Kernel 76 purge:

```
ERROR: update or delete on table "users" violates foreign key constraint
"actions_actor_id_fkey" on table "actions"
```

Foreign keys to `users` fall into three groups, and the third is the problem:

| Behaviour | Tables | Disposition on deletion |
|---|---|---|
| `ON DELETE CASCADE` | sessions, password credentials, discord identities, location memberships, player profile tables, relationships, headshots | **hard delete** — correct as-is |
| `ON DELETE SET NULL` | `location_memberships.granted_by_user_id` and similar provenance columns | **detach** — correct as-is |
| **No action (blocks)** | `actions.actor_id`, and other authored-history references | **must be decided** |

**Recommended per-class disposition for Kernel 77:**

| Record class | Disposition | Why |
|---|---|---|
| Credentials, sessions, OAuth links | hard delete | no residual value |
| Private journals, notes, relationships, drafts | hard delete | the user's alone |
| Direct messages | hard delete sender copy; retain recipient copy with anonymised author | the recipient's copy is also *their* record |
| Authored `actions` | **preserve with anonymised author** | Actions are shared Production history; cascade-deleting them would corrupt other people's Show record |
| Characters, workbooks | hard delete unless staged in a Show that has run; then anonymise | |
| Uploaded assets | hard delete rows and files | |
| Productions/Show Runs the user owns | **block deletion until ownership transfers** | deleting a Producer must not delete the Production |
| Memberships | hard delete | |

The controlling principle: a user may erase *themselves* but not other people's history of a
shared performance. That is the same distinction Victory already draws between a person and
their Character.

---

## 4. Export map (§5.17)

Nothing exists. The minimum viable export for Kernel 77 is a single JSON document plus a
files directory, covering: account identity, profile and workbook, Characters and workbooks,
journals (Markdown), relationships and private notes, messages sent and received, Show/Session
participation, aftercare submissions, and uploaded assets as files.

Excluded by construction: password hashes, session tokens, reset tokens, OAuth state, and any
other user's private content.

---

## 5. Markdown and HTML pre-audit (§5.12)

Victory Documents begins in Kernel 78. Requirements to be met **before** any user-authored
Markdown is accepted:

1. **Raw HTML disabled** in the Markdown renderer. Not sanitised — disabled.
2. **Server-side sanitisation** with a maintained allowlist library (`bluemonday` UGC policy
   is the natural Go choice), applied on read as well as write.
3. **Allowed:** headings, emphasis, lists, links, block/inline code, tables, blockquotes,
   images from Victory-hosted `/api/assets/…` URLs only.
4. **Forbidden:** `<script>`, `<style>`, `<iframe>`, `<object>`, `<embed>`, `<form>`, every
   `on*` handler, and the `javascript:`, `data:`, `vbscript:` URL schemes.
5. **Links** rendered with `rel="noopener noreferrer nofollow"`; external targets visibly
   marked.
6. **Store the original, render the sanitised form.** Sanitising destructively on write makes
   a future policy fix unable to repair already-stored content.
7. **Regression payloads** in the test suite from day one: `<img src=x onerror=alert(1)>`,
   `[x](javascript:alert(1))`, `<svg/onload=alert(1)>`, nested-encoding variants, and a
   `data:text/html;base64,…` link.

Victory's existing upload path already rejects SVG and re-encodes images, so the Documents
work inherits a good position — the risk is entirely in the text rendering path.
