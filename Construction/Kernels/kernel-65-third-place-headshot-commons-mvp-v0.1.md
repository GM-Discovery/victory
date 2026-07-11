# Kernel 65 — Third Place Headshot Commons MVP

**Revision:** 0.1
**Status:** READY FOR BUILD
**Primary track:** V — Player social presence and venue commons
**Complementary tracks:** O — privacy, test safety, and account stability; C — concierge/private-community social tooling; S — Socio player discovery and continuity
**Kernel type:** product, venue, social-presence, profile integration, frontend, backend, verification
**Depends on:** Kernel 61A PASS; Kernel 62 PASS; Kernel 63 PASS; Kernel 64 PASS; Trailer Face; My People; targeted profile websocket; Back-to-Map component; dedicated test DB safety gate; authenticated account/session identity
**Expected next consumer:** Show Run primitive, Run Roster MVP, Headshot impressions/autographs, or Third Place social-context expansion

---

## 1. Objective

Build the first version of **Third Place**, a signed-in Victory venue where users can leave one active **Headshot** that represents their opt-in social presence.

The completed loop is:

```text
signed-in user opens Third Place
→ clicks Leave Headshot
→ one active Headshot row exists for that account
→ Headshot renders from the user's live Trailer Face
→ another signed-in user can browse/search/sort Headshots
→ another user can open the Trailer or Add to My People / Open My Notes
→ owner can remove their Headshot
→ historic Headshot placement/removal records remain
```

Third Place is not a feed, public social network, comment wall, autograph system, or run roster. It is the smallest useful opt-in social commons built on top of Trailer Faces and My People.

---

## 2. Why this kernel now

Victory now has the pieces Third Place needs:

- **Kernel 61A:** Player Workbook, Trailer Face, cross-user Trailer viewing, stage-name stability, live profile websocket, and privacy boundaries.
- **Kernel 62:** My People, private directional relationships, Add to My People / Open My Notes integration, journal/follow-up/archive behavior, and subject invisibility.
- **Kernel 63:** consistent Back to Map navigation and cleaned Discord fixture leaks.
- **Kernel 64:** DB-touching tests now require a dedicated `TEST_DATABASE_URL`, unsafe test DB URLs are rejected, and `go test ./...` is meaningful again with the dedicated test DB.

This kernel resumes product work while keeping the scope narrow.

---

## 3. Resolved owner decisions

These decisions are fixed.

### 3.1 Name

The venue is:

```text
Third Place
```

The opt-in presence item is:

```text
Headshot
```

Do not call it Faceprint in product copy, database comments, UI labels, or reportback language unless referring to an abandoned earlier concept.

### 3.2 Access timing and visibility

Third Place appears at the same time and under the same broad account/venue visibility assumptions as Trailers.

Meaning:

- signed-in users who can see/use Trailers should be able to see/use Third Place;
- no anonymous public access;
- no internet-public Third Place;
- no special Producer/Director/Operator-only gate;
- no new social-access role system.

If the repository has a canonical venue-visibility seeding path for Trailers, Third Place should follow that path.

### 3.3 Live Headshot rendering

A Headshot always renders from the user's current live Trailer Face projection.

Do not snapshot or duplicate old Trailer Face content into Third Place.

If a user changes their stage name, portrait, visible Trailer facts, or Face priority, their active Headshot should reflect the current Trailer Face after refresh and ideally through existing profile invalidation.

### 3.4 One active Headshot per account

Each account may have at most one active Headshot in Third Place.

Repeated `Leave Headshot` should be idempotent:

```text
no active Headshot → create active Headshot
active Headshot exists → return existing active Headshot / refresh updated_at
```

### 3.5 Historic Headshot records

Keep historic Headshot placement/removal records.

History records preserve:

- user/account identity internally;
- when a Headshot was placed;
- when it was removed;
- source/version fields needed for audit/debug;
- optional note/status metadata if useful.

History records do **not** preserve old Trailer Face content. Since Headshots are live, old profile snapshots would fight the model.

