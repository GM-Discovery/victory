# Canonical Role and Authority Resolution

**Kernel:** 97 (Canonical Role, Authority & Experience Reconciliation)
**Purpose:** implementation doctrine, not user-facing policy text. Answers, for any authority question in Victory, which function is the canonical source and what it actually checks — so a future change has one place to go, not a guess about which of several plausible-looking functions to trust.

---

## 1. Operator

**Canonical source:** `access.IsOperatorUser(ctx, pool, userID)` (`backend/internal/access/operator.go`).

Checks, in order: exact match against `OPERATOR_USER_ID` env var, or case-insensitive match against `OPERATOR_HANDLE` env var compared to the user's real `users.handle`. Nothing else. No hidden per-user special case exists anywhere in the backend (confirmed by inventory, ~50 call sites, all converging here).

**Known duplicate, not yet migrated:** `actions.isOperatorDiceRoller` (`backend/internal/actions/authority.go:887`) reimplements the identical check independently against a narrower `actionQuerier` interface, for one call site (dice-roll authority). Gives the same answer today; would silently drift if `IsOperatorUser`'s logic ever changes. Low priority — not fixed in this pass, noted for whenever that file is next touched.

Operator is full authority everywhere: `CanManage`, `CanViewBackstage`, `CanEnterVenue`, `CanParticipate` in `participation.Context` all resolve true, and every location-scoped check treats Operator as an automatic pass before even looking at membership rows.

---

## 2. Producer / Director (management authority)

**Canonical source:** `showruns.CanManageShowRun(ctx, pool, userID, locationID)` (`backend/internal/showruns/authority.go`).

Resolves true for Operator, or an active `location_memberships` row with role `producer` or `director` **at that specific location** (via `access.CurrentLocationRoleForLocation` — see §6). Producer and Director are treated as equivalent for *management* purposes — this function does not distinguish "specifically Director" from "Producer" the way some UI copy might imply. Used across ~30 files: shows, showtime, scenes, tickets, merchant, cohorts, directorprep, audienceadmission, stageobjects.

No "Producer silently becomes the active Director in live interaction" fallback was found anywhere in the inventory. If product intent ever wants Producer to inherit live-Director tools, that would need to be built explicitly, not assumed to exist.

**Backstage visibility is a separate, broader question** — see §4.

---

## 3. Director-specific live authority

