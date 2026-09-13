# Kernel 68 — Venue Visibility Gates, Stage Management Surface, and Production Onboarding

## 0. Purpose

Victory now has the social/player spine and the show/run spine:

```text
Trailer → Trailer Face → Third Place Headshots → My People
Production → Show Run → Show → Session link
```

Kernel 68 tightens the visibility and readiness rules before adding more VTT-core scene machinery.

The goals are:

1. Hide **Third Place** until the user has intentionally set a usable **Trailer Face**.
2. Treat **Stage Management** as the user-facing backstage venue for the existing Show Runs tools.
3. Hide Stage Management from users who do not have backstage authority.
4. Add the missing minimal **Create Production** button/flow so Show Run setup is not blocked by an empty Production picker.
5. Increase map cloud/fog behavior so hidden/locked venues feel undiscovered rather than merely absent.
6. Keep crew edit authority low-priority and non-destructive; do not build a permissions monster.

## 1. Resolved operator decisions

These are locked and should not be re-asked.

### 1.1 Third Place readiness

Third Place should only appear after the user has set their Trailer Face.

Readiness requires **both**:

```text
B. The user has intentionally saved/published/committed their Trailer Face.
C. The user has a stage name plus at least one visible Face field beyond bare defaults.
```

Stage name alone is not enough.

A valid visible Face field may include an intentionally visible portrait, public description, featured quote, favorite TTRPGs, class/archetype answer, or another existing Trailer Face fact projected publicly.

### 1.2 Third Place visibility

Third Place should be **hidden from the map** until Trailer Face readiness is true.

Do not display an inert Third Place tile to unready users. Instead, Trailers should explain the unlock:

```text
Set your Trailer Face to enter Third Place.
```

Direct URL/API access should return a clean authorization/readiness failure, not leak private data.

### 1.3 Map cloud/fog behavior

The map should use heavier cloud/fog coverage for undiscovered/locked areas. As more venues become visible, the clouds should visibly recede.

For this kernel, the minimum acceptable implementation is:

- noticeably increase the cloud/fog layer over hidden/locked venue areas;
- make the cloud/fog recede when the venue becomes visible;
- do not reveal hidden venue names through the cloud layer;
- ensure Third Place appears from behind the cloud after Trailer Face readiness.

Do not build a full exploration/metroidvania system. This is a visual readiness/visibility pass.

### 1.4 Stage Management

**Stage Management** is the user-facing backstage venue for the existing Show Runs tools.

The internal slug may remain `show-runs` to avoid churn, but the map/page label should become **Stage Management** where user-facing.

Stage Management appears only for:

- Operator;
- Producer at the relevant location;
- Director at the relevant location;
- optionally, appointed Show Run crew editor if this can be implemented without a large schema.

Audience, Player-only, Guest, and Observer users should not see the Stage Management venue tile.

Audience Program pages must remain accessible through their own allowed routes/links where appropriate. Hiding Stage Management from Audience must not break curated audience access to a Show Program.

### 1.5 Crew edit authority

Crew permissions are low priority.

Do **not** build a full permission schema.

Implement crew edit authority only if it can be done safely using existing Show Run roster rows or a very small additive change. If it requires a full schema or a broad permission matrix, defer it explicitly.

If implemented, crew edit authority must be Show Run-scoped and non-destructive.

Allowed for crew editor:

- edit non-destructive Show Run presentation details;
- edit non-destructive Show details;
- edit audience-facing blurbs/program text;
- later, help with scene configuration when that exists.

Not allowed for crew editor:

- archive/delete/cancel a Show Run;
- archive/delete/cancel a Show;
- appoint/revoke other crew;
- block/ban users;
- change Producer/Director authority;
- edit location membership;
- perform Operator-only actions.

If no safe low-friction implementation exists, this kernel should leave a clear note and move on.

### 1.6 Production creation button

Add a minimal in-app **Create Production** button and make sure it works.

Kernel 66 identified a real onboarding gap: Show Run creation consumes `/api/productions`, but there is no in-app Production creation flow. Kernel 68 should close that gap enough for a real Producer/Director/Operator to create the first Production without direct database access.

### 1.7 Browser/visual proof

If headless browser tooling is available, provide screenshots. If it is not available, provide authenticated HTTP proof **plus** a manual visual-check checklist for the operator.

