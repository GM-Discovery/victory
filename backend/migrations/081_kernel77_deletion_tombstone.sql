BEGIN;

-- Kernel 77 account deletion. The load-bearing decision (see
-- Construction/Security/kernel-76-data-classification.md §3 and
-- Construction/Privacy/account-deletion-policy.md): a deleting user's
-- private data is hard-deleted, but their footprint in shared canonical
-- history (actions, cue triggers, roster/audience-block provenance, Show
-- Run/Show/showing creation, inventory-acquisition provenance, permission
-- grants, and any asset still in active shared use) is reassigned to one
-- dedicated, credential-less tombstone account rather than left NULL or
-- deleted along with the history. This follows the existing precedent set
-- by the `discord_bridge` system account (migration 054): a reserved
-- handle, no password/Discord/session/membership rows, resolved the same
-- way any other `users` row is resolved, so no rendering code needs to
-- special-case a NULL actor.
--
-- This account must never be reachable via login (no auth.password_credentials
-- or auth.discord_identities row is ever created for it), must never hold a
-- membership or access grant, and must be excluded from any "list users" or
-- "last operator" surface by virtue of having neither.
INSERT INTO users (id, handle, display_name)
SELECT '00000000-0000-0000-0000-0000000000dd'::uuid, 'deleted-user', 'Deleted User'
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE handle = 'deleted-user'
);

COMMIT;
