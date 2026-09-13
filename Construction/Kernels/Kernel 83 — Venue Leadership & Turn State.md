# Kernel 83 — Venue Leadership & Turn State

**Status:** READY FOR IMPLEMENTATION  
**Type:** Bounded platform coordination kernel  
**Primary track:** Victory Core  
**Sequence position:** After Kernel 82  
**Primary surface:** Presence Tray / collaborative venues  
**Feature scope:** Group Leader + Current Turn

---

## 0. Kernel contract

Kernel 83 adds two reusable, lightweight coordination signals to Victory venues where people actively work or play together:

- **Group Leader**
- **Current Turn**

These are **live venue-session coordination states**.

They are not Victory roles, permission grants, campaign state, durable Storyboard state, participant-order systems, or automatic turn engines.

The required vertical is:

```text
Collaborative venue opts into leadership/turn state
→ venue session begins
→ Group Leader initializes to venue owner when available
→ Current Turn begins unset
→ Presence Tray visibly marks Group Leader and Current Turn
→ authorized users right-click a participant
→ choose Make Group Leader and/or Give Turn when allowed
→ state updates server-authoritatively
→ connected clients receive the change live
→ disconnecting does not auto-reassign either state
→ ending the live venue session clears both states
→ a later session starts fresh
```

Kernel 83 must be reusable by Storyboards Timeline and future collaborative venues without making Storyboards the owner of the logic.

---

# 1. Locked product decisions

## 1.1 Scope

Leadership/turn state exists **per collaborative venue session**.

Not globally across Victory, not per account, not per Production, not per arbitrary surface inside a venue, and not permanently on a Storyboard.

A venue that supports this capability has one current:

```text
group_leader_user_id
current_turn_user_id
```

for the active live session.

## 1.2 Opt-in capability

Do not inject this feature into every venue automatically.

Collaborative venues opt into the capability.

The implementation should provide a generic capability/contract that venues can enable.

Storyboards should be an initial consumer because Kernel 82 already exposes the integration seam.

Future venues may opt in without duplicating the state model.

## 1.3 Group Leader initialization

When a collaborative venue session begins:

- Group Leader defaults to the venue owner when an owner is meaningful and available;
- if the owner is not present or the venue has no applicable owner, Group Leader may begin unset;
- do not automatically elect another participant.

Do not invent a fallback hierarchy.

## 1.4 Current Turn initialization

Current Turn begins unset by default.

Do not automatically assign first turn to Group Leader, owner, first participant, highest role, or first person entering.

The room may explicitly assign it.

## 1.5 No persistence across sessions

Group Leader and Current Turn are ephemeral live-session state.

When the collaborative venue session ends, both are cleared.

A later session begins fresh.

Do not serialize either field into Storyboard persistence, Timeline export, campaign save state, user profile, or Production durable state.

Kernel 82's Timeline integration seam must consume live session state only.

## 1.6 No participant order

Do not implement participant order.

Therefore do not implement Next Turn, Previous Turn, automatic rotation, numbered turn-order badges, reorder participant controls, move earlier/later, or 'end turn' that infers the next player.

Turn changes happen only through explicit handoff.

## 1.7 Explicit handoff only

Current Turn changes because an authorized person explicitly selects a target participant.

Example:

```text
right-click Bob
→ Give Turn
```

No software-inferred next participant.

## 1.8 Disconnect behavior

If the Group Leader disconnects:

- keep Group Leader assigned for the live venue session;
- visibly mark them absent if Presence already supports absence;
- do not auto-reassign.

If the Current Turn holder disconnects:

- keep Current Turn assigned;
- mark absent if supported;
- do not auto-advance.

Authorized users may explicitly assign someone else.

Victory does not make a gameplay decision because a person lost connection.

## 1.9 Permissions remain independent

Group Leader is not a Victory role.

Current Turn is not a Victory role.

Neither state changes read permissions, write permissions, structural permissions, Director authority, Crew authority, Cast/Audience authority, ownership, or sharing.

Being Group Leader must not make someone Director.

Having Current Turn must not lock everyone else out.

These are coordination signals only.

---

# 2. Authority model

## 2.1 Who may assign Group Leader

The following may assign Group Leader to a participant in the same collaborative venue session:

