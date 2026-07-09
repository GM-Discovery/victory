# Kernel 62 — Player Relationship Matrix, Private Notes, and Relationship Journals

**Revision:** 0.1
**Status:** READY FOR BUILD
**Primary track:** V — Player identity and social-profile layer
**Complementary tracks:** O — privacy and accountable identity; C — concierge/private-community social tooling; S — Socio player continuity
**Kernel type:** social-private, workbook, projection, privacy, frontend, backend, verification
**Depends on:** Kernel 61A PASS; Player Workbook; Trailer Face; cross-user Trailer view link; targeted `/ws/player-profile` profile invalidation; stage-name ledger; stable account UUIDs; social Face route; account/session identity
**Expected next consumer:** a future Third Place / Faceprint Commons kernel, Show Run roster primitive, or production-scoped social discovery kernel

---

## 1. Objective

Build the private player-to-player relationship layer.

The completed loop is:

```text
I view another player's Trailer
→ I add them to My People
→ Victory creates a private directional relationship record
→ I record how I know them, private categories, qualitative relationship fields, notes, and journal entries
→ their stage name may change, but my record follows the same stable account
→ I can archive the relationship without affecting them
→ they cannot see that the record exists
```

Kernel 62 is not a public social network. It is a private relationship workbook for remembering real people inside a Victory installation.

---

## 2. Product boundary

### 2.1 What Kernel 62 is

Kernel 62 creates:

```text
My People
A private list of real users I am tracking.

Relationship Workbook
My private record about one real user.

Relationship Summary
A private, compiled "how I know this person" view.

Private Journal
Dated notes about this person.

Follow-Up Items
Stored private intentions or reminders without scheduled notification.
```

### 2.2 What Kernel 62 is not

Do not build:

- friend requests;
- mutual connections;
- public friend lists;
- followers;
- likes;
- comments;
- feeds;
- direct messages;
- impressions;
- autographs;
- tags visible to the subject;
- Third Place;
- Faceprint Commons;
- show/run rosters;
- cross-server federation;
- blocking or moderation;
- `/ban` or `/boot`;
- AI summaries;
- automatic relationship scoring;
- character relationships;
- notifications or scheduled reminders.

Those belong to later kernels if needed.

### 2.3 Core privacy rule

The subject does **not** know the relationship record exists.

They cannot read it, query it, infer it from an endpoint, or see a count of how many people have records about them.

---

## 3. Kernel 61A contracts to reuse

Do not invent parallel identity or profile systems.

Reuse:

- stable account identity for ownership and authority;
- Player Workbook / Trailer Face distinction;
- social Face route and profile identifier;
- stage-name update behavior;
- profile websocket invalidation where relevant;
- account-private data exclusion rules;
- existing authentication/session helpers;
- existing privacy discipline from Kernel 61A.

Assume the following are available because Kernel 61A passed:

```text
A real user has one Trailer / Player Workbook.
Another authenticated user can open a copied Trailer Face link.
The social Face exposes only compiled public-in-installation presentation.
Stage-name changes do not change the stable account identity.
Private workbook/account fields are not exposed in social Face.
```

---

## 4. Resolved owner decisions

These decisions are fixed.

### 4.1 Private and directional

A relationship record is directional:

```text
observer_user_id
subject_user_id
```

Example:

```text
Straturli → Alex
```

does not imply:

```text
Alex → Straturli
```

Each user's relationship workbook is separate and private.

### 4.2 Subject invisibility

The subject is not notified.

The subject cannot see:

- that a record exists;
- private nickname;
- categories;
- qualitative fields;
- notes;
- journal entries;
- follow-ups;
- archive state;
- last-viewed or last-edited metadata.

### 4.3 Qualitative fields use words

Use named dropdown values, not visible numeric scores.

The backend may store sort order internally, but the user sees labels.

### 4.4 Relationship categories use checkboxes

Multiple categories may apply.

No category grants access rights or implies mutual agreement.

### 4.5 Follow-ups are stored only

