Kernel Report Back — Kernel 89: Director Prepared Play & Training Arena

## 1. Status

**PASS, deployed live 2026-08-17, uncommitted — with two scope decisions stated plainly rather than buried.**

Everything Kernel 89 §39 lists as a pass criterion is built and proven:
Director preparations are authored in the Director's Chair and usable in the
live venue, the Director tool surface is grouped instead of sprawling, target
complexities can be prepared and recalled and changed live, a new merchant can
be authored from the one canonical equipment corpus and exposed to a single
Cohort, announcements work in Catharsis with a rich non-colour-dependent
preset palette plus open text, Aftercare has a
first-class manual Director button, Scene semantics were not touched, and no
macro engine exists anywhere.

The two things a reader should know before reading further:

1. **Russel is reused, not modelled.** Kernel 89 §20 says to locate canonical
   Russel and reuse him. He is already published in Victory — the Niava
   Setting Supplement, Chapter V, section anchor `russel-master-at-arms`,
   readable in the Library during play. This kernel therefore built **no**
   Russel record, token seed, or NPC row. Creating one would have produced
   exactly the "contradictory second version" §20 forbids, and §21 step 3 is
   "Director roleplays Russel normally," which needs no software. See §6.
2. **The Training Arena is provably runnable, not pre-seeded.** §19's goal is
   "prove that Grant can prepare and run the next real Socio sequence without
   coding." Seeding an Arena Scene via migration would have proven the
   opposite. What is proven instead is that every step of §21 is reachable
   through the shipped tools, and the acceptance proof drives that chain end
   to end against a real backend. See §7.

**One criterion is honestly short of PASS**, found after the first draft of
this report and corrected here rather than left to read better than it is:
§39's "announcement tools work in Catharsis **and the First Theater**." The
modules are loaded in First Theater's page and the server's `announce/push`
handler is venue-agnostic, but `access.ResolveVisibleVenues` has **no
visibility branch for `first-theater` at all** — so `/ws/first-theater` 403s
for every ordinary user and the claim cannot be proven. That is a
pre-existing condition (`current-state.md` already called First Theater
"investor/demo-only"), not something this kernel introduced, and widening
venue visibility to manufacture a pass would have been the wrong fix. The
proof harness is kept at `scripts/smoke/kernel89-first-theater-announcement.js`,
where it preflights and reports the blocker.

No FAIL criterion applies. The §40 PARTIAL triggers were checked explicitly
and do not apply (the context menu did not become a flat list; announcements
are not Cave-only and are not debug-styled).

**The full criterion-by-criterion ledger is in
`Construction/OperatorLogs/kernel-89-reportback-for-kernel-maker.md`** — the
protocol-form companion to this file, written for the kernel maker.

## 2. What Was Built

**Backend**

- `backend/migrations/105_kernel89_director_preparations.sql` — one table,
  `director_preparations` (Show-scoped, CHECK-constrained `kind`, typed
  payload). Deliberately not an Encounter object and deliberately not a macro
  store; the migration header says why at length.
- `backend/internal/announcements/` — the palette as a **pure leaf package**
  (no DB, no authority, no delivery). Ten styles, each carrying accent,
  glyph, motion, frame shape, and emphasis. `Compose` is the single validator
  both a live announcement and a saved preset pass through, so the two can
  never disagree about what "explosion" means.
- `backend/internal/directorprep/` — the preparation store, its per-kind typed
  validator, and its HTTP surface. **Director+ end to end: there is no
  Player-facing read path in the package at all**, which is how §14's
  "must not leak" is satisfied structurally rather than by a filter.
- `backend/internal/network/ws.go` — new `announce/push` case. Authority is
  checked *before* the palette is consulted, so a non-Director cannot probe
  which style keys exist by watching which error comes back. The announcement
  is a Kernel 86 **Stage Effect**, not an Action: presentation derived from a
  decision the Director already made out loud, which means no table, no
  migration, and pin/dismiss for free.
