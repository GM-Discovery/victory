# Victory Account Deletion — Schema Dependency Map

**Purpose.** The full inventory of foreign keys to `users(id)`, grouped by delete behavior, that
`account-deletion-policy.md` reasons about. Captured 2026-08-01 against the live schema (81
migrations at the time of capture; 083 after Kernel 77's own migrations).

Query used to regenerate this list:

```sql
SELECT conrelid::regclass AS table_name, a.attname AS column_name, confdeltype
FROM pg_constraint c
JOIN unnest(c.conkey) WITH ORDINALITY AS ck(attnum, ord) ON true
JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ck.attnum
WHERE c.contype = 'f' AND c.confrelid = 'users'::regclass
ORDER BY confdeltype, table_name;
```

---

## CASCADE — hard delete, correct as-is (Kernel 76, unchanged by Kernel 77)

```
access_grants.user_id
active_user_characters.user_id
aftercare_response_drafts.user_id
aftercare_skips.user_id
aftercare_submissions.user_id
character_cards.owner_user_id
character_interaction_attempts.actor_user_id
character_journals.author_user_id
character_workbook_entries.author_user_id
character_workbook_rolls.owner_user_id
command_execution_receipts.actor_user_id
current_session_personas.user_id
location_memberships.user_id
memberships.user_id
messages.to_user_id
participant_dialogue_topic_views.user_id
participant_freeform_submissions.user_id
participant_local_projections.user_id
participant_tutorial_progress.user_id
auth.password_credentials.user_id
auth.password_reset_tokens.user_id
auth.email_verification_tokens.user_id        (Kernel 77)
account_export_jobs.user_id                     (Kernel 77)
performer_profiles.user_id
permission_grants.grantee_user_id
player_profile_events.user_id
player_profile_workbooks.user_id
player_recognition_grants.user_id (as recipient)
player_relationship_events.user_id (via relationship)
player_relationship_journal_entries (via relationship)
player_relationships.observer_user_id
player_relationships.subject_user_id
player_stage_name_history.user_id
session_participants.user_id
auth.sessions.user_id
show_run_audience_blocks.user_id
show_run_roster_members.user_id
show_run_tickets.user_id
third_place_headshots.user_id
auth.discord_identities.user_id
invites.inviter_user_id
```

Note on `player_relationships`: **both** `observer_user_id` and `subject_user_id` are CASCADE.
Deleting either party in a relationship pair removes the relationship row, including the
*other* party's private note about them. This is pre-existing Kernel-76-vetted behavior, not
introduced by Kernel 77 — flagged here because it is the one CASCADE case that touches
someone other than the account holder, worth remembering if this schema is ever revisited.

## SET NULL — detach, correct as-is

```
access_grants.granted_by_user_id
aftercare_submissions.surprising_user_id
character_story_events.authored_by_user_id
cues.created_by_user_id
dialogue_packet_revisions.edited_by_user_id
dialogue_packets.updated_by_user_id
dialogue_topics.updated_by_user_id
auth.discord_chat_imports.* (edit_action_id, etc.)
equipment_items.created_by_user_id
location_memberships.granted_by_user_id
memberships.granted_by_user_id
merchant_packets.created_by_user_id
messages.from_user_id
participant_interactions.created_by_user_id
productions.created_by_user_id
scene_stage_elements.created_by_user_id
scenes.created_by_user_id
show_run_tickets.director_punched_by_user_id
show_run_tickets.player_punched_by_user_id
show_scene_placements.created_by_user_id
stage_element_bindings.created_by_user_id
venue_active_maps.created_by_user_id
venue_active_maps.updated_by_user_id
venue_grid_configs.updated_by_user_id
warehouse_storage_settings.updated_by
show_run_roster_members.character_card_id (not a users FK, but detaches when the *character* is deleted)
```

## RESTRICT — the twelve columns Kernel 77 had to resolve

```
actions.actor_id
assets.owner_user_id
assets.producer_user_id
assets.uploader_user_id
character_inventory_items.acquired_by_user_id
cue_executions.triggered_by_user_id
permission_grants.granted_by_user_id
show_run_audience_blocks.blocked_by_user_id
show_run_roster_members.added_by_user_id
show_runs.created_by_user_id
showings.created_by
shows.created_by_user_id
```

Disposition for each: see `account-deletion-policy.md` §3. Summary — all twelve reassign to the
tombstone `deleted-user` account (migration 081) rather than blocking deletion, except that
`assets.*` specifically anonymizes-in-place only when the asset is still a venue's active map
(itself RESTRICT via `venue_active_maps.asset_id`); every other exclusively-owned asset is hard
deleted, row and file.

## Uploaded files on disk (not a database FK, but part of the same accounting)

`assets.storage_root`/`original_path` point into `STORAGE_ROOT/producers/{producer_user_id}/assets/{asset_id}/`
(see `internal/assets/upload.go`). The deletion service captures `producer_user_id` **before**
any reassignment so it can compute the correct on-disk path, deletes the row inside the same
transaction as everything else, and removes the directory from disk only after the transaction
commits (filesystem operations aren't transactional with PostgreSQL, so ordering matters: never
delete the file before the row is durably gone).

## What blocks deletion in the application, not the schema

Only one condition, and it has no FK behind it at all: being the sole active `producer`-role
`location_memberships` row at a Location that has any `productions` row. See
`account-deletion-policy.md` §4 for why this can't be expressed as a foreign-key constraint in
this schema (Production authority is Location-scoped, not per-row-owned).