Follow-up items are private stored records.

Do not create reminders, notifications, scheduler tasks, calendar events, emails, or websocket alerts for due follow-ups.

### 4.6 Archive, do not delete

Kernel 62 supports archiving a relationship.

It does not need destructive relationship deletion.

Ordinary private journal entries may be edited or deleted if implemented safely, but the relationship itself uses archive/unarchive.

### 4.7 Private nickname

The observer may assign a private nickname or memory label for the subject.

The current stage name remains visible alongside it.

---

## 5. Canonical domain model

### 5.1 Relationship record

Recommended table:

```text
player_relationships
- id UUID primary key
- observer_user_id UUID not null references users(id)
- subject_user_id UUID not null references users(id)
- private_nickname TEXT not null default ''
- relationship_state TEXT not null default 'active'
- trust_level TEXT not null default 'unknown'
- closeness_level TEXT not null default 'unknown'
- reliability_level TEXT not null default 'unknown'
- communication_ease_level TEXT not null default 'unknown'
- archived_at TIMESTAMPTZ nullable
- created_at TIMESTAMPTZ not null default now()
- updated_at TIMESTAMPTZ not null default now()
- unique(observer_user_id, subject_user_id)
```

Rules:

- `observer_user_id` cannot equal `subject_user_id`;
- only observer may read or mutate;
- subject never receives the row through normal endpoints;
- record follows `subject_user_id` through stage-name changes;
- archiving sets `archived_at`, does not delete;
- unarchiving clears `archived_at`.

### 5.2 Relationship categories

Recommended table:

```text
player_relationship_categories
- relationship_id UUID not null
- category_key TEXT not null
- custom_label TEXT not null default ''
- created_at TIMESTAMPTZ not null default now()
- primary key (relationship_id, category_key, custom_label)
```

Initial checkbox categories:

```text
acquaintance
friend
close_friend
family
collaborator
coworker
player
gm_facilitator
mentor
mentee
client
community_contact
creative_partner
professional_contact
custom
```

Rules:

- multiple categories allowed;
- custom category requires non-empty `custom_label`;
- categories are private labels only;
- categories do not affect authority, memberships, invitations, or visibility.

### 5.3 Relationship facts

For extensibility, relationship pages should produce private facts rather than one blob.

Recommended table:

```text
player_relationship_facts
- id UUID primary key
- relationship_id UUID not null
- field_key TEXT not null
- value_json JSONB not null
- display_value TEXT not null
- source_event_id UUID nullable
- updated_at TIMESTAMPTZ not null default now()
- unique(relationship_id, field_key)
```

Fields are private to the observer.

### 5.4 Relationship events

Recommended table:

```text
player_relationship_events
- id UUID primary key
- relationship_id UUID not null
- event_type TEXT not null
- page_key TEXT not null
- payload JSONB not null
- human_summary TEXT not null
- created_by_user_id UUID not null
- created_at TIMESTAMPTZ not null default now()
- deleted_at TIMESTAMPTZ nullable
```

Rules:

- events are observer-private;
- subject cannot read them;
- ordinary events may be soft-deleted or hard-deleted according to project convention;
- deletion recomputes current relationship facts;
- stage-name history remains account-level from Kernel 61A and is not duplicated here.

### 5.5 Relationship journal

Recommended table:

```text
player_relationship_journal_entries
- id UUID primary key
- relationship_id UUID not null
- title TEXT not null default ''
- body TEXT not null
- entry_date DATE nullable
- note_category TEXT not null default 'general'
- tags TEXT[] not null default '{}'
- production_id UUID nullable
- session_id UUID nullable
- created_by_user_id UUID not null
- created_at TIMESTAMPTZ not null default now()
- updated_at TIMESTAMPTZ not null default now()
- deleted_at TIMESTAMPTZ nullable
```

Rules:

- private to observer;
- body never appears in subject Trailer;
- optional context links must be authority-checked;
- no automatic sharing;
- no notifications.

### 5.6 Follow-up items