### 3.6 Persistence

A Headshot persists until the owner removes it.

No automatic expiration in Kernel 65.

### 3.7 Deferred features

Defer:

- impressions;
- autographs;
- comments;
- reactions;
- public tags;
- user-created circles;
- show/run rosters;
- feeds;
- notifications;
- algorithmic discovery;
- anonymous/public access.

---

## 4. Product boundary

### 4.1 What Kernel 65 is

Kernel 65 creates:

```text
Third Place venue
A signed-in social commons available alongside Trailers.

Headshot
One active live Trailer-Face presence marker per account.

Headshot Commons
A searchable/sortable roster-like view of active Headshots.

Headshot History
A private/operator-safe history of placement/removal events, not old Face snapshots.
```

### 4.2 What Kernel 65 is not

Do not build:

- Third Place impressions;
- autograph walls;
- public tags on people;
- relationship tags;
- show-run tags;
- Show Run;
- Run Roster;
- friend requests;
- mutual connections;
- follower lists;
- direct messages;
- comments;
- likes;
- global directory;
- moderation commands;
- `/ban` or `/boot`;
- account deletion;
- stale-user sweep;
- CI wiring.

---

## 5. Canonical domain model

Exact table names may follow repo conventions, but the domain must remain clear.

### 5.1 Third Place venue

Create or seed a venue:

```text
slug: third-place
name: Third Place
```

Visibility/access:

- same broad availability as Trailers;
- signed-in only;
- appears on the map/venue list when Trailers appears;
- includes Back to Map via the shared component from Kernel 63.

### 5.2 Headshot records

Recommended table:

```text
third_place_headshots
- id UUID primary key
- user_id UUID not null references users(id)
- status TEXT not null default 'active'
- placed_at TIMESTAMPTZ not null default now()
- removed_at TIMESTAMPTZ nullable
- created_at TIMESTAMPTZ not null default now()
- updated_at TIMESTAMPTZ not null default now()
```

Recommended indexes/constraints:

```text
unique active Headshot per user:
UNIQUE(user_id) WHERE removed_at IS NULL AND status = 'active'

index active Headshots by placed_at
index Headshot history by user_id, placed_at
```

Allowed statuses:

```text
active
removed
```

Rules:

- `active` rows have `removed_at IS NULL`;
- `removed` rows have `removed_at IS NOT NULL`;
- only one active row per user;
- removing a Headshot closes the active row rather than deleting it;
- re-leaving a Headshot after removal creates a new active row;
- historic rows do not store old Trailer Face content.

### 5.3 Headshot projection

The Headshot projection should be derived live from the user's current Trailer Face.

Recommended projection shape:

```json
{
  "headshot_id": "uuid",
  "profile_id": "workbook/profile id from Kernel 61A",
  "placed_at": "timestamp",
  "stage_name": "current stage name",
  "portrait_url": "current portrait if visible/available",
  "headline_facts": [],
  "trailer_url": "/venues/trailers/view.html?id=...",
  "relationship_state_for_viewer": "none|exists",
  "can_add_to_my_people": true,
  "can_open_my_notes": false
}
```

Do not expose:

- user raw UUID;
- email;
- handle;
- account metadata;
- Player Workbook source answers;
- Trailer History;
- stage-name ledger;
- private relationship notes;
- private nickname;
- relationship journal;
- follow-ups.

### 5.4 Headline facts

Use a small number of visible facts from the live Trailer Face.

Recommended selection:

- stage name always;
- portrait/banner where available;
- up to 3 facts from `At a Glance` or equivalent Trailer Face region;
- optional current relationship action state for viewer.

Do not display every Trailer Face field in the Headshot grid. Opening the Trailer remains the full view.

---

## 6. Backend operations

Implement focused operations.

### 6.1 Headshot lifecycle

