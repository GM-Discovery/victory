# Kernel Report Back - Kernel 59

**Note:** this report is retroactive. Kernel 59 shipped in a single commit (`f324901`, "Implement command registry and execution for venue slash commands") with no plan doc saved under `Construction/Kernels/` and no reportback filed at the time. This document was written afterward, during Kernel 60 development, by reviewing the shipped code and by drawing on Kernel 60's live isolated-database verification (which exercises the same idempotency and command-dispatch machinery kernel 59 built).

## 1. Status
PASS WITH MINOR FINDINGS

Kernel 59 built the single server-authoritative command registry, resolver, and execution surface for venue slash commands: `/char set`, `/char face|mechanics|history|journal` (navigate), `/bio set`, `/quote set`, `/journal add|recent`, `/ooc`, and `/help`, plus the `/api/commands/{available,preview,execute}` HTTP surface and idempotent execution via `command_execution_receipts`. It deliberately scoped out `/value`, `/override`, skill-add, and collection-item commands because the domain layer they'd need didn't exist yet — an honest, well-documented descope that Kernel 60 later built on top of.

## 2. What Was Built

- `backend/internal/commands/registry.go` — the `Command` metadata shape and `AllCommands()`/`Find()`/`Available()`, the single source of truth for command metadata shared by parsing, `/help`, and the frontend command palette.
- `backend/internal/commands/resolver.go` — `ResolveActiveCharacter`, composing session-scoped persona equip and account-level active character (two sources that existed independently but had never been combined).
- `backend/internal/commands/idempotency.go` — `ExecuteIdempotent`, modeled on the existing coin-flip unique-index + `ON CONFLICT DO NOTHING` + read-back pattern, backed by migration `032_kernel59_command_registry.sql` (`command_execution_receipts`).
- `backend/internal/commands/char.go`, `bio.go`, `quote.go`, `journal.go`, `ooc.go`, `help.go` — thin execute-layer wrappers around existing `characters`/`actions` package functions.
- `backend/internal/actions/ooc.go` — `StoreOOCMessage`, modeled directly on `StoreChatMessage`, stamping `chat/ooc` instead of `chat/message`; `chat/ooc` added to `CanAct`'s allowlist reusing the same chat-enabled/talking-enabled/role gate as regular chat.
- `backend/internal/network/commands_http.go` — `/api/commands/available|preview|execute` HTTP handlers and per-command dispatch, wrapping mutating commands in `ExecuteIdempotent`.
- `frontend/lib/victory-command-palette.js` — the client-side slash-command parser and executor.
- Journal entries now thread the venue/session they were written from (columns existed since migration 031 but no call site populated them until this kernel).

## 3. Evidence (original commit)

Files changed in `f324901` (from `git show --stat`): `backend/cmd/victory/main.go`, `backend/internal/actions/{authority.go,authority_test.go,ooc.go,ooc_test.go}`, `backend/internal/characters/{characters.go,journal.go,journal_venue_session_test.go}`, `backend/internal/commands/*.go` (registry, resolver, idempotency, char, bio, quote, journal, ooc, help + tests), `backend/internal/network/commands_http.go`, `database/migrations/032_kernel59_command_registry.sql`, `frontend/lib/victory-command-palette.js`.

No reportback evidence (test output, browser checks) was recorded at ship time. Retroactive checks run during this review:

- `cd backend && go test ./internal/commands/... -v` — all 14 tests pass (registry, resolver guard clause, idempotency guard clauses, help, char).
- Live verification during Kernel 60 work: `ExecuteIdempotent`'s real database-backed dedup/replay path (the part `idempotency_test.go` can't cover without a DB) was exercised end-to-end via repeated `/char advance` calls against an isolated database — same call returned the identical stored result on retry, confirming the receipt table and replay logic work correctly in practice.

## 4. Findings From This Review

- **Latent field inconsistency (harmless today)**: `ActivePersonaForSession` (new in this kernel, `characters.go`) returns a persona map without `workbook_status`/`workbook_context`, while the pre-existing `ActiveCharacterForUser` (the account-level fallback in `ResolveActiveCharacter`) includes both. Since `ResolveActiveCharacter` prefers the session-scoped source when present, the shape of `active_character` in `/api/commands/available`'s response differs depending on which source resolved it. No current consumer reads those two fields from this particular response, so it's inert, but it will surprise whoever adds one.
- **Untested precedence branch**: `resolver_test.go` only covers the empty-`userID` guard clause. The actual precedence logic in `ResolveActiveCharacter` — session-persona-wins-over-account-level, and the `pgx.ErrNoRows` fallback path — has no test coverage and was not exercised by Kernel 60's verification either (that work went through the account-level `active_user_characters` path, not session-equipped personas).
- **No test files** for `bio.go`, `quote.go`, `journal.go`, or `ooc.go` in the commands package (unlike `char.go`, which has `char_test.go`). These are thin wrappers with no guard clauses of their own — defensible to leave untested, but worth naming since it's a real gap relative to the package's own convention.
- No correctness bugs found in the reviewed code. The `chat/ooc` authority reuse, the registry's `Find`/`Available` filtering, the legacy-command guardrails (`/mic`, `/session` explicitly rejected from the generic executor), and the idempotency receipt schema are all sound.

## 5. Operator Notes

- This kernel's own doc comments in `registry.go` are explicit about scope: "/value, /override, skill-add, and collection-item commands" were excluded because their domain layer (a mechanical value registry, persisted priority/lock state, a collection-item store, real Director authority) didn't exist yet. Kernel 60 later built the skill domain layer and consumed this registry directly (`/char add skill`, `/char skills`, `/char advance`), validating that the registry's extension points work as intended.
- `/roll` was already registered with alias `/r` in this kernel's registry — Kernel 60's draft incorrectly listed registering `/r` as still-pending work; that was corrected during Kernel 60's review.

## 6. Known Issues

- No plan doc exists under `Construction/Kernels/` for kernel 59 (numbering jumps from kernel-31 to kernel-60 in that directory).
- No reportback existed until this retroactive one.
- See "Findings From This Review" above for the two minor code-level gaps.

## 7. Next Recommended Step

- If a future kernel adds a UI surface that reads `active_character.workbook_status` from `/api/commands/available`, fix `ActivePersonaForSession` to include it first.
- Add unit coverage for `ResolveActiveCharacter`'s precedence branches (would need a DB fixture, consistent with how the rest of this package tests only guard clauses).