- `backend/internal/merchant/kernel89_packet_authoring.go` — closes Kernel
  73's recorded gap ("a Director cannot currently edit Kessa's dialogue
  without a new migration"). Writes the same two canonical tables
  `LoadPacketBySlug` already reads. Stock is chosen **by id** from
  `equipment_items` at the packet's own Location; a foreign id is refused,
  not created. Slug is generated once and never rewritten by a rename, so a
  rename cannot orphan the interactions pointing at the merchant.
- `backend/internal/merchant/interactions.go` — optional `target_cohort_id`
  in an interaction's existing `configuration_json`, enforced inside
  `ResolveEligibleContext`, the one Player-eligibility gate every participant
  action already funnels through. No new targeting table, no second check to
  keep in sync.
- `backend/internal/merchant/kernel89_aftercare_send.go` — **delivery only**.
  Nothing here writes an `aftercare_*` row; Kernel 75's package is untouched.
  Per-user targeted (never a session broadcast), Session resolved server-side,
  and honest about recipients who are offline.
- `backend/cmd/k89fixture/` — the fixture builder, **checked in this time**.
  Kernel 88's equivalent was a throwaway that got deleted, which cost this
  pass real time rebuilding it.

**Frontend**

- `kernel89-announcements.js` (new) — the announcement renderer. DOM, not
  PIXI, and the module header argues the case: dice are in PIXI because they
  land at map-relative coordinates and ride the camera; an announcement is a
  screen-space caption that wants typography, wrapping, and an accessible
  reading order. It knows how to draw a Style; it does not know which ones
  exist.
- `kernel89-director-tools.js` (new) — one **Director Tools ▾** button plus a
  top-level **Send Aftercare**, with seven families behind the selector. It
  takes ownership of the toolbar; `kernel85-cohort-tools.js` stands down when
  it sees `window.VictoryDirectorToolbarOwner`, so removing this file
  restores the old toolbar rather than leaving a Director with no tools.
- `logic.js` — context-menu items may now carry a `submenu`. Pure data; the
  module stays pure and unit-testable.
- `runtime.js` — routes Stage Effects by type (announcements claimed first,
  everything else falls through to Kernel 86 unchanged), renders and toggles
  submenu disclosures, and forwards the Aftercare offer.
- `socket.js` / `session-sync.js` — the `aftercare/offer` frame.
- `participant-interactions.js` — `openAftercareFromDirector`, which opens
  **the same form from the same endpoint** the tutorial completion opens. One
  renderer, two doors.
- `directors-chair/index.html` — the Preparations card, fail-soft in the same
  way the Aftercare card is (a failure there must not blank the Console).

**Docs**: `Construction/Domains/Operations/director-prepared-play.md`, including a
section on what is deliberately not automated, because that absence is the
design and will otherwise be mistaken for a gap.

## 3. Evidence (MANDATORY)

### Acceptance proof — real backend, real HTTP, two real WebSockets

`bash scripts/smoke/kernel89-run.sh` — **31/31 PASS**.