- `LeaveHeadshot(userID)`
- `RemoveHeadshot(userID)`
- `GetMyHeadshot(userID)`
- `ListActiveHeadshots(viewerUserID, filters)`
- `ListMyHeadshotHistory(userID)`
- `ProjectHeadshot(viewerUserID, headshotRow)`

Rules:

- authenticated user comes from session;
- no client-supplied `user_id` accepted;
- leave is idempotent while active;
- remove is idempotent when no active row exists;
- list returns active Headshots only by default;
- history endpoint returns only the caller's own placement/removal history unless an existing Operator-only diagnostic pattern already exists and is explicitly used;
- Headshot projection calls/reuses Trailer Face projection rather than duplicating face data.

### 6.2 Search/sort/filter

Minimum list behavior:

- search by current stage name;
- sort by recently placed;
- sort by stage name;
- filter to active only.

Optional if cheap:

- search headline facts;
- sort by recently updated Trailer Face;
- filter by users already in My People vs not yet added.

Do not build tags.

### 6.3 Relationship integration

Reuse Kernel 62.

From a Headshot, viewer can:

- open Trailer;
- Add to My People if no relationship exists;
- Open My Notes if relationship exists.

Rules:

- adding someone to My People does not notify them;
- Headshot owner does not see who added them;
- no relationship data appears in the public Headshot projection except action state for the current viewer, such as whether **this viewer** already has notes.

### 6.4 Profile invalidation

If feasible, reuse `/ws/player-profile` to refresh visible Headshots when a user changes their Trailer Face.

PASS may allow refresh-on-reload if all core Headshot lifecycle and privacy behavior pass, but live update is strongly preferred because Kernel 61A already provides the mechanism.

If live updating is implemented:

- a Headshot watcher subscribes to visible profile IDs or refreshes the list on relevant profile updates;
- no private data is pushed;
- unrelated/stale events are ignored.

---

## 7. HTTP/API surface

Exact paths may follow repo conventions.

Required capabilities:

```text
GET    /api/third-place/headshots
GET    /api/third-place/headshots/me
POST   /api/third-place/headshots/me
DELETE /api/third-place/headshots/me
GET    /api/third-place/headshots/me/history
```

Optional if cleaner:

```text
POST /api/third-place/headshots/{headshot_id}/add-to-my-people
```

But prefer reusing the existing Kernel 62 relationship creation endpoint from the frontend rather than creating a duplicate backend path.

Authorization:

- all routes require authenticated user;
- anonymous requests return 401;
- mutation routes always act on authenticated user;
- no route permits one user to remove another user's Headshot in Kernel 65.

---

## 8. Frontend requirements

### 8.1 Third Place venue page

Create:

```text
frontend/venues/third-place/index.html
```

or the repo's canonical equivalent.

Must include:

- clear title: `Third Place`;
- short purpose copy;
- Back to Map visible without hover hunting;
- account/identity affordance consistent with nearby venues if available;
- Headshot status panel;
- Leave Headshot / Remove My Headshot button;
- active Headshot grid/list;
- search;
- sort;
- empty state;
- mobile layout.

### 8.2 Headshot card

Each card shows:

- current stage name;
- portrait or fallback avatar;
- placed date;
- up to 3 headline facts;
- Open Trailer;
- Add to My People or Open My Notes;
- clear indication if this is `You`;
- no private notes or relationship details.

### 8.3 My Headshot status

If no active Headshot:

```text
You have not left a Headshot in Third Place yet.
```

If active:

```text
Your Headshot is visible in Third Place.
```

Actions:

```text
Leave Headshot
Remove My Headshot
```

### 8.4 History

The owner should be able to see their own Headshot history somewhere simple:

- placed date;
- removed date or Active;
- status.

No old profile snapshot rendering.

### 8.5 Responsive/accessibility

Required:

- keyboard-accessible buttons/links;
- semantic headings;
- readable card layout;
- mobile proof;
- no hover-only critical controls;
- Back to Map visible;
- no horizontal overflow;
- alt text or accessible names for portraits.

---

## 9. Authority and privacy