Scene configuration, stage objects, cues, and live show controls mostly route through `CanManageShowRun` (§2) — there is no separate "is this specifically a Director, not a Producer" resolver for that class of action, by design (they're treated as the same management tier).

`cues.ExecuteCue` (`backend/internal/cues/execute.go`) uses its own `CanTriggerCue` check — a third, parallel authority gate alongside `CanManageShowRun` and `actions.CanAct` (§9). Not a conflict (it answers a genuinely different, narrower question), but worth knowing it exists as its own thing if cue authority is ever the subject of a future change.

---

## 4. Backstage visibility (Producer/Director/Operator/Crew)

**Canonical source:** `world.isBackstageRole(role)` / `stageobjects.IsBackstageRole(role)` (intentionally duplicated across two files to avoid an import cycle — a closed 3-line list, not a drift risk) and `messages.backstageNoteRoles()`.

All three answer: **can this role see backstage state at all** — Director, Producer, Operator, and **Crew**. This is a visibility question, not a management question — see §2 vs. this section is the "always scoped" distinction: Crew gets broad backstage *read* access but never broad *management* authority.

**Do not confuse this with `world.isShowManagementRole`**, which deliberately *excludes* Crew (per an explicit 2026-08-29 decision cited in that function's own comment) — that's the narrower "can actually manage the Show Run" question, matching `CanManageShowRun`. Two different questions, correctly different answers for Crew. This is not a bug and was verified as such during this kernel's reconciliation pass (2026-09-12) after initially being flagged as a possible conflict.

`messages.backstageNoteRoles()` includes Crew at the *session-role* level for reading backstage notes; any further per-action scoping ("explicitly authorized Stage Management Crew") is applied by the caller through `showruns` authority, not re-derived in the notes package itself.

---

## 5. Crew authority (bounded, contract-based)

**Canonical source:** `showruns.CanViewBackstage` (visibility, §4) plus `showruns.CanCrewPerformNonDestructiveEdit`.

`CanCrewPerformNonDestructiveEdit` reuses the identical boolean as `CanViewBackstage` — it does not independently verify an action is actually non-destructive. The scoping contract is: **every caller must itself be one of Kernel 70's named non-destructive actions.** Five live callers found (`merchant/interactions.go`, `merchant/kernel75_dialogue_authoring_http.go`, `merchant/equipment.go`, `cues/cues.go`, `scenes/composition.go`) — each was not individually re-verified against that contract in this pass; flagged as a maintainability risk (a new destructive action could be added by a future caller without the function itself catching it) rather than a confirmed active bug.

Crew must never inherit Director/Producer management authority (§2) through the backstage-visibility check (§4) — no call site was found doing this incorrectly.

---

## 6. Location / lot authority

**Canonical source:** `access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)` (`backend/internal/access/location_role.go`), or its convenience wrapper `access.CurrentDefaultLocationRole(ctx, pool, userID)` when the relevant location is this install's own (`access.DefaultLocationID`).

**The unscoped `access.CurrentLocationRole` (ignored `location_id` entirely, returned the user's single globally-best role across every Location they belong to) was removed in this pass.** It had 11 call sites that all needed the install's own default location, never a genuinely different one — migrated to `CurrentDefaultLocationRole`, not left as a redesign. See the Kernel 97 ledger for the full list and the cross-Location test that proves this holds.

Location membership governs the overall lot/campus, Third Place, and venue discovery. It must never be read as authorizing anything Show/Showing-specific — see §8.

---

## 7. Show Run roster (Cast / Player / Crew, in a specific Show)

**Canonical source:** `participation.ResolveParticipationContext(ctx, pool, userID, venueSlug, showRunID)` (`backend/internal/participation/resolver.go`).

Precedence (highest wins, nothing lower can downgrade a role a higher step already found):
1. Operator — full authority.
2. Active `location_memberships` row at the resolved location (producer/director only elevate here).
3. Active `show_run_roster_members` row at the resolved Show Run (**only reached if `showRunID` is non-empty** — this is the exact gap Kernel 97 fixed, see below).
4. Fallback: any active `location_memberships` row at all → audience.
5. Legacy `memberships`/`access_grants`, read as an additive upgrade only — never able to downgrade a canonical role, never reached unless steps 1-4 all found nothing.

**`LegacyLookupVenueRole`** (the same file) is the drop-in entry point `main.go`'s three `/api/world/*` snapshot handlers (`the-cave`, `catharsis`, `first-theater`) actually call. Kernel 97 fixed it to resolve the venue's current live/rehearsal session's Show Run (`sessions.show_id → shows.show_run_id`) before delegating to `ResolveParticipationContext`, instead of always passing `showRunID=""` — which used to skip Step 3 entirely and misclassify a genuine roster Player as audience. This means the three `main.go` call sites did not need to change at all; the fix lives entirely inside the resolver.

**Selected Character is canonical for Show participation.** `current_session_personas` and sitewide "active Character"/presence are confirmed **not** used as participation authority anywhere live — every remaining reference resolves which Character's *name/portrait* to display, never who's allowed to do what. `socio/mechanics.go` documents a third, narrower identity mechanism (equipped persona vs. roster selection) that caused a real bug, already fixed for its one HUD endpoint by switching to the roster source.

---

## 8. Venue access vs. production authority

**These are answered by different functions on purpose, and must stay that way.**

- **Venue access** (can this user open this venue at all): `access.UserCanAccessVenueSlug`. Purely an entry gate — confirmed at every call site (map, WebSocket connect, asset serving, tour eligibility) that it is never used as a stand-in for production authority.
- **Production authority** (edit, manage, see private data): `CanManageShowRun` (§2), `ewrite` authority, `participation` roster resolution (§7) — always a separate check, never a fallback from "can enter."

No shortcut was found anywhere treating venue-enterable as authorization for anything more.

---

## 9. WebSocket and command authority

**WebSocket:** all stage-mutation and chat/dice actions (`react/emote`, `chat/message`, `roll/dice`, `act/reveal_element`, `create/token`, etc.) route exclusively through `actions.CanAct` — there is no parallel HTTP path for the same actions, so there's no divergence risk by construction. `stageobjects.ApplyMutation` itself has no built-in authorization (trusts its callers); both current callers are already gated upstream by `CanAct` or `cues.CanTriggerCue` — a latent gap only if a future caller is added without going through a gate first.

**Commands:** `/api/commands/execute` dispatches "legacy" commands (`/showtime`, `/session`, `/mic`) by explicitly refusing to execute them itself and pointing at their real HTTP endpoints — true parity by construction, not a reimplementation. Non-legacy commands (`ooc`, `ic`, `char`, `bio`, `quote`, `journal`) route through the same `CanAct` gate as the equivalent GUI action.

**Conclusion: no HTTP/WebSocket/command divergence was found.** This is the one area where the inventory came back fully clean.

---

## 10. Frontend role labels and gating

Frontend role-conditional UI (drawing tools, stage-object reveal/hide, spot-checked — not exhaustive across every file) is backed by a real independent server check in every sampled case (`drawing.CanDraw`, `CanAct`).

**Fixed this pass:** `frontend/lib/stage-runtime/session-sync.js` used to silently overwrite an Operator's own real, server-resolved "audience" role to "producer" client-side. The server was never wrong; the UI lied about it. This is why an Operator could never see a true Audience experience through ordinary use, and is very likely the concrete cause of a long-standing personal impression that "every other view feels broken." Removed — the client now always displays the server's real resolved role.

---

## 11. Legacy access grants and memberships

`access_grants` is a boolean venue/production *reach* grant carrying no role of its own — explicitly documented and used as "an additive upgrade only," layered on top of the real role/roster model, never instead of it. The older `memberships` table is under-populated by newer code paths (a Producer/Director bootstrapped or signed up after Kernel 66 normally has no row there at all — this was the Kernel 70A bug already closed). Neither was found standing in as accidental primary truth for anything a newer Show/Showing-specific model should own.

---

## 12. Cohorts and presence

Cohorts are contextual grouping, not role. `stageobjects/projection.go` explicitly documents that backstage roles (§4) bypass cohort-scoping entirely — "a backstage viewer perceives everything" — matching spec intent that Producer/Director/Operator/Crew are never accidentally forced into player-cohort semantics.

Presence is confirmed **not** used as durable authority anywhere in the inventory — it indicates who's currently connected, nothing more.

---

## 13. "View as" / preview authority

**The only mechanism found:** "Preview as Player" (Kernel 73A), a narrow, read-only render of one Scene's resolved stage composition — tokens and index cards only. No chat, trays, navigation, presence, or notifications. It genuinely runs through the same canonical projection path (`stageobjects.CanPerceive`, constructing a real `Viewer{Backstage: false}`) a true Player would get, and cannot mutate the previewing account's stored role or leak privileged data.

**There is no general-purpose "view Victory as Role X" tool.** Every historical non-Operator verification (including Kernel 93's Audience dress rehearsal) required an agent manually fabricating a disposable account by hand-inserting rows into the database and forging a session cookie — not available to an Operator as ordinary end-user functionality. Combined with §10's fix, this means an Operator's only paths to experiencing another role have been either narrow (Scene-composition preview) or inaccessible (requires database access). Building a real, safe, general preview tool is out of this kernel's scope (§34 — no capability expansion) but is recorded here as a legitimate product gap worth a future kernel's attention.

---

## 14. What this document does not cover

- Every one of the ~30 `CanManageShowRun` call sites individually (the function is the canonical source; auditing every caller's own surrounding logic was out of scope for this pass).
- Every frontend role-conditional UI location (§10 covers what was sampled, not exhaustive coverage).
- Storyboards- and eWrite-specific permission nuances beyond what routes through the resolvers listed above.

Where this document and the code disagree in the future, the code is correct and this document is stale — update it, don't trust it blindly (matching the project's own memory-hygiene doctrine).