Recommended table:

```text
player_relationship_followups
- id UUID primary key
- relationship_id UUID not null
- title TEXT not null
- notes TEXT not null default ''
- target_date DATE nullable
- status TEXT not null default 'open'
- created_at TIMESTAMPTZ not null default now()
- updated_at TIMESTAMPTZ not null default now()
- completed_at TIMESTAMPTZ nullable
```

Allowed statuses:

```text
open
done
dismissed
```

Rules:

- no scheduler;
- no reminder delivery;
- no email;
- no notification;
- no calendar integration.

### 5.7 Shared context projection

Create a read-only derived projection, not manually edited facts.

Useful automatic shared context may include only reliable existing data:

- shared productions both users belong to;
- each person's roles in those productions;
- shared scoped venues or access contexts if meaningful;
- first known membership overlap where determinable;
- manually entered "how we met" fact.

Do not infer:

- trust;
- closeness;
- friendship;
- game attendance unless a reliable attendance model exists;
- chat frequency;
- emotional meaning;
- "you may know this person."

---

## 6. Qualitative vocabularies

Use small, named sets.

### 6.1 Trust

Keys:

```text
unknown
cautious
developing
trusted
deeply_trusted
```

Labels:

```text
Unknown
Cautious
Developing
Trusted
Deeply Trusted
```

### 6.2 Closeness

Keys:

```text
unknown
distant
familiar
friendly
close
core_relationship
```

Labels:

```text
Unknown
Distant
Familiar
Friendly
Close
Core Relationship
```

### 6.3 Reliability

Keys:

```text
unknown
inconsistent
usually_reliable
reliable
highly_reliable
```

Labels:

```text
Unknown
Inconsistent
Usually Reliable
Reliable
Highly Reliable
```

### 6.4 Communication ease

Keys:

```text
unknown
difficult
uneven
workable
easy
very_easy
```

Labels:

```text
Unknown
Difficult
Uneven
Workable
Easy
Very Easy
```

### 6.5 Relationship state

Keys:

```text
active
quiet
strained
rebuilding
archived
```

Labels:

```text
Active
Quiet
Strained
Rebuilding
Archived
```

The visible UI must show labels, not numeric values.

---

## 7. Relationship Workbook pages

Build a private multi-page relationship workbook.

### 7.1 Connection

Fields:

- private nickname;
- how I know them;
- where we met;
- first meaningful context;
- relationship categories;
- shared contexts;
- what name I know them by;
- freeform connection notes.

### 7.2 Understanding Them

Fields:

- interests;
- strengths;
- things they care about;
- communication preferences;
- things to remember;
- practical boundaries;
- what seems to help collaboration.

### 7.3 Our Relationship

Fields:

- trust dropdown;
- closeness dropdown;
- reliability dropdown;
- communication ease dropdown;
- relationship state dropdown;
- what I value about this relationship;
- unresolved matters;
- current caution or care notes.

Do not label these as objective truth. They are the observer's private working notes.

### 7.4 Shared Work and Play

Fields:

- productions;
- campaigns;
- projects;
- roles held together;
- systems played together, manually entered;
- collaboration history;
- things we said we might do.

Automatic shared context may be displayed beside this page but must remain clearly marked as "Victory can currently verify."

### 7.5 Follow-Up

Fields:

- open follow-up items;
- target date;
- completion;
- dismissed items;
- notes.

No notifications.

### 7.6 Journal

Features:

- add journal entry;
- edit own journal entry;
- delete own journal entry;
- filter by category;
- filter by tag;
- optional date;
- optional production context;
- optional session context only if reliable session IDs exist.

---

## 8. User experience

### 8.1 Entry points

Minimum required entry points:

1. From another user's Trailer Face:
   - button: `Add to My People` if no relationship exists;
   - button: `Open My Notes` if relationship exists.

2. From My Trailer or Account:
   - link to `My People`.

3. Direct URL:
   - `/people/` or `/venues/trailers/people.html`;
   - exact path may follow repository conventions.