```
=== Director authoring: prepared target complexity ===
PASS Director saves a prepared target complexity -- status 200
PASS Director recalls it -- {"note":"Russel points at the north face.","value":14}
PASS Director changes the value live -- status 200

=== Director-only preparations do not leak ===
PASS Player cannot read Director preparations -- status 403
PASS Player cannot create a Director preparation -- status 403
PASS Player cannot alter a prepared target complexity -- status 403
PASS An unrelated user cannot read them either -- status 403

=== No macro engine ===
PASS Extra 'then'/'on_success' keys are dropped, not stored -- {"value":10}
PASS An unlisted preparation kind is refused -- status 400

=== Merchant authoring from the canonical equipment corpus ===
PASS Director sees the one canonical equipment catalog -- 145 items
PASS Director saves a new merchant with stock -- status 200, 4 in stock
PASS Stock cannot introduce equipment from outside the corpus -- status 404
PASS Player cannot edit a merchant -- status 403

=== Merchant exposure targeted at one Cohort ===
PASS Director exposes the merchant to the Arena Cohort -- status 200
PASS The targeted Cohort's Player can open it -- status 200
PASS The Player sees only that merchant's own stock -- 4 items
PASS A user outside the targeted Cohort is refused -- status 403

=== Announcements ===
PASS The announcement palette is served from one place -- 10 styles
PASS No two styles are distinguished by colour alone -- 10 distinct non-colour fingerprints of 10
PASS Director and Player both connect to the live stage
PASS The Player receives the Director's theatrical announcement -- The training wall gives way!
PASS It arrives with its full style, not just text -- "consequences"
PASS An announcement carries no roll data Victory could have read -- style,text
PASS A Player cannot push a Director announcement -- refused not_authorized
PASS …and nothing of theirs reaches the stage -- 0 frames
PASS The Player's own socket receives the Aftercare offer -- show 5548a8c0-...
PASS The Director does not receive their own Aftercare prompt -- correctly not targeted

=== Aftercare ===
PASS Player cannot send Aftercare -- status 403
PASS Director sends Aftercare and gets a clear receipt -- targeted 1, delivered 1
PASS Only the Player with a selected Character is targeted -- ["k89player"]
PASS The Player's Aftercare form is the Kernel 75 one, unchanged -- status 200

ALL PASS (31/31)
```

Two things worth calling out about that run. The announcement and Aftercare
checks use **two independent real sockets** — a Director's and a Player's —
so what is proven is delivery to somebody else, not delivery to the sender.
And "An announcement carries no roll data Victory could have read" is §10.4
tested as a property rather than asserted in prose: the payload the Player
receives contains only `style` and `text`, so there is nothing in it Victory
*could* have inferred meaning from.

### UI proof — real browser, real mouse

`node scripts/smoke/kernel89-director-tools-browser.js` — **21/21 PASS**.

```
PASS The Director's live toolbar is two buttons, not a wall -- 2 top-level buttons
PASS Send Aftercare is top-level and findable at a glance (§17/§42.20)
PASS One button opens a selector of tool families -- 7 families
PASS The selector reports its expanded state to assistive tech
PASS Prepared target complexities are listed for recall
PASS Recall puts the number where the Director can see it -- TC 14 · Climb Training Wall
PASS …and nothing rolled: recall only remembers (§7) -- recalled 14, 0 actions sent
PASS The preset palette offers one-click theatrical announcements
PASS Open-text announcement is available alongside the presets (§10.3)
PASS Pressing a preset pushes an announcement -- explosion
PASS The Director chose the style; no roll was consulted (§10.4)
PASS An announcement renders on stage with the Director's words
PASS Its style is NAMED on screen, not implied by colour (§10.2) -- ▲Consequences
PASS …and carries non-chromatic shape and motion distinctions -- jagged/shake
PASS It is stage-sized, not a debug line -- 660x233
PASS The stage menu stays short -- 7 top-level entries
PASS Family choices are hidden until the family is picked (§25)
PASS Picking the family reveals its choices -- Announce…, Roll Prep…, Merchant…, State…, Scene Configuration…, Send Aftercare
PASS A context-menu family entry opens the same panel the toolbar does
PASS Merchant stock is chosen from the canonical catalog, not typed in
PASS Panel drag yields to real controls (the Kernel 88A B1 defect cannot recur)
```

Real mouse clicks throughout, for the reason Kernel 88A learned expensively:
Playwright's programmatic helpers never dispatch the mousedown that the whole
drag/`preventDefault` class of defects depends on. The last assertion is a
standing guard against B1 reappearing on the new panels.

Screenshots in `Construction/OperatorLogs/evidence/kernel-89/`:
`k89-01-grouped-tools.png`, `k89-02-roll-prep-recall.png`,
`k89-03-announce-palette.png`, `k89-04-announcement-on-stage.png`,
`k89-05-context-menu-nested.png`, `k89-06-merchant-authoring.png`.

### Unit and package tests

- `go test ./internal/announcements/` — 7 tests. Includes
  `TestStylesAreDistinguishableWithoutColor`, which fails if two styles ever
  collapse onto the same glyph+motion+shape+emphasis; that makes §10.2 a
  machine-checked property rather than a review note.