The operator can visually check if told exactly what to look for.

## 2. Current-state dependencies and facts to preserve

- Kernel 65 made Third Place a venue full of Headshots, with one active live Headshot per account and Trailer Face projection. Preserve that model.
- Kernel 66 created Show Runs, roster, audience program, and the `show-runs` venue. Preserve the Show Run APIs and pages.
- Kernel 67 created Shows as a new user-facing concept under Show Runs, leaving old `showings` untouched and adding an optional `sessions.show_id` link. Preserve `/session` and `/mic` behavior.
- Kernel 64 established the dedicated `TEST_DATABASE_URL` workflow. All DB-touching tests must use it.

## 3. Required work

### 3.1 Trailer Face readiness function

Create a single backend readiness function, for example:

```go
TrailerFaceReady(ctx, pool, userID) (ReadyResult, error)
```

It should report:

```text
ready: true/false
reason_code: face_ready | missing_face_commit | missing_visible_face_field | not_authenticated | other
visible_field_count
has_stage_name
```

Use existing Trailer Face/workbook/projection data where possible.

Prefer a computed readiness check. If existing data cannot reliably distinguish intentional Face setup from defaults, add the smallest possible durable marker, such as a `face_ready_at` timestamp or profile event, but do not build a checklist subsystem.

### 3.2 Hide Third Place until ready

Update venue visibility so Third Place is only returned when Trailer Face readiness is true.

Required behavior:

- new account can see Trailers;
- new account cannot see Third Place;
- new account sees a Trailers prompt telling them to set Trailer Face;
- after completing Face readiness, Third Place appears on the map;
- direct Third Place API access by an unready authenticated user fails cleanly;
- anonymous users still fail authentication.

### 3.3 Trailers unlock prompt

Add a clear prompt to Trailers/My Face when the user is not ready:

```text
Set your Trailer Face to enter Third Place.
```

When ready, show something like:

```text
Third Place unlocked.
```

Keep this small. Do not redesign Trailer Face.

### 3.4 Map cloud/fog pass

Adjust the map UI so hidden/locked venues feel clouded over.

Minimum behavior:

- heavier cloud/fog layer by default;
- clouds recede as venue visibility increases;
- Third Place is hidden under cloud/fog until ready;
- cloud/fog does not reveal labels for locked venues;
- no broken click targets under the cloud.

If exact per-venue cloud geometry is too much, implement a simpler cloud-level model tied to the count/category of visible venues, then document it.

### 3.5 Rename/show user-facing Show Runs venue as Stage Management

Without breaking existing routes, update user-facing labels:

```text
/venues/show-runs/ → Stage Management
```

Do not rename DB slugs or routes unless the codebase already has a clean alias pattern.

Expected changes:

- map tile label: Stage Management;
- page heading/nav language: Stage Management;
- Back to Map still works;
- API names may remain `show-runs` unless low-risk to alias.

### 3.6 Stage Management visibility gate

Update venue visibility so Stage Management appears only for backstage authority.

Suggested rule:

```text
visible if Operator OR location Producer OR location Director OR eligible Show Run crew editor
```

Audience-only users should not see the tile. Player-only users should not see the tile by default.

Direct URL access should return a clean 403/not-authorized for users without backstage authority. Curated Show Program access must still work through its own page/API.

### 3.7 Minimal Create Production flow

Audit existing `productions` package/routes first.

If there is no usable POST/create route, add one.

Minimum fields:

```text
title/name
slug auto-generated or safely derived
short description optional
location_id resolved server-side
created_by_user_id
status active/draft if existing conventions require it
```

Authority:

- Operator can create;
- Producer/Director at a location can create for that location if consistent with existing access rules;
- never trust client-supplied location authority without server-side verification.

UI:

- add a **Create Production** button in Stage Management / Show Runs where the production picker/list would otherwise block setup;
- make it work end-to-end;
- after creation, the new Production should be selectable for a Show Run.

Keep this minimal. Do not build full Production management unless already almost present.

### 3.8 Optional low-friction crew editor support

Audit whether existing `show_run_roster_members` can support crew editor semantics without new schema.

Potential low-friction rule:

```text
A user with active role = crew on a Show Run may perform non-destructive edits for that Show Run.
```

Only implement if safe and clear.