Do not build global discovery.

### 8.2 My People list

The list must show:

- current stage name;
- portrait thumbnail from subject's Trailer Face when available;
- private nickname when set;
- categories;
- trust/closeness/reliability/communication summary;
- last journal entry date;
- open follow-up count;
- shared production count if available;
- archived badge/state.

Required controls:

- search by stage name and private nickname;
- filter by category;
- filter active/archived;
- sort by stage name;
- sort by last updated;
- sort by open follow-ups.

Do not show follower counts, popularity, public tags, or mutuality.

### 8.3 Person view

The relationship detail page must include:

- subject's current social Face header;
- clear indication that notes are private to the observer;
- private nickname;
- relationship categories;
- qualitative dropdown fields;
- shared context panel;
- workbook pages;
- journal;
- follow-up items;
- archive/unarchive.

Stage-name changes from Kernel 61A should update displayed current stage name after reload and ideally via profile invalidation if already watching the subject profile.

### 8.4 Privacy copy

Use plain language:

```text
Only you can see these notes.
They are not shared with this person.
```

Do not overstate database/operator privacy. The application must not expose ordinary UI/API access to another user, but self-hosted operators can technically inspect a database.

### 8.5 Empty states

For a user with no relationships:

```text
My People is empty.
Open someone's Trailer and choose Add to My People.
```

For a relationship with no notes:

```text
You have not written notes about this person yet.
```

For archived relationships:

```text
Archived relationships are hidden from your default list but can be restored.
```

---

## 9. Backend operations

Implement focused operations.

### 9.1 Relationship root

- `EnsureRelationship(observerUserID, subjectProfileID or subjectUserID)`
- `GetRelationship(observerUserID, relationshipID)`
- `GetRelationshipBySubject(observerUserID, subjectProfileID)`
- `ArchiveRelationship(observerUserID, relationshipID)`
- `UnarchiveRelationship(observerUserID, relationshipID)`
- `ListRelationships(observerUserID, filters)`

Rules:

- authenticated observer comes from session;
- client-supplied observer ID is ignored/rejected;
- subject is resolved from public profile/workbook ID without exposing account UUID;
- observer cannot create relationship with self;
- archived records are excluded by default.

### 9.2 Workbook mutations

- `SaveRelationshipPage(observerUserID, relationshipID, pageKey, answers)`
- `DeleteRelationshipEvent(observerUserID, relationshipID, eventID)`
- `RecomputeRelationshipFacts(relationshipID, affectedFields)`

### 9.3 Journal operations

- `CreateRelationshipJournalEntry(observerUserID, relationshipID, input)`
- `UpdateRelationshipJournalEntry(observerUserID, relationshipID, entryID, input)`
- `DeleteRelationshipJournalEntry(observerUserID, relationshipID, entryID)`
- `ListRelationshipJournalEntries(observerUserID, relationshipID, filters)`

### 9.4 Follow-up operations

- `CreateFollowUp(observerUserID, relationshipID, input)`
- `UpdateFollowUp(observerUserID, relationshipID, followupID, input)`
- `CompleteFollowUp(observerUserID, relationshipID, followupID)`
- `DismissFollowUp(observerUserID, relationshipID, followupID)`
- `ListFollowUps(observerUserID, relationshipID)`

### 9.5 Shared context

- `ProjectSharedContext(observerUserID, subjectUserID)`

Rules:

- uses only reliable server-known relationships;
- does not infer meaning;
- does not create facts automatically unless the owner explicitly saves something;
- safe when no shared data exists.

---

## 10. HTTP/API surface

Exact paths may follow repository conventions. Required capabilities:

```text
GET    /api/player-relationships
POST   /api/player-relationships
GET    /api/player-relationships/{relationship_id}
PATCH  /api/player-relationships/{relationship_id}
POST   /api/player-relationships/{relationship_id}/archive
POST   /api/player-relationships/{relationship_id}/unarchive

POST   /api/player-relationships/{relationship_id}/pages/{page_key}
DELETE /api/player-relationships/{relationship_id}/events/{event_id}

GET    /api/player-relationships/{relationship_id}/journal
POST   /api/player-relationships/{relationship_id}/journal
PATCH  /api/player-relationships/{relationship_id}/journal/{entry_id}
DELETE /api/player-relationships/{relationship_id}/journal/{entry_id}

GET    /api/player-relationships/{relationship_id}/followups
POST   /api/player-relationships/{relationship_id}/followups
PATCH  /api/player-relationships/{relationship_id}/followups/{followup_id}
POST   /api/player-relationships/{relationship_id}/followups/{followup_id}/complete
POST   /api/player-relationships/{relationship_id}/followups/{followup_id}/dismiss
```

Creating a relationship from a Trailer Face should accept the subject's public profile/workbook ID, not raw subject UUID.

Responses must not include:

- observer raw UUID;
- subject raw UUID where avoidable;
- subject private account fields;
- subject workbook/history;
- observer notes to anyone but observer.

---

## 11. Invalidation and live updates

### 11.1 Relationship-private updates

A relationship workbook is private to the observer. If live updates are added:

- send only to the observer's own sessions;
- do not notify the subject;
- do not broadcast relationship IDs globally.

A simple refetch-after-mutation is acceptable for Kernel 62 if the UI remains correct.

### 11.2 Subject Trailer changes

When a subject changes stage name or Face presentation, the observer's relationship detail page should display the current stage name/portrait after reload.

If practical, reuse `/ws/player-profile` to update the subject Face header live when the observer is viewing that relationship.

Do not block PASS solely on live subject-Face updates if all privacy and persistence requirements pass, unless the implementation already depends on stale cached stage names.

---

## 12. Authority and privacy

### 12.1 Owner-only relationship access

Every relationship route must prove:

```text
relationship.observer_user_id == authenticated_user_id
```

If not:

```text
404
```

Prefer 404 over 403 to avoid confirming that a private relationship exists.

### 12.2 Subject cannot query observer notes

A subject attempting to read or mutate another user's relationship about them must get 404 or no route shape that allows it.

### 12.3 Other users cannot enumerate

No route should allow:

- list everyone who has notes about me;
- count relationships about me;
- search private relationship notes globally;
- infer archive state;
- infer categories;
- infer journal existence.

### 12.4 Operator caveat

Application UI/API must enforce privacy.

Do not claim cryptographic secrecy from the server operator.

### 12.5 Cross-account mutation tests

Required negative tests:

- subject tries to read observer relationship;
- third user tries to read relationship;
- subject tries to write private nickname;
- third user tries to delete journal;
- third user tries to archive relationship;
- client tries to spoof `observer_user_id`.

All fail.

---

## 13. Shared context rules

### 13.1 Allowed automatic context

Only surface data the server can reliably verify:

- shared productions;
- shared roles in a production;
- shared scoped venue access if it is not merely global/common;
- first shared production membership date if available;
- current overlapping production membership.

### 13.2 Deferred automatic context

Do not implement unless reliable tables already exist:

- games attended together;
- session co-attendance;
- chat activity;
- direct interaction frequency;
- "last talked to";
- emotional closeness;
- recommendation ranking.

If a source is not reliable, leave it out.

### 13.3 Manual context

Allow the observer to manually record:

- how we met;
- systems played together;
- campaigns;
- projects;
- notes about shared work.

Manual notes must remain clearly private.

---

## 14. Archive behavior

### 14.1 Archive

Archiving:

- sets `archived_at`;
- hides from default My People list;
- keeps all notes/facts/journals/follow-ups;
- does not affect subject account;
- does not notify subject;
- can be undone.

### 14.2 Archived view

My People must have a filter:

```text
Active
Archived
All
```

Archived rows show subdued styling and an `Unarchive` action.

### 14.3 No destructive relationship deletion in Kernel 62

Do not add relationship delete unless explicitly reauthorized later.

Journal entry delete is allowed.

---