- Owner / Director+ according to the venue's established authority model;
- the current Group Leader, who may pass leadership.

Crew/Cast/Audience may not seize leadership merely by right-clicking themselves.

## 2.2 Who may assign Current Turn

The following may assign Current Turn:

- Owner / Director+;
- current Group Leader;
- current Current Turn holder, who may pass the turn.

No one else may arbitrarily assign Current Turn unless an existing higher authority model already grants equivalent Director+ control.

## 2.3 Target requirements

Assignment target should normally be a participant known to the current collaborative venue session and preferably currently present.

Do not allow assigning arbitrary account IDs.

If the product already distinguishes session participants from transient Presence entries, use the narrower venue-participant concept.

## 2.4 Self-assignment

Do not create special 'Take Leadership' or 'Take Turn' actions.

If an authorized person is allowed to assign the state, they may target themselves through the same generic action.

Example:

```text
Director right-clicks self
→ Make Group Leader
```

No additional seizure mechanic is needed.

---

# 3. Presence Tray interaction

## 3.1 Primary interaction

The Presence Tray right-click menu is the primary control surface.

Right-clicking a participant should conditionally expose:

```text
Make Group Leader
Give Turn
```

Only show actions the acting user is authorized to perform.

## 3.2 Wording

Use concise generic wording:

- **Make Group Leader**
- **Give Turn**

Do not create redundant variants such as Pass Leadership To, Set Leader, Transfer Leadership, Pass Turn To, or Set Current Player.

The action target is already the participant being right-clicked.

## 3.3 Current-state handling

If the target is already Group Leader, hide or disable Make Group Leader.

If the target already has Current Turn, hide or disable Give Turn.

Avoid no-op mutations.

## 3.4 Visible indicators

Presence Tray must clearly show who currently has each state.

The exact visual treatment may follow existing tray conventions, but users must be able to distinguish Group Leader, Current Turn, and someone holding both.

Examples may include a small badge/icon, label, border marker, or compact status chip.

Do not reorder the Presence Tray based on these states.

## 3.5 No additional dedicated panel in Kernel 83

Do not build a turn-order window, session-management drawer, leader-control modal, or participant-order panel.

Right-click plus visible indicators is sufficient for this kernel.

A future discoverability control may be added later if testing shows right-click is too hidden.

---

# 4. Live venue-session state model

## 4.1 State identity

State should attach to a stable live collaborative venue-session identifier.

Conceptually:

```text
venue_session_id
venue_type
venue_instance_id
group_leader_user_id nullable
current_turn_user_id nullable
```

Use repository-consistent session/location identifiers.

Do not invent a duplicate session concept if Victory already has an active venue/session identity.

## 4.2 Ephemeral persistence

This state may be stored in memory, a live-session table, existing venue-presence/session infrastructure, or another ephemeral server-authoritative store.

The key product rule is lifecycle, not storage technology:

```text
active collaborative venue session exists
→ state may exist

session ends
→ state cleared
```

Do not retain it as durable game state.

## 4.3 Server authority

All mutations are server-authoritative.

Client requests conceptually look like:

```text
assign_group_leader(target_user_id)
assign_current_turn(target_user_id)
```

Server verifies:

- acting user is authenticated;
- acting user belongs to the venue session;
- target belongs to the venue session;
- venue opted into the capability;
- actor has authority;
- session is active.

Then the server mutates state and broadcasts the resulting canonical state.

## 4.4 Live sync

Changes must propagate to all connected participants in that collaborative venue session.

Reuse Victory's existing WebSocket/live event infrastructure.

Do not create a polling loop if live venue events already exist.

## 4.5 Idempotency

Assigning the same target repeatedly should be harmless.

Prefer a no-op response or same canonical state.

Do not produce duplicate event storms.

---

# 5. Session lifecycle

## 5.1 Session start

When the collaborative venue session is created/activated:

1. determine whether the venue has opted into leadership/turn capability;
2. initialize Group Leader to the venue owner if applicable and available;
3. initialize Current Turn to null;
4. publish initial state to connected clients.

## 5.2 Participants joining

Joining later does not automatically change either state.

If Group Leader was unset, a later owner join should not silently seize leadership unless the implementation's existing session-start concept means the session has not yet actually begun.

