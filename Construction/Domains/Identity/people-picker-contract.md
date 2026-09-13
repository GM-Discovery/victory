# People Picker / Identity Resolution Contract (Kernel 85)

## The bug this closes

Storyboard sharing's suggestion list (`board.html`'s `renderMyPeopleSuggestions`) showed a related person by **stage name** (from `GET /api/player-relationships?state=active`), explicitly documented in its own comment as inert — "not a picker that grants anything by itself." The actual grant action, `POST /api/storyboards/{board_id}/grants` → `storyboards.AddGrant`, required a **Victory handle**, matched exactly against `users.handle`. Stage name and handle are two different, unrelated columns with no bridge between them, and `playerprofile` deliberately never projects `handle` into any Face-visible shape. A real, currently-related person was visible by one identity field and rejected when the *same visible name* was typed into the field the grant endpoint actually needed.

## The fix: select, don't type

`frontend/lib/people-picker.js` (`VictoryPeoplePicker.open({title, hint, onSelect})`) is a reusable modal merging two sources — My People (`GET /api/player-relationships?state=active`, using `subject.subject_profile_id`) and Third Place (`GET /api/third-place/headshots`, using `profile_id`) — de-duplicated by `profile_id`, My People's entry winning ties since it's the more deliberate "I know this person" signal. Selecting a row always yields the person's `profile_id`: the same opaque `player_profile_workbooks.id` identifier `playerrelationships.resolveSubjectUserID` and `tickets.resolveProfileUserID` already resolve server-side, in both cases via a narrow package-local duplicate of the same 10-line query — this codebase's established convention for this exact lookup, not a new cross-package dependency.

Server-side, `storyboards.AddGrantByProfile` (`grants.go`) resolves a `profile_id` the same way, and `HandleGrants`' POST body now accepts an optional `profile_id` field, preferred over `user_handle` when present. The handle path is untouched and still works for manual entry — the picker's fallback text field is relabeled to make clear it expects the raw Victory handle specifically, not a display name.

## Where it's wired in

- **Storyboard sharing** (`board.html`): selecting a suggested person now grants directly by `profile_id` — no handle transcription required.
- **Show People** (`show.html`, new "People" card): "Invite to this Show" calls the *existing* canonical Show admission mechanism — `tickets.InviteFromDirector` via `POST /api/show-runs/{id}/tickets/invite`, targeting the Show's `show_run_id`. This is deliberately a two-punch flow (the invited person must accept) rather than an instant grant, matching kernel-85 §7.3's "use the current canonical Show admission/access model... do not create an alternate membership universe." No new grant table or membership concept was added.
- **Show Run roster** (`roster.html`): the picker fills the existing manual-paste field as a convenience, not a replacement.

## The handle/canonical-ID display resolution

Kernel-85 §7.2 asks the picker to show "display name/Face," "handle," and a "canonical stable user ID visible/copyable where useful." The repository audit (kernel doc §13.10) found this codebase deliberately never exposes the raw account UUID or Victory handle through any people-facing projection (`playerprofile/types.go`: Face projections "never include email, handle, account [UUID]") — a privacy boundary established on purpose in earlier kernels (62, 65), not an oversight the spec text anticipated.

**Resolution:** the picker shows display name/portrait plus the `profile_id` itself, labeled "ID," as the canonical/copyable identifier — satisfying "enough identity information to know I have the right person" and the letter of §7.2's "canonical stable user ID" without reopening the handle/UUID exposure those earlier kernels deliberately closed. It does not show the Victory handle. This is a considered substitution, not an oversight — flagged explicitly here per the kernel's own "no ask Grant to restate repository-answerable facts" instruction, since the audit already answered why.

## Tests

`backend/internal/storyboards/kernel85_grants_profile_dbtest_test.go` (profile-id grant success, malformed profile_id → `ErrUserNotFound`, non-owner rejected), `backend/internal/tickets/tickets_test.go`'s `TestInviteFromDirectorRejectsNonDirectorActor`, `backend/internal/access/kernel85_requestable_venues_dbtest_test.go` (see `Construction/OperatorLogs/kernel-85-reportback.md` for the Audition Hall / locked-door pieces).