### 9.1 Authenticated only

Third Place and its API require authentication.

Anonymous users cannot list Headshots.

### 9.2 Owner-only mutation

Only the authenticated owner can leave/remove their Headshot.

No request body may specify another user to mutate.

### 9.3 No private leakage

Headshot list/projection must not contain:

- email;
- handle;
- raw account UUID;
- private workbook answers;
- profile event history;
- stage-name history;
- relationship notes;
- private nickname;
- journal entries;
- follow-ups;
- archive state from My People.

### 9.4 Relationship action privacy

It is acceptable for the current viewer's own UI to show:

```text
Add to My People
```

or:

```text
Open My Notes
```

This must be calculated per viewer and must not expose anyone else's relationship state.

### 9.5 Historic records

Historic Headshot records are not a public feed.

For Kernel 65:

- owner can view their own Headshot history;
- public Third Place shows active Headshots only;
- removed Headshots do not appear in the active commons.

---

## 10. Tests

### 10.1 Database/domain tests

- cannot create more than one active Headshot per user;
- `LeaveHeadshot` is idempotent while active;
- `RemoveHeadshot` closes active row and preserves history;
- re-leaving after removal creates a new active row;
- listing active Headshots excludes removed rows;
- projection uses current Trailer Face data;
- no old Face snapshot is stored;
- mutation uses authenticated user only;
- anonymous access rejected;
- unsafe client-supplied user IDs ignored/rejected.

### 10.2 API tests

- GET list authenticated works;
- GET list anonymous rejected;
- POST leave creates active Headshot;
- repeated POST does not create duplicate;
- DELETE removes active Headshot;
- repeated DELETE is safe/idempotent;
- history shows placed/removed records for owner;
- other user cannot remove owner's Headshot;
- response payload excludes private fields.

### 10.3 Relationship integration tests

- Headshot for unknown-to-viewer user shows Add to My People;
- after relationship exists, same Headshot shows Open My Notes;
- Add to My People creates private directional relationship only for viewer;
- owner is not notified;
- owner's Headshot does not change.

### 10.4 Live Trailer Face tests

- user leaves Headshot;
- user changes stage name or visible Trailer fact;
- Headshot projection reflects current Trailer Face after refresh;
- if live update implemented, open Third Place updates without reload.

### 10.5 Dedicated test DB

All DB-touching tests must run with `TEST_DATABASE_URL` per Kernel 64.

Do not add tests that fall back to live `DATABASE_URL`.

---

## 11. Browser acceptance

Use real browser sessions.

### 11.1 Leave/remove Headshot

1. User A signs in.
2. A opens Third Place.
3. A sees no active Headshot state.
4. A clicks Leave Headshot.
5. A appears in active Headshot list.
6. A clicks Leave Headshot again or reloads and verifies no duplicate.
7. A clicks Remove My Headshot.
8. A disappears from active list.
9. A's history shows placed and removed timestamps.
10. A leaves another Headshot.
11. History now shows prior removed row plus current active row.

### 11.2 Viewer actions

1. User B signs in.
2. B opens Third Place.
3. B sees A's Headshot.
4. B opens A's Trailer.
5. B returns to Third Place.
6. B clicks Add to My People.
7. B now sees Open My Notes for A.
8. A receives no notification and sees no evidence B added them.

### 11.3 Live projection

1. A has active Headshot.
2. B sees A's Headshot.
3. A changes stage name or visible Trailer Face fact.
4. B refreshes Third Place and sees current data.
5. If live update implemented, B's open page updates without refresh.

### 11.4 Privacy proof

1. Anonymous user attempts Third Place API/list page and is rejected.
2. B inspects Headshot payload/DOM.
3. Payload/DOM contains no email, handle, raw UUID, private notes, journal text, relationship nickname, follow-ups, or stage-name history.
4. B attempts to remove A's Headshot through route/body manipulation and fails.

### 11.5 Desktop/mobile

Capture:

- Third Place empty state;
- active Headshot list desktop;
- active Headshot list mobile;
- My Headshot active state;
- Headshot history;
- Add to My People state;
- Open My Notes state;
- privacy/anonymous rejection evidence.

---

## 12. Required validation commands

Use Kernel 64 workflow.

```bash
cd /opt/victory
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh

cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache \
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
go test ./...

GOCACHE=/tmp/victory-gocache go build ./...
GOCACHE=/tmp/victory-gocache go vet ./...
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

Record any remaining failures honestly.

---

## 13. Documentation and project-memory duties

Before reporting completion:

1. save this kernel in the repository under the canonical Kernels path;
2. create `Construction/OperatorLogs/kernel-65-reportback.md`;
3. append a dated entry to `operator-log.md`;
4. update `operator-notes.md` with:
   - Third Place venue rule;
   - Headshot terminology;
   - one active Headshot per account;
   - live Trailer Face projection rule;
   - Headshot history stores placement/removal, not old profile snapshots;
   - deferred impressions/autographs/tags/rosters;
5. update `kernel-maker-field-guide.md` only if the test/browser workflow changes;
6. update `dev-workflow.md` only if commands change;
7. update Master Actual Implementation Guide;
8. update Parallel Track Roadmaps;
9. update fresh-install smoke if Third Place must exist in clean install;
10. record changed files and commit hash/status;
11. document next recommended kernel.

Unchecked required memory work means PARTIAL.

---

## 14. Acceptance-criterion ledger

The reportback must mark every row **PASS**, **FAIL**, or **NOT DONE** with evidence.

| Criterion | Status | Evidence |
|---|---|---|
| Third Place venue exists |  |  |
| Third Place appears with Trailers-level visibility |  |  |
| Third Place requires authentication |  |  |
| Back to Map visible and works |  |  |
| Headshot terminology used consistently |  |  |
| One active Headshot per account enforced |  |  |
| Leave Headshot works |  |  |
| Leave Headshot is idempotent while active |  |  |
| Remove My Headshot works |  |  |
| Re-leave after removal creates new active record |  |  |
| Historic placement/removal records preserved |  |  |
| History does not snapshot old Face content |  |  |
| Active Headshot renders live Trailer Face |  |  |
| Stage-name/Face changes reflect in Headshot after refresh |  |  |
| Headshot list/search/sort works |  |  |
| Headshot card opens Trailer |  |  |
| Headshot card can Add to My People |  |  |
| Existing relationship shows Open My Notes |  |  |
| Adding to My People is private/directional |  |  |
| Owner is not notified when added |  |  |
| Removed Headshots absent from active list |  |  |
| Anonymous list/API rejected |  |  |
| Other user cannot remove owner Headshot |  |  |
| Payload excludes email/handle/raw UUID |  |  |
| Payload excludes relationship notes/journal/follow-ups |  |  |
| No impressions/autographs/tags/feeds built |  |  |
| Desktop browser proof attached |  |  |
| Mobile browser proof attached |  |  |
| Dedicated test DB workflow used |  |  |
| go test ./... recorded with TEST_DATABASE_URL |  |  |
| fresh-install smoke passes |  |  |
| Operator and roadmap docs updated |  |  |

Any required row marked FAIL or NOT DONE means Kernel 65 remains **PARTIAL**.

---

## 15. Definition of done

Kernel 65 reaches **PASS** only when:

```text
Third Place appears alongside Trailers
→ user leaves one active Headshot
→ Headshot renders from live Trailer Face
→ other users can browse Headshots
→ other users can open Trailer or Add to My People
→ owner can remove and re-leave Headshot
→ history preserves placement/removal only
→ privacy boundaries hold
→ no impressions/autographs/tags/feeds/rosters are built
→ desktop/mobile browser proof, dedicated test DB proof, fresh install, and docs pass
```

A Third Place page without persistence, or a Headshot system that snapshots old Face contents, or a social feature that grows into tags/impressions/feeds, remains PARTIAL.