Prefer explicit assignment after initialization.

## 5.3 Participants leaving

Leaving does not clear their assignment while the venue session remains active.

Presence may indicate absence.

No automatic reassignment.

## 5.4 Session end

When the collaborative venue session actually ends:

- clear Group Leader;
- clear Current Turn;
- remove/expire the ephemeral state record.

Define 'session end' using existing Victory venue-session lifecycle, not merely 'one browser tab closed.'

If Victory currently lacks an explicit collaborative session lifecycle, investigate the narrowest existing lifecycle primitive and document the chosen mapping.

Do not create durable persistence as a shortcut.

---

# 6. Collaborative venue opt-in contract

## 6.1 Generic capability

Provide a simple reusable opt-in mechanism.

Possible shapes:

```go
SupportsLeadershipTurnState bool
```

or:

```text
venue capability:
coordination_state = enabled
```

Exact implementation should follow existing venue registry patterns.

## 6.2 Storyboards integration

Storyboards should opt in during Kernel 83.

Timeline's Kernel 82 integration seam should display live state if it already has a natural place to consume it.

Do not save live state into the Timeline board.

Blank Storyboards may also receive the same venue-session capability if Storyboards is treated as one collaborative venue family.

Use repository architecture to choose the cleanest scope.

## 6.3 Other venues

Do not bulk-enable every venue.

Do not refactor unrelated venues merely to prove reusability.

Document how a future venue opts in.

---

# 7. Timeline integration

Kernel 82 documented an optional future session-state slot/provider.

Kernel 83 should wire that seam to the generic live session state.

Expected Timeline behavior when active:

```text
Group Leader: Grant
Current Turn: Alice
```

or equivalent compact presentation if the panel seam was implemented as a hidden slot.

Requirements:

- read-only display in Timeline;
- Presence Tray remains the mutation surface;
- Timeline does not own the state;
- Timeline export does not include it;
- reopening a later session does not restore it.

If the Kernel 82 seam was deliberately documentation-only rather than rendered UI, do not expand 83 into a Timeline redesign. A minimal read-only integration is sufficient.

---

# 8. API / events

Use repository conventions.

Possible mutation routes:

```text
POST /api/venues/{venueSessionID}/coordination/group-leader
POST /api/venues/{venueSessionID}/coordination/current-turn
```

with:

```json
{"target_user_id":"..."}
```

Possible read surface:

```text
GET /api/venues/{venueSessionID}/coordination
```

Possible canonical state:

```json
{
  "venue_session_id": "...",
  "group_leader_user_id": "...",
  "current_turn_user_id": "..."
}
```

Do not treat these exact route names as mandatory if Victory already has a better venue-session API pattern.

Live event examples:

```text
venue_coordination_changed
```

or:

```text
group_leader_changed
current_turn_changed
```

Prefer the minimum event shape consistent with existing event handling.

---

# 9. Security and authorization

## 9.1 Authentication

All mutations require authenticated session.

## 9.2 Venue membership

Actor and target must be valid participants in the relevant collaborative venue session.

Do not accept cross-venue participant assignment.

## 9.3 Capability check

Reject mutations when the venue has not opted into leadership/turn state.

## 9.4 Authority check

Server enforces the authority rules from §2.

Never rely on hidden right-click options alone.

## 9.5 No privilege escalation

Add explicit tests proving:

- Group Leader does not gain Director permission;
- Current Turn holder does not gain write permission;
- assigning either state does not modify role rows/grants;
- a Cast/Audience user who becomes Current Turn still has only their normal permissions;
- a Crew user who becomes Group Leader still has only Crew permissions unless independently granted more.

This criterion is critical.

---

# 10. Presence Tray frontend behavior

## 10.1 Context menu

Reuse the existing Presence Tray context-menu system if one exists.

Do not create a separate participant-list implementation.

On right click:

1. determine current actor authority from server-provided/canonical data;
2. render eligible actions;
3. send assignment mutation;
4. update from server response/live event.

## 10.2 Indicators

Indicators should update live without page refresh.

Need to handle:

- leader only;
- turn only;
- same user holds both;
- assigned user disconnects;
- state cleared at session end.