If implemented, add tests proving:

- crew editor can edit non-destructive Show/Show Run text;
- crew editor cannot archive/cancel/delete;
- crew editor cannot block users;
- crew editor cannot appoint/revoke crew;
- crew editor cannot manage unrelated Show Runs.

If not implemented, document why it was deferred and keep Producer/Director/Operator only.

## 4. Explicit exclusions

Do not build:

- scenes;
- scene capture;
- fly scene;
- cue groups;
- session auto-linking to Show;
- full Production admin suite;
- full crew permissions schema;
- social scarcity/daily Headshot views;
- connection requests;
- circles/tags;
- impressions/autographs;
- feeds/comments;
- tickets/calendar scheduling;
- destructive actions for crew editors.

## 5. Tests and evidence required

### 5.1 Automated backend checks

Run:

```bash
go build ./...
go vet ./...
git diff --check
```

Run full DB test suite with Kernel 64 workflow:

```bash
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh

cd backend
GOCACHE=/tmp/victory-gocache \
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
go test -count=1 ./...
```

Also prove DB-touching tests hard-fail clearly without `TEST_DATABASE_URL` for any new DB-touching package.

### 5.2 Required behavioral tests

Prove:

- blank/new account sees Trailers but not Third Place;
- after Face readiness, Third Place appears;
- Third Place direct API rejects unready authenticated user;
- Third Place still works for ready user;
- Stage Management tile appears for Director/Producer/Operator;
- Stage Management tile does not appear for Audience-only user;
- direct Stage Management URL rejects unauthorized user;
- Show Program remains accessible to authorized Audience despite Stage Management being hidden;
- Create Production button creates a usable Production;
- newly created Production can be used to create a Show Run;
- existing K65/K66/K67 behavior remains intact.

### 5.3 Visual proof / manual checklist

If browser automation is available, provide screenshots for:

1. new/unready user map: Trailers visible, Third Place hidden/clouded;
2. Trailers unlock prompt;
3. ready user map: Third Place visible/cloud receded;
4. Player/Audience map: Stage Management hidden;
5. Director/Producer map: Stage Management visible;
6. Stage Management page with Create Production button;
7. successful Production creation and Show Run creation from it;
8. mobile view if feasible.

If browser automation is not available, provide a manual checklist with exact URLs, accounts/roles, and expected visible text so the operator can visually verify.

### 5.4 Fresh install proof

Update `scripts/smoke/fresh-install.sh` if new migrations/routes require it.

Fresh install must prove at minimum:

- Third Place hidden for unready user;
- Third Place available after readiness fixture/setup;
- Stage Management hidden for Audience-only user;
- Stage Management available for Director/Producer;
- Create Production route works;
- Show Run can be created from newly created Production.

## 6. Documentation updates

Update:

- `Construction/Canon/Dictionary.txt`
  - define Trailer Face Ready;
  - define Stage Management as user-facing name for Show Runs backstage surface;
  - note Third Place unlock rule.
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/operator-notes.md`
- relevant roadmap files if the name Stage Management replaces user-facing Show Runs language.
- reportback template only if a new recurring proof requirement is introduced.

## 7. PASS standard

Kernel 68 passes only if:

1. Third Place is hidden until Trailer Face readiness is true.
2. Trailers clearly tells an unready user how to unlock Third Place.
3. Stage Management is the user-facing label for the Show Runs backstage venue.
4. Stage Management visibility is limited to Operator/Producer/Director and optional safe crew editor.
5. Audience/Player users do not see Stage Management on the map.
6. Curated Audience Program access is not broken.
7. Create Production button works end-to-end and unblocks Show Run creation when no production exists.
8. Map cloud/fog behavior visibly supports hidden/unlocked venues.
9. K65/K66/K67 core behavior remains intact.
10. Kernel 64 dedicated-test-DB workflow is followed.

## 8. Reportback requirements

The reportback must include:

- PASS/PARTIAL/FAIL;
- exact readiness rule implemented;
- whether readiness is computed or uses a new durable marker;
- exact Stage Management visibility rule;
- whether crew editor support was implemented or deferred;
- Production creation route/UI summary;
- cloud/fog implementation summary;
- tests and proof;
- visual screenshots or manual visual checklist;
- known issues;
- files changed;
- next recommended step.
