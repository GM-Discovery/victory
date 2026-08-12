# In Character Chat Contract — Kernel 87

Canon for the In Character chat tab: server-resolved speaker identity, impersonation resistance, and the "choose a Character first" failure. Backend: `backend/internal/actions/ic_chat.go`, `backend/internal/commands/ic.go`. Frontend: `frontend/lib/stage-runtime/runtime.js` (IC tab wiring), `frontend/venues/catharsis/index.html` (tab markup).

## 1. Chain (kernel §10)

```text
authenticated user
→ server resolves current Show/participation Character
→ stores accountable user identity
→ stores/presents Character speaker identity
→ broadcasts message
```

Implemented end to end in `actions.StoreICChatMessage`. The **only** client-authored content is the message text; `ICChatMessageRequest` (its request type) has `SessionID`, `ActorID` (from the authenticated session, never the request body — mirrors `StoreChatMessage`'s existing convention), and `Text`. **There is no `character_id` field on the request type at all** — not validated-and-rejected, structurally absent, so there is nothing for a forged request to populate.

## 2. Canonical Character source (kernel §10 "Preferred canonical Character source: Show Run roster-selected Character")

`resolveSpeakerCharacter` in `ic_chat.go` resolves the speaker exactly the way `world/snapshot.go`'s `resolveTheaterContext` already answers "has this Player chosen a Character" (Kernel 71's canonical signal, duplicated here rather than imported to keep `actions` from depending on the much larger `world` package for one lookup — same avoidance pattern `rollaudience` documents for its own cohort-membership duplication):

```sql
SELECT rm.character_card_id, cc.name, cc.portrait_url, cc.is_deleted
FROM show_run_roster_members rm
LEFT JOIN character_cards cc ON cc.id = rm.character_card_id
WHERE rm.show_run_id = $1 AND rm.user_id = $2 AND rm.removed_at IS NULL
```

resolved via `sessions.show_id → shows.show_run_id → show_run_roster_members` (the same chain, same table, same nullable/`is_deleted` fallback rules as Kernel 71's roster Character selection). Session persona (`current_session_personas`) and site-wide active Character were **not** used as the source, per the kernel's explicit instruction not to revive session persona as authority; they remain unrelated concepts.

## 3. No forged Character is possible

Three independent facts make impersonation structurally impossible, not just policy-forbidden:

1. The request type carries no Character field (§1).
2. Even a raw JSON payload with an extra `character_id` key is silently ignored — Go's `json.Decode` into a struct without that field simply drops unknown keys; nothing in `StoreICChatMessage` ever reads a Character ID from the request. Proven in `TestICChat_ForgedCharacterFieldIsIgnored` (backend) and the browser proof's "extra client-supplied character_id field is ignored" step (a raw `/api/commands/execute` call with a `character_id` field present in the body — the stored action still shows the real, server-resolved Character).
3. The WS path (`network/ws.go`'s `chat/ic_message` case) reads only `session_id`/`text` out of the inbound payload before calling `StoreICChatMessage` — the same non-existence-of-a-field guarantee applies to the live-socket send path, not just the HTTP command path.

## 4. Accountable user is always stored

`actions.actor_id` is the authenticated user, always — the same column and the same non-optional guarantee every other action type in this durable event log already has. The Character is *presentation* (who the message is attributed to on screen); the user is *accountability* (who is answerable for it). Both are stored on the same row (`payload.character_id`/`character_name`/`character_portrait` alongside the ordinary `actor_id`/`actor_display_name`/`actor_handle`), never conflated.

## 5. Character switch affects only future messages

`payload.character_name`/`payload.character_id`/`payload.character_portrait` are stamped into the `actions` row **at send time** — a later `UPDATE show_run_roster_members SET character_card_id = ...` (switching Characters) has no effect on any already-stored row's `payload`. Proven in `TestICChat_CharacterSwitchAffectsFutureMessagesOnly`: two messages sent around a Character switch retain their respective speaker identities when the first is re-read fresh from the database afterward. The frontend renders `payload.character_name` directly (`appendChatActionLine` in `runtime.js`), never the sender's *current* Character, so old messages visually keep the right speaker even in a long-lived chat log.

## 6. No Character selected fails cleanly

`ErrNoCharacterSelected` (a named, exported sentinel error) is returned whenever the roster row is missing, has no `character_card_id`, or that Character `is_deleted`. Both entry points surface it as a clean, distinguishable failure:

- HTTP command path (`/ic`): `error.Error()` is the literal string `no_character_selected`, mapped client-side to *"Choose a Character first (Show Run roster) to speak In Character."* (`commandErrorMessages` in `runtime.js`).
- Proven live in the browser proof: a fresh, unrostered account attempting `/ic` gets a clean `403`/`no_character_selected` (wrapped through `not_session_participant` first if not even joined — both are equally clean, non-crashing refusals, never a silent send).

No send is ever silently dropped or silently attributed to a fallback identity.

## 7. Distinct from OOC (kernel §10 "Existing OOC/venue chat remains separate")

IC chat is a new, sibling action type (`chat/ic_message`, alongside the pre-existing `chat/message` and `chat/ooc`), a new command (`/ic`, alongside `/ooc`), and a new tab (`#chat-ic-tab`/`#chat-ic-panel`/`#chat-ic-log`, alongside `#chat-ooc-tab`). Nothing about OOC's storage, authority, or rendering changed. Both tabs share the same composer input box and the same `/command`-based send mechanism the existing OOC tab already used (there is no per-tab-active-state dispatch in this codebase prior to Kernel 87 — OOC was already slash-command-driven, not "type in the OOC tab and it's OOC," so IC follows that exact precedent rather than introducing a new send mechanism).

## 8. Display (kernel §10 "Display Character name and Face/avatar where current chat supports it")

The IC log renders the Character's name as the speaker label (`appendChatActionLine`'s `isIC` branch). `character_portrait` is captured in the payload but **not yet rendered as an avatar image** in the chat log UI — a real, acknowledged gap (the data is there; the `<img>` wiring in the chat entry template is not), recorded here rather than silently left unclear.

## 9. Known gaps

- No avatar image render in the IC log (§8).
- IC chat is not Cohort-scoped — it is Show/session-wide, matching existing OOC/main chat's own scope. The kernel spec does not ask for Cohort-scoped IC chat; recorded here only so a future kernel doesn't have to re-derive that this was a deliberate read of the spec, not an oversight.