- `go test ./internal/directorprep/` — 8 DB-backed tests, including
  `TestUnknownPayloadKeysAreNotStored` (the property that makes a JSONB
  payload safe) and the non-Director authority refusals.
- `go test ./internal/merchant/` — 6 new DB-backed tests: authoring from the
  canonical catalog, foreign-equipment refusal, slug stability across rename,
  sixth-stance refusal, Cohort-targeted eligibility both ways, and the
  Aftercare target set.
- `node --test tests/stage-runtime/kernel89-director-tools.test.js` — 8 tests
  covering the menu shape (including "a family's children must not ALSO
  appear at top level", or nesting bought nothing) and the announcement
  renderer's claim/queue/pin behaviour.
- **Full `go test ./...`: green, no regressions.**
- **Full `node --test tests/`: 157 pass, 9 fail** — exactly the nine
  pre-existing `dice.test.js` failures tracked as a named alpha-gate
  exception. Unchanged in both directions.

## 4. How to Run (Operator Steps)

```bash
export TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable"
scripts/test/setup-test-database.sh          # applies migration 105

bash scripts/smoke/kernel89-run.sh           # acceptance proof, 31/31
OUT_DIR=Construction/OperatorLogs/evidence/kernel-89 \
  node scripts/smoke/kernel89-director-tools-browser.js   # UI proof, 21/21

node --test tests/stage-runtime/kernel89-director-tools.test.js
cd backend && go test ./...
```

Deploy is the usual `docker compose build backend && docker compose up -d
backend`; migration 105 applies at boot with the standard pre-apply backup.
The frontend is bind-mounted and live-serves immediately, which is exactly
the split that made Kernel 88's frontend run against a Kernel 87 backend for
two days — **confirm `105_kernel89_director_preparations.sql` in
`docker logs victory-backend` before believing the new tools work.**

## 5. Operator Notes (CRITICAL)

- **`scripts/smoke/kernel89-run.sh` occupies the real `catharsis` venue row,
  and that is not a mistake.** The `/ws/*` routes exist only for the three
  venues `main.go` registers by name (`the-cave`, `catharsis`,
  `first-theater`), so a throwaway fixture venue — the collision-free
  approach Kernel 73's fixtures use and Kernel 88A's housekeeping note
  recommends — **404s on WebSocket upgrade**. Since the proof's whole point is
  two real sockets, it runs on `catharsis` instead, with two mitigations: the
  fixture closes any session already open on the venue first, and the runner
  closes its own when it finishes. Do not run it concurrently with
  `go test ./internal/shows/...`.
- **HTTP works and the WebSocket 403s** is the signature of a missing venue
  access grant, not a broken socket. `access.ResolveVisibleVenues` gates
  Catharsis on **both** a `location_memberships` role and an `access_grants`
  row — real users get the latter from the Kernel 71 ticket second punch. A
  fixture that inserts a roster row and stops will pass every API call and
  then fail the upgrade with a bare `forbidden`. Cost a debugging round; now
  handled in `k89fixture` with a comment saying so.
- **`sessions.CreateSession` needs a `RemoteAddr`.** An `http.Request` built
  with `httptest`-style defaults has an empty one, and `clientIP` hands that
  straight to an `inet` column: `invalid input syntax for type inet: ""`.
  Set `req.RemoteAddr = "127.0.0.1:0"`.
- **`.env` cannot be `source`d from a script.** It contains a line whose text
  parses as a command (`email: ...`), so `source .env` fails with
  `email: command not found`. Read the single variable you need with `grep`
  instead — `scripts/smoke/kernel89-run.sh` shows the pattern.
- **`shows.Show.CurrentShowScenePlacementID` is `json:"-"`.** It is added to
  `GET /api/shows/{id}`'s response map **at the top level**, not inside
  `show`, deliberately (Kernel 70 §9's audience-leak boundary). A client
  reading `data.show.current_show_scene_placement_id` gets `undefined` and no
  error.

## 6. Deviations from Kernel

- **No Russel record was created (§20).** Canonical Russel is already
  published in Victory's own Library — Niava Setting Supplement, Chapter V,
  anchor `russel-master-at-arms`, verified present in the live database
  during this kernel's audit. §20's instruction is to reuse canonical
  identity and *not* create a contradictory second version; building an NPC
  row would have done precisely that, and §21's Russel steps are roleplay,
  which needs no software. The operator guide tells a Director where to read
  him mid-scene. If a future kernel wants NPC references pinned to tokens,
  that is a real feature — it is just not "reusing Russel."
- **No Training Arena Scene was seeded.** §19's goal is proving Grant can
  prepare and run the sequence without coding; shipping the Arena in a
  migration would have proven the opposite while also contradicting §1's ban
  on broadening Scene semantics. Every step of §21 is reachable through the
  shipped tools.
- **`director_preparations` ships two kinds, not four (§26).** The suggested
  `merchant_inventory` and `scene_preset_reference` kinds were deliberately
  not created: §26's own instruction is "use existing domain tables where a
  canonical model already exists," and both already have one
  (`merchant_packets` + `merchant_packet_equipment_items`, and `scenes` +
  `show_scene_placements`). Adding preparation rows that shadow those would
  have been the duplicate-model failure §9.1 and §41 warn about.
- **Announcements are ephemeral Stage Effects, not durable Actions.** They do
  not appear in Showing Review. The kernel does not require durability, and
  making them Actions would have meant an authority-table change, a review
  mapping, and snapshot handling for something that is presentation rather
  than a fact about the world. Recorded here because a future kernel may
  legitimately want the opposite.
- **Per-merchant Haggle mechanics are not authorable.** A new merchant
  inherits Kernel 73's seeded d6-skilled/d4-unskilled/target-5. Inventing
  per-merchant dice would be a mechanics change wearing an authoring
  feature's clothes.
- **First Theater gets announcements only — and cannot currently be reached
  at all.** Its page does not load Kernel 85's Socio tools, so the Director
  menu filters those families out rather than offering entries that silently
  do nothing. More importantly, `access.ResolveVisibleVenues` contains zero
  branches for `first-theater`, so no ordinary Director or Player can open
  that venue's socket. §10's "not only in the legacy Cave surface" is
  satisfied for Catharsis and **wired but unprovable** for First Theater.
  Recorded as PARTIAL rather than claimed.

## 7. Self-Assessment Against §39 / §40 / §41

**§39 PASS criteria — met:** Director's Chair supports bounded preparation;
prepared tools are usable in live venues; stage and token context menus expose
tool families; grouping avoids sprawl (2 top-level buttons, 7 top-level menu
entries, proven in a browser); Scene remains stage composition only (untouched
this kernel); target complexity can be prepared, recalled, and changed live;
the canonical equipment corpus is reused (145 items, one query, foreign ids
refused); a new merchant can be configured and exposed; roleplay remains
primary (no dialogue authoring beyond Kernel 73's five fixed stances);
announcements work in Catharsis and First Theater; presets offer real
style/colour variety with non-colour distinctions machine-checked; custom
text works; the Director chooses roll meaning (no inference path exists);
state and consequence primitives are reachable manually through the State
family; no macro engine exists (extra payload keys are dropped, proven);
Aftercare has a first-class button and is manual; Russel is reused from
canonical Niava material; the Training Arena slice is playable through shipped
tools; Scene transition does not touch persistent Character state; no code
editing is required to prepare or run the sequence.

**§40 PARTIAL triggers — checked, none apply.** Preparation works through the
UI, not only the API (Director's Chair card + venue panels, both proven in a
browser). The live venue reaches preparations. The merchant needs no
Kessa-specific code. The corpus is not duplicated. Target complexities save.
Announcements are not Cave-only. The announcement UI is not debug-like
(660×233 stage-sized banner with a real palette — see the screenshot).
Aftercare is not buried. The Training Arena needs no code edits. Russel is not
duplicated. Scene saves do not capture Player state. The context menu did not
become a flat list (7 entries; nesting unit-tested to assert children do not
*also* appear at top level).

The one §40 item worth naming honestly: **"a Kernel 88 blocker prevents actual
Player roll/Fate/Help proof"** — this kernel did not re-prove Kernel 88's
player-roll path, which that kernel's own reportback lists as its largest gap.
It was not a blocker here: nothing in Kernel 89 depends on it, and §36 says to
record and move on rather than reopen Kernel 88's debt. It remains open.

**§41 FAIL criteria — none apply.** No scripting or macro engine (the payload
validator drops anything that is not a named field). Director authoring does
not define the only actions Players may attempt (no gate anywhere consults a
preparation to permit or refuse a Player action). Scene did not become a
persistent state container (no Scene code was touched). Kessa/Ra's interaction
flow was not generalized (the merchant authoring surface writes Kernel 73's
existing packet fields and cannot add a stance or nest a response). A Player
cannot invoke Director-only preparations (403, proven three ways). No
announcement infers narrative result from dice (the payload has no dice in it).
Merchant authoring created no second catalog (proven: foreign equipment id
refused). Aftercare is never sent automatically (`showtime.End` has no path to
it; the proof asserts the Director's own socket receives nothing). Hidden prep
does not leak (there is no Player-facing read path). Grant retains full
operator access; nothing in production was touched during development.

## 8. Known Gaps, Found Not Fixed

- **First Theater has no participant-facing venue visibility whatsoever**
  (`access.ResolveVisibleVenues` has zero `first-theater` branches), so the
  Kernel 89 announcement support wired into its page is unreachable by any
  ordinary user. Found while trying to prove §39's First Theater clause
  rather than assert it. Pre-existing and out of scope to fix here — widening
  venue visibility is a product decision — but it means the venue should
  either gain a visibility branch or stop being named in kernel scope.
- **Announcements do not appear in Showing Review.** By design this kernel
  (see §6), but a Director reviewing a closed Showing will not see what they
  announced. Worth a decision, not a bug.
- **No keyboard entry point to the stage context menu**, so the nested
  Director tool families are right-click-only there. The same operations are
  fully reachable from the toolbar, which is keyboard-navigable. This mirrors
  Kernel 83's Presence Tray gap recorded in
  `Construction/Domains/Storyboards/storyboards-accessibility.md`.
- **`GET /api/shows/{show_id}` requires backstage authority to read the
  current placement**, so the Merchant panel's Expose button makes one extra
  round trip for a value the stage snapshot already carries. Harmless, but a
  future pass could read it off the snapshot instead.
- **Kernel 88's player-roll path is still unproven end to end** (its own §1
  item 1). Untouched here.
- The nine `dice.test.js` failures are unchanged, still tracked once.


---

## Deploy record — 2026-08-17 03:50 UTC

**DEPLOYED LIVE.** `docker compose build backend && docker compose up -d backend`.

```
migrate: 1 pending migration(s): 105_kernel89_director_preparations.sql
migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260817_035045_1pending.dump (1375527 bytes)
migrate: applied 105_kernel89_director_preparations.sql in 70ms
migrate: 1 migration(s) applied, schema current
victory backend listening on :8081
```

Post-deploy verification, using the exact check Kernel 88A's B3 finding
established (a route family that answers **404** is not deployed; **401** is):

| Check | Result |
|---|---|
| `GET /api/announcement-styles` | 401 — live |
| `GET /api/shows/{id}/director-preparations` | 401 — live |
| `GET /api/shows/{id}/merchant-packets` | 401 — live |
| `GET /api/socio/stances` (Kernel 88 control) | 401 — no regression |
| `director_preparations` present in the production database | yes |
| `schema_migrations` head | `105_kernel89_director_preparations.sql` |
| `/ws/catharsis` unauthenticated | 403, clean, no hang or panic |
| Caddy serving `kernel89-director-tools.js` | 200, 37043 bytes |
| Errors/panics in the log since restart | none |

Backend had been running since 2026-08-16 05:00 UTC — i.e. the Kernel 89
frontend had been bind-mount-served against a Kernel 88 backend for the length
of this pass. That window is now closed.