## 15. Tests

### 15.1 Database/domain tests

- cannot create self-relationship;
- unique relationship per observer/subject;
- relationship follows subject through stage-name changes;
- categories allow multiple values;
- custom category requires label;
- qualitative values reject invalid keys;
- archive hides from default list;
- unarchive restores;
- shared context returns only verified shared data;
- no inferred trust/closeness.

### 15.2 API tests

- create relationship from subject profile ID;
- list own active relationships;
- list archived relationships;
- get own relationship detail;
- update private nickname;
- update categories;
- update qualitative fields;
- archive/unarchive;
- save workbook page;
- create/update/delete journal entry;
- create/complete/dismiss follow-up;
- subject gets 404 trying to read observer relationship;
- third user gets 404;
- spoofed observer ID ignored/rejected.

### 15.3 Privacy tests

Responses for non-observers never include:

- private nickname;
- categories;
- qualitative fields;
- journal title/body;
- follow-up data;
- relationship existence.

### 15.4 Trailer integration tests

- another user's Trailer Face shows `Add to My People`;
- after creation it shows `Open My Notes`;
- social Face itself remains unchanged for other viewers;
- adding someone does not alter their Trailer or notify them.

### 15.5 Stage-name update tests

- subject changes stage name;
- relationship list displays new current stage name;
- private nickname remains unchanged;
- relationship row still points to same subject account.

### 15.6 Clean install

Update fresh-install smoke with minimal Kernel 62 assertions:

- fresh account A and B can exist;
- A can open B's Trailer Face by copied link/profile ID;
- A can create relationship with B;
- A can save private nickname;
- A can write journal entry;
- B cannot read A's relationship record;
- A can archive relationship.

---

## 16. Browser acceptance

Use real browser sessions.

### 16.1 Add from Trailer

1. User A has a Trailer Face.
2. User B opens A's Trailer Face.
3. B clicks `Add to My People`.
4. B sees private relationship workspace.
5. A's browser receives no notification.
6. A's Trailer does not change.

### 16.2 My People

1. B opens My People.
2. A appears with current stage name and portrait.
3. B adds private nickname.
4. B checks multiple categories.
5. B sets trust/closeness/reliability/communication dropdowns.
6. B reloads; data persists.

### 16.3 Journal

1. B opens A relationship.
2. B writes journal entry.
3. B filters by tag/category.
4. B edits entry.
5. B deletes entry.
6. Entry disappears and does not appear anywhere for A.

### 16.4 Follow-up

1. B creates follow-up item.
2. B marks it done.
3. B creates another and dismisses it.
4. No notification/reminder is generated.

### 16.5 Archive

1. B archives relationship with A.
2. A disappears from default Active list.
3. A appears in Archived filter.
4. B unarchives.
5. A returns to Active list.
6. A is never notified.

### 16.6 Privacy / cross-user

1. A tries to open B's relationship detail URL about A.
2. A gets 404 or safe not-found.
3. User C tries same.
4. C gets 404 or safe not-found.
5. Direct API attempts to mutate B's notes fail.
6. Browser DOM for A never includes B's private note content.

### 16.7 Stage-name change

1. A changes stage name.
2. B's My People list shows A's new current stage name after refresh or live update.
3. B's private nickname remains unchanged.
4. B's notes remain attached to same person.

### 16.8 Desktop/mobile

Capture:

- My People desktop;
- My People mobile;
- relationship detail desktop;
- relationship detail mobile;
- journal editor;
- follow-up list;
- archive filter;
- Add to My People on another user's Trailer;
- privacy failure/404 state.

---

## 17. Required validation commands

