# Victory Account Deletion Policy

**Status:** implemented, Kernel 77 (2026-08-01).
**Code:** `backend/internal/identity/account_deletion.go`.
**Migrations:** `081_kernel77_deletion_tombstone.sql`, `082_kernel77_deletion_export_recovery.sql`.

**Controlling principle** (from Kernel 76's data classification work, carried forward here): a
user may erase *themselves* but not other people's history of a shared performance.

---

## 1. What gets hard-deleted

Everything already `ON DELETE CASCADE` from `users` — verified correct-as-is by the Kernel 76
audit and unchanged by Kernel 77:

- password credentials, sessions, OAuth/Discord identity links;
- private journals, relationship records and their notes, drafts;
- player profile workbooks and facts;
- Characters and their workbook entries, skills, face overrides;
- memberships (Location and Production-scoped);
- a deleted user's own received-message copies (`messages.to_user_id`);
- aftercare submissions and drafts, tutorial/interaction progress records.

Plus, explicitly by the deletion service (not by a plain FK cascade): uploaded assets
exclusively owned/produced/uploaded by the user, *unless* still in active shared use — see §3.

## 2. What gets detached (`SET NULL`), unchanged from Kernel 76

Provenance-only columns — "who granted/created/authored this" — across roughly forty tables
(`location_memberships.granted_by_user_id`, `productions.created_by_user_id`,
`messages.from_user_id`, and similar). These were already nullable and already correct; Kernel
77 did not touch them.

## 3. What gets anonymized instead of deleted or nulled

Twelve columns were `ON DELETE RESTRICT` — the actual reason account deletion was structurally
impossible before Kernel 77 (`K76-M04`). Each is shared-history attribution, not the user's own
private property, so deleting the user reassigns these to a dedicated tombstone account
(`users.handle = 'deleted-user'`, `display_name = 'Deleted User'`, created by migration 081, no
credentials, no session, no membership, never resolvable via login) rather than being left NULL
or cascade-deleted:

```
actions.actor_id
cue_executions.triggered_by_user_id
character_inventory_items.acquired_by_user_id
permission_grants.granted_by_user_id
show_run_audience_blocks.blocked_by_user_id
show_run_roster_members.added_by_user_id
show_runs.created_by_user_id
showings.created_by
shows.created_by_user_id
assets.owner_user_id / producer_user_id / uploader_user_id  (only when still in active shared use, see below)
```

This follows the same pattern already established for bot-attributed actions: the
`discord_bridge` system account (migration 054). No rendering code needed to change — anything
that already joins to `users.display_name` shows "Deleted User" automatically. None of these
twelve columns participate in a unique constraint alongside the reassigned value, so many
different deleted users collapsing onto the same tombstone id cannot violate a uniqueness
constraint.

**Assets are the one nuanced case.** They are classified Production-confidential, not private
to the uploader (Kernel 76 §7.6), and the spec's own recommendation ("hard delete rows and
files") has to yield when an asset is load-bearing for something still live: `assets` FKs are
RESTRICT and `venue_active_maps.asset_id` is *also* RESTRICT, so an asset currently serving as a
venue's active map cannot be hard-deleted without breaking that venue regardless of what the
uploader does. The deletion service checks this specifically: an asset in active use as a
venue's current map is anonymized (attribution reassigned to the tombstone, file and row kept);
every other asset the user exclusively owned/produced/uploaded is hard-deleted, row and file
both.

## 4. Blocking conditions

Only one, because it's the only one with no safe automatic resolution in this schema:

**Sole active Producer at a Location with any Production.** Authority to administer a
Location's Productions and Show Runs comes from an active `producer` role membership at that
Location — not per-Production ownership (there is no such column; `productions.created_by_user_id`
is mere provenance, already `SET NULL`). If deleting this user would leave a Location with
Productions and no one able to manage them, deletion is blocked with the specific Location
named and a resolution: grant Producer access to another member (through Victory's existing
invite/grant flow — no new "transfer ownership" UI was built, because the existing one already
does this) and retry.

**The operator account is also blocked**, unconditionally — self-deleting the account matching
`OPERATOR_HANDLE` is refused outright; changing operator authority to a different account is an
operator-shell action (edit `.env`, redeploy), not self-service.

## 5. Confirmation model

Two independent checks, both required, chosen to work uniformly whether or not the account has
a password (most accounts don't — password signup has been closed since Kernel 76, so most
users are Discord-only):

1. **Recent authentication.** An account with a password confirms with it (identical check to
   `HandleUpdateAccountEmail`). An account without one must have a session younger than 15
   minutes — in practice, sign out and back in with Discord immediately before deleting.
2. **Deliberate confirmation phrase.** The exact account handle, case-sensitive, typed by the
   client and checked server-side against the session-resolved account — never a client-supplied
   target id.

## 6. What's returned afterward

A deletion receipt (`account_deletion_receipts`): a fresh random id unrelated to the deleted
user's original id, counts of deleted/anonymized rows by the representative categories in
§7.6 of the kernel spec, and a status. No email, handle, or Discord id is retained anywhere.

## 7. Tests

`backend/internal/identity/account_deletion_test.go` — private-only deletion succeeds and
cascades correctly; a shared Action survives reassigned to the tombstone (not deleted, not
NULL); sole-producer blocks and clears once a second producer is granted; the operator account
is refused; a confirm-handle mismatch is refused; an unauthenticated request is refused. Full
`go test ./...` passes.