## 10.3 Accessibility fallback

Right-click is the required product interaction for Kernel 83.

Do not expand this kernel solely to build an alternate interaction surface.

However, keep context-menu actions represented as real focusable menu items where the existing tray architecture permits, and do not block a future keyboard/touch trigger.

Log any right-click-only accessibility limitation rather than inflating scope.

---

# 11. Required backend tests

## 11.1 Initialization

- opted-in venue session initializes Group Leader to applicable owner;
- Current Turn initializes null;
- venue without capability gets no coordination state;
- owner absent/no applicable owner leaves Group Leader null rather than electing another user.

## 11.2 Group Leader authority

- Director+ assigns leader;
- current leader passes leader;
- Crew cannot seize leader;
- Cast cannot seize leader;
- Audience cannot seize leader;
- target must belong to venue session;
- cross-venue target rejected.

## 11.3 Current Turn authority

- Director+ gives turn;
- Group Leader gives turn;
- current turn holder passes turn;
- unrelated Crew/Cast/Audience cannot assign turn;
- target must belong to venue session.

## 11.4 No participant order

Prove there is no automatic next-player behavior:

- assigning turn to Alice;
- Alice disconnects;
- Current Turn remains Alice;
- Bob is not auto-assigned.

## 11.5 Disconnect

- leader disconnect leaves assignment intact;
- current-turn holder disconnect leaves assignment intact;
- reconnect restores visible present state with same assignment during same live session.

## 11.6 Session end

- ending venue session clears leader;
- ending venue session clears current turn;
- new later venue session does not restore prior assignments.

## 11.7 Permission independence

- assigning leader leaves Victory role unchanged;
- assigning turn leaves Victory role unchanged;
- leader cannot perform Director action unless independently Director+;
- current-turn holder cannot perform Crew action unless independently Crew+.

## 11.8 Live sync

- leader change broadcasts;
- turn change broadcasts;
- unrelated venue sessions do not receive event;
- repeat same assignment is idempotent/no duplicate mutation harm.

---

# 12. Required browser proof

Playwright proof is mandatory.

Use disposable users and a collaborative Storyboards session.

Prove:

1. Storyboards opts into coordination state.
2. Owner begins as Group Leader when applicable.
3. Current Turn begins unset.
4. Presence Tray marks Group Leader.
5. Director+ right-clicks another participant and sees Make Group Leader.
6. Director+ assigns new Group Leader.
7. all connected clients update live.
8. previous leader no longer marked.
9. Group Leader right-clicks participant and sees Make Group Leader.
10. unauthorized participant does not see/cannot successfully invoke leader assignment.
11. Director+ right-clicks participant and sees Give Turn.
12. Group Leader can Give Turn.
13. Current Turn holder can Give Turn.
14. unauthorized unrelated participant cannot change turn.
15. Presence Tray marks Current Turn.
16. same participant can visibly hold both states.
17. changing Current Turn does not reorder Presence Tray.
18. current-turn holder disconnects; state remains assigned and absent/presence status is truthful.
19. leader disconnects; state remains assigned.
20. session ends and state clears.
21. new session starts fresh.
22. becoming Group Leader does not expose Director-only Storyboard controls.
23. becoming Current Turn does not expose Crew-only Storyboard controls.
24. Timeline live-state seam displays values if implemented.
25. Timeline JSON export does not contain leader/turn state.

No PASS based only on API tests.

---

# 13. Required artifacts

```text
Construction/Kernels/Kernel 83 — Venue Leadership & Turn State.md
Construction/Domains/Venues/collaborative-venue-coordination-contract.md
Construction/Domains/Venues/presence-tray-coordination-actions.md
Construction/Domains/Venues/venue-session-state-lifecycle.md
Construction/Domains/Storyboards/session-state-integration-seam.md
Construction/OperatorLogs/kernel-83-reportback.md
```

Update existing docs rather than duplicating them where appropriate.

---

# 14. Required evidence

## Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}	{{.Status}}	{{.Ports}}'
```

## Backend

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
```

Use isolated `TEST_DATABASE_URL`.

## Frontend

Run Node syntax checks, existing frontend logic tests, and Playwright live browser proof.

## Static

```bash
git diff --check
```

---