Run and record:

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test ./internal/playerprofile/...
GOCACHE=/tmp/victory-gocache go test ./internal/playerrelationships/...
GOCACHE=/tmp/victory-gocache go test ./internal/identity/...
GOCACHE=/tmp/victory-gocache go test ./internal/access/...
GOCACHE=/tmp/victory-gocache go test ./internal/network/...
GOCACHE=/tmp/victory-gocache go build ./...
GOCACHE=/tmp/victory-gocache go vet ./...
GOCACHE=/tmp/victory-gocache go test ./...
```

Frontend:

```bash
node --check [every touched HTML inline script or JS file]
node --check [browser smoke scripts]
```

Repository:

```bash
git diff --check
```

Fresh install:

```bash
scripts/smoke/fresh-install.sh --local
```

Production rebuild:

```bash
docker compose up -d --build backend
```

Record pre-existing unrelated failures honestly.

---

## 18. Documentation and project-memory duties

Before reporting completion:

1. save this kernel in the repository under the canonical Kernels path;
2. create `Construction/OperatorLogs/kernel-62-reportback.md`;
3. append a dated entry to `operator-log.md`;
4. update `operator-notes.md` with:
   - private directional relationship model;
   - subject invisibility rule;
   - qualitative dropdown vocabulary;
   - archive behavior;
   - no notifications for follow-ups;
   - shared context limits;
   - privacy caveat about self-hosted operators;
5. update `kernel-maker-field-guide.md` if two-browser/privacy workflow changes;
6. update `dev-workflow.md` if fresh-install or Playwright workflow changes;
7. update Master Actual Implementation Guide;
8. update Parallel Track Roadmaps;
9. update fresh-install smoke;
10. record changed files and commit hash/status;
11. document next recommended kernel.

Unchecked memory work means PARTIAL.

---

## 19. Acceptance-criterion ledger

The reportback must mark every row **PASS**, **FAIL**, or **NOT DONE** with evidence.

| Criterion | Status | Evidence |
|---|---|---|
| Kernel 61A contracts reused, not duplicated |  |  |
| Relationship records are directional |  |  |
| Subject cannot see relationship exists |  |  |
| Cannot create self-relationship |  |  |
| Unique relationship per observer/subject |  |  |
| Add to My People from Trailer works |  |  |
| Open My Notes from Trailer works |  |  |
| My People list works |  |  |
| Search/filter/sort works |  |  |
| Private nickname works |  |  |
| Multiple categories work |  |  |
| Custom category works |  |  |
| Qualitative dropdowns work |  |  |
| Invalid qualitative values rejected |  |  |
| Shared context uses only reliable data |  |  |
| No inferred trust/closeness |  |  |
| Relationship Workbook pages save |  |  |
| Relationship facts recompute |  |  |
| Private journal create/edit/delete works |  |  |
| Journal private from subject |  |  |
| Follow-ups store/open/done/dismissed |  |  |
| No reminders/notifications created |  |  |
| Archive hides from default list |  |  |
| Archived filter works |  |  |
| Unarchive restores |  |  |
| No destructive relationship delete added |  |  |
| Subject stage-name change preserves relationship |  |  |
| Private nickname unaffected by stage-name change |  |  |
| Cross-account reads rejected |  |  |
| Cross-account writes rejected |  |  |
| Spoofed observer ID rejected/ignored |  |  |
| Social Face does not expose notes |  |  |
| Clean-install smoke updated and passes |  |  |
| Existing accounts preserved |  |  |
| Desktop browser proof attached |  |  |
| Mobile browser proof attached |  |  |
| Two-user privacy proof attached |  |  |
| Required automated checks recorded |  |  |
| Operator and roadmap docs updated |  |  |

Any required row marked FAIL or NOT DONE means Kernel 62 remains **PARTIAL**.

---

## 20. Definition of done

Kernel 62 reaches **PASS** only when:

```text
User B opens User A's Trailer
→ B adds A to My People
→ B records private nickname, categories, qualitative relationship values, notes, journal entries, and follow-ups
→ B archives and unarchives the relationship
→ A never sees or receives any of this
→ stage-name changes keep the relationship attached to the same stable person
→ cross-account reads/writes fail
→ clean install and browser proof pass
→ documentation and roadmap updates are complete
```

A private notes backend without My People UI, or a visible social feature without privacy proof, remains PARTIAL.