# 15. Pass criteria

Kernel 83 passes when:

- collaborative venues can opt into generic leadership/turn state;
- Storyboards is wired as an initial consumer;
- Group Leader exists per active collaborative venue session;
- Current Turn exists per active collaborative venue session;
- owner initializes as Group Leader when applicable;
- Current Turn starts unset;
- Director+ may assign Group Leader;
- current Group Leader may pass leadership;
- Director+ may assign Current Turn;
- Group Leader may assign Current Turn;
- current Current Turn holder may pass turn;
- turn handoff is explicit only;
- no participant order exists;
- no automatic rotation exists;
- disconnect does not auto-reassign either state;
- session end clears both states;
- later sessions start fresh;
- Presence Tray right-click provides authorized actions;
- Presence Tray visibly indicates both states;
- Presence Tray ordering is unaffected;
- live changes synchronize to connected clients;
- server enforces authority;
- neither state changes Victory permissions;
- Timeline consumes live state through the Kernel 82 seam without persisting it;
- Timeline export excludes live coordination state;
- no unrelated venue regression occurs;
- Playwright proof passes.

---

# 16. Partial and fail rules

## PARTIAL

Use when:

- backend state works but Presence Tray integration is incomplete;
- live sync requires refresh;
- Group Leader works but Current Turn does not;
- session cleanup is unreliable;
- Timeline seam is not wired but generic venue capability is otherwise complete.

## FAIL

Use when:

- leader/turn state persists into later sessions contrary to contract;
- disconnect automatically chooses another participant;
- software creates participant order;
- state assignment changes actual Victory roles/permissions;
- unauthorized users can assign leader/turn;
- Presence Tray is reordered by turn state;
- Storyboards owns a duplicated special-case state instead of generic venue capability;
- live state leaks into Timeline export;
- users are locked out or existing permissions regress.

---

# 17. Non-goals

Kernel 83 does not:

- implement participant order;
- implement Next Turn;
- implement Previous Turn;
- implement automatic turn rotation;
- implement round tracking;
- implement initiative;
- implement timers;
- implement turn locks;
- disable controls when it is not someone's turn;
- grant permissions to Group Leader;
- persist leadership across sessions;
- persist Current Turn across sessions;
- create a dedicated turn-order panel;
- create a session-management drawer;
- redesign Presence Tray ordering;
- bulk-enable every venue;
- create game-specific turn rules.

---

# 18. Kernel-size guardrail

Kernel 83 should remain a small platform primitive.

Target implementation shape:

- one generic collaborative-venue capability flag/contract;
- one live venue-session coordination state model;
- two mutation paths;
- one live-event shape;
- Presence Tray context actions + indicators;
- Storyboards/Timeline integration;
- one complete multiplayer browser journey.

Do not expand into initiative tracker, turn-order manager, session host tools, role delegation, Presence Tray redesign, or persistent campaign state.

---

# 19. Immediate operator outcome

At completion, Grant must be able to answer:

1. When I open a collaborative Storyboards session, can I see who the Group Leader is?
2. Does the owner begin as Group Leader when appropriate?
3. Does Current Turn begin unset?
4. Can I right-click someone and make them Group Leader when I have authority?
5. Can the current Group Leader pass leadership the same way?
6. Can I right-click someone and give them the turn when I have authority?
7. Can the current turn holder explicitly give the turn to another person?
8. Does Victory avoid inventing a next player?
9. Does the Presence Tray stay in its normal order?
10. If the leader leaves, does Victory refrain from electing someone else?
11. If the current turn holder leaves, does Victory refrain from advancing?
12. When they reconnect during the same session, is the assignment still there?
13. When the whole live venue session ends, is the state gone?
14. When we return later, do we start fresh?
15. Can someone hold both Group Leader and Current Turn visibly?
16. Does becoming Group Leader leave their actual Victory role unchanged?
17. Does having Current Turn leave their permissions unchanged?
18. Do all connected participants see changes live?
19. Does Timeline display the live state without saving it into the board?
20. Does Timeline export remain free of ephemeral coordination state?

Kernel 83 succeeds when Victory can tell a collaborative room **who is leading and whose turn it is** without becoming a rules engine, permission system, or turn-order manager.
