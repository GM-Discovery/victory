# Kernel Report Back — Kernel 79A: Skill Directory + Character Skill Rule-Links

## 1. Status

**PASS, DEPLOYED LIVE 2026-08-05.**

This is Kernel 79A, the follow-up scoped out of Kernel 79 Phase 1's
reportback: the reusable eWrite directory abstraction (kernel spec §5),
proven through its first real instance, the Skill Directory, plus the
Character-mechanics rule-link half of spec §6.1 (skills only — attributes,
values/resources, health systems, and actions/reactions remain deferred;
see Deviations). Grant confirmed this exact scope ("Kernel 79A only") over
the broader remaining Kernel 79 goals (ruleset-wide navigation,
hierarchical export, tutorial/Cue/index-card links, Brevo) before this
session began.

Baseline: clean tree at `3320721` (character visibility gating for
Library), all containers up before this work started. Everything from this
kernel is uncommitted for Grant's review, per house practice.

---

## 2. What Was Built

### Reusable directory abstraction (Goal B)

- New tables `ewrite_directories` / `ewrite_directory_entries` (migration
  087) — a directory is scoped to a Ruleset collection, `directory_type` is
  a CHECK enum (`'skill'` today, new arms add future directories per spec
  §5.5) so no new table is needed for actions/health systems/equipment
  later. An entry's target (`target_publication_id`/`target_section_id`)
  is a separate nullable pair, not `ewrite_object_links` — a directory
  entry is compact-index metadata pointing *at* the canonical rule (spec
  §5.4), not an object binding with an owning row of its own.
- `backend/internal/ewrite/directories.go`: `ListDirectoriesForCollection`,
  `ListDirectoryEntries` (search + category filter, visibility-safe target
  resolution), `SetDirectoryEntryLink` (Crew+ curation, dual authority:
  `CanAuthorInScope` at the directory's own location AND
  `CanEditPublication` on the target — mirrors `SetEquipmentItemRuleLink`'s
  precedent so an author can't wire another production's draft into a
  public directory), `RuleLinksForCharacterSkills`.
- **Link-status resolution is per-requester, not per-status-only** (unlike
  the equipment rule-link precedent, which only filters
  `status='published'`): `ListDirectoryEntries` runs every target through
  `CanReadPublication` for the actual caller. An entry whose target exists
  but is unreadable comes back `"hidden"` with the publication/section
  fields stripped entirely — the directory must never leak a hidden
  target's title (spec §5.3). Three states: `unlinked` (no target
  curated), `hidden`, `linked`.
- HTTP: `GET /api/ewrite/directories?collection_id=`,
  `GET /api/ewrite/directories/{directory_id}/entries?search=&category=`,
  `PUT /api/ewrite/directory-entries/{entry_id}/link` (Crew+; empty
  `publication_id` clears the link back to `unlinked`).

### Skill Directory (Goal B, spec §5.2)

- `EnsureSkillDirectory` (`seed.go`), boot-time, create-if-absent-only per
  entry (same idempotency rule as `EnsureCanonicalSocioManuscript` — a
  Crew+ member's manual re-curation is never reverted by a redeploy).
  Seeds one entry per `characters.Chapter4Skills` catalogue row (100
  skills) under the Socio ruleset.
- **Auto-matched to real Core Rulebook sections, not left empty for manual
  curation.** The manuscript's own catalogue appendix renders every skill
  as a level-4 heading `"{Name} ({AttributeName})"` — verified directly
  against the imported sections table before writing the matcher, not
  assumed. **98 of 100 entries auto-linked on live deploy.** The 2 misses
  are real, pre-existing content drift between the Go catalogue
  (`chapter4_skills.go`) and the manuscript, not a bug in the matcher —
  see Known Issues.
- **Import-cycle avoidance**: `ewrite` cannot import `characters` directly
  (`internal/assets` already imports `ewrite` for Kernel 79's
  image-visibility fix, and `characters` needs `ewrite.RuleLink` for the
  reverse direction below — importing `characters` from `ewrite` would
  close the cycle). `EnsureSkillDirectory` instead takes
  `[]ewrite.SkillCatalogueEntry`; `main.go`, which already imports both
  packages, builds that slice from `characters.Chapter4Skills` and passes
  it in.

### Character-sheet skill rule-links (Goal 6.1, skills only)

- `ewrite.RuleLinksForCharacterSkills` resolves Skill Directory links for
  a set of catalogue skill IDs in one query — same shallow
  `status='published'`-only filter as `RuleLinksForEquipmentItems` (this
  is deep-link metadata on an already-authority-gated payload; the reader
  route re-enforces real access when the link is opened).
- Wired into both shared character-sheet builders:
  `CharacterWorkbookPage.SkillRuleLinks` (Greenroom Mechanics page, via
  `LoadCharacterWorkbookView`) and `CharacterSheetProjection.SkillRuleLinks`
  (venue right tray, via `ProjectCharacterSheet`) — both keyed by
  `skill_id`, keeping Kernel 59A's "one shared projector" rule intact.
- Frontend: Greenroom's Mechanics page renders a "View rule →" link next
  to each trained skill when one resolves (opens the Library reader, new
  tab, exact section). The venue right tray's backend data is populated
  but **not** wired to a UI element this pass — see Deviations.

---

## 3. Evidence (MANDATORY)

### Unit/integration tests

```
$ cd backend && GOCACHE=/tmp/victory-gocache go build ./... && go vet ./...
(clean, no output)

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty.

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  go test -count=1 ./...
ok  	victory/backend/cmd/victory	0.016s
ok  	victory/backend/internal/access	2.148s
ok  	victory/backend/internal/actions	0.167s
ok  	victory/backend/internal/aftercare	0.315s
ok  	victory/backend/internal/assets	1.698s
ok  	victory/backend/internal/characters	0.568s
ok  	victory/backend/internal/commands	0.013s
ok  	victory/backend/internal/cues	7.295s
ok  	victory/backend/internal/dice	0.005s
ok  	victory/backend/internal/ewrite	8.885s
ok  	victory/backend/internal/identity	9.928s
ok  	victory/backend/internal/merchant	4.500s
ok  	victory/backend/internal/messages	0.012s
ok  	victory/backend/internal/migrate	0.100s
ok  	victory/backend/internal/network	4.006s
ok  	victory/backend/internal/participation	0.535s
ok  	victory/backend/internal/playerprofile	0.924s
ok  	victory/backend/internal/playerrelationships	0.011s
ok  	victory/backend/internal/profiles	0.009s
ok  	victory/backend/internal/ratelimit	0.009s
ok  	victory/backend/internal/scenes	6.017s
ok  	victory/backend/internal/showings	0.013s
ok  	victory/backend/internal/showruns	3.060s
ok  	victory/backend/internal/shows	3.953s
ok  	victory/backend/internal/showtime	1.216s
ok  	victory/backend/internal/storysofar	0.021s
ok  	victory/backend/internal/thirdplace	2.601s
ok  	victory/backend/internal/tickets	2.251s
ok  	victory/backend/internal/venues	0.010s
ok  	victory/backend/internal/world	1.993s
(full repo, zero failures, against a freshly reset victory_test)
```

New tests added:
- `backend/internal/ewrite/directories_dbtest_test.go` — 4 tests:
  link-status resolution (unlinked/hidden/linked, hidden leaks nothing,
  search matches name and alias substrings, category filter);
  `SetDirectoryEntryLink` dual-authority (Cast denied outright, Crew+
  denied when they lack edit authority on the *target* publication even
  though they can curate the directory itself, Crew+ succeeds on their own
  publication and resolves `linked` pre-publish since edit authority
  implies draft-read); clear-link and cross-publication section-mismatch
  rejection; `RuleLinksForCharacterSkills` draft-vs-published resolution.
- `backend/internal/ewrite/seed_skill_directory_dbtest_test.go` — proves
  `EnsureSkillDirectory` auto-matches two real catalogue skills
  (`Alertness`, and the disambiguated `Intimidation (Physical)`) to their
  actual manuscript sections, that a catalogue entry with no matching
  heading comes back `unlinked` rather than erroring, and that a second
  boot never reverts a Crew+ member's manual re-curation of that unlinked
  entry (the idempotency regression pattern from Kernel 79 Phase 1's own
  bug, applied here preemptively).

**A real test-authoring mistake was caught by the suite itself, not
shipped**: the first draft of `TestRuleLinksForCharacterSkills` reused a
real `characters.Chapter4Skills` ID (`SKILL_AWARENESS_INSIGHT`) for a
throwaway test fixture. Because `RuleLinksForCharacterSkills` resolves
across *all* `'skill'`-type directories (not scoped to one), it collided
with the already-boot-seeded canonical Skill Directory's real entry and
made the draft-vs-published assertion fail nondeterministically. Fixed by
using a synthetic `SKILL_TEST_RULELINK_INSIGHT` ref instead — flagged here
because it's a real, reusable trap for any future test that touches a
`'skill'`-type directory on a database where the real binary has already
booted (which `setup-test-database.sh` always does).

### Kernel 64 DB isolation proof

```
$ unset TEST_DATABASE_URL
$ go test -count=1 ./internal/ewrite/...
--- FAIL: TestDirectoryEntryLinkStatusResolution (0.00s)
    dbtest: TEST_DATABASE_URL is required for database-touching tests
[... remaining new and pre-existing dbtest tests in this package, same hard failure ...]
FAIL
```

### Live deploy

```
$ docker exec -i victory-postgres pg_dump -U victory victory > \
    /tmp/victory-pre-kernel79a-deploy-backup.sql   # manual backup, extra to
                                                     # the automatic one below

$ docker compose build backend && docker compose up -d backend
 Container victory-backend  Recreated
 Container victory-backend  Started

$ docker logs victory-backend --since 40s
2026/08/05 21:53:39 migrate: 1 pending migration(s): 087_kernel79a_ewrite_skill_directory.sql
2026/08/05 21:53:40 migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260805_215339_1pending.dump (1158492 bytes)
2026/08/05 21:53:40 migrate: applied 087_kernel79a_ewrite_skill_directory.sql in 121ms
2026/08/05 21:53:40 migrate: 1 migration(s) applied, schema current
2026/08/05 21:53:40 victory backend listening on :8081
(discord gateway reconnect lines, unrelated, omitted)
```

Live database, before this kernel's deploy and right now — zero drift on
existing rows:
```
before: users=4  location_memberships=1  locations=2  ewrite_publications=3
after:  users=4  location_memberships=1  locations=2  ewrite_publications=3
```

Skill Directory, verified directly against `victory` after deploy:
```
$ psql -d victory -c "SELECT d.title, d.slug, COUNT(e.id) entries, COUNT(e.target_publication_id) linked
                       FROM ewrite_directories d LEFT JOIN ewrite_directory_entries e ON e.directory_id=d.id
                       GROUP BY d.id, d.title, d.slug;"
      title      |  slug  | entries | linked
------------------+--------+---------+--------
 Skill Directory  | skills |     100 |     98
```

### Full end-to-end live proof (real HTTP, real auth, real character)

No browser automation tooling exists in this environment (same constraint
as every prior kernel); substituted a real disposable user + session +
character card created directly against the live database (field guide's
documented technique for a fixture account with a known credential),
driven entirely through the real deployed HTTP API, then fully deleted
afterward:

1. `GET /api/ewrite/directories?collection_id=<socio-ruleset>` →
   `{"directories":[{"title":"Skill Directory","entry_count":100,...}]}`
2. `GET /api/ewrite/directories/<id>/entries?search=alert` → one match,
   `Alertness`, `link_status:"linked"`, resolved to the real Core Rulebook
   `alertness-awareness` anchor.
3. `GET /api/ewrite/directories/<id>/entries?category=Might` → exactly 10
   entries (the real catalogue's Might skill count).
4. Created a disposable character, gave it the `Alertness` skill directly
   (bypassing the Discord-only `/char add skill` command surface, which
   this environment can't drive), then
   `GET /api/character-workbooks/<card_id>` → the `mechanics` page's
   `skill_rule_links` map correctly resolved
   `{"SKILL_AWARENESS_ALERTNESS": {"publication_id":..., "section_anchor":"alertness-awareness", ...}}`.
5. Unauthenticated `GET /api/ewrite/directories` → `401`.
6. Cleanup verified: `users`, `location_memberships`, `character_cards`
   counts identical before and after (Grant's own pre-existing character
   `"I"` confirmed untouched by name/ID).

### Static checks

```
$ git diff --check
(clean, no output)

$ node --check <extracted inline script from library/index.html>
$ node --check <extracted inline script from library/skill-directory.html>
$ node --check <extracted inline script from greenroom/index.html>
(all clean)
```

---

## 4. How to Run (Operator Steps)

Already deployed live — nothing to run. For a fresh install, the Skill
Directory seeds automatically at boot alongside the Socio manuscripts.

To browse it: **Library → Socio: Stories of Us → "Skill Directory" (under
the ruleset, next to the three Series)** → search/filter/alphabetical
browse. To see a Character's own rule links: Greenroom → open a character
→ Mechanics → any trained skill with a resolved link shows "View rule →".

---

## 5. Operator Notes (CRITICAL)

- **2 of 100 skills are unlinked by design, not by bug** — real content
  drift between `chapter4_skills.go` and the manuscript (see Known
  Issues). Worth a deliberate decision (fix the manuscript heading, fix
  the catalogue name, or just hand-link via the new `PUT
  /api/ewrite/directory-entries/{entry_id}/link` endpoint) rather than
  silent papering-over.
- The venue right-tray (Cave/Catharsis/First Theater) skill roll panel now
  carries `skill_rule_links` in its backend payload but has **no UI**
  reading it yet — deliberate scope cut, see Deviations. The data is
  there for whichever kernel wants to add the affordance.
- `TEST_DATABASE_URL` is still not set in `.env` — export it manually per
  Kernel 64/78/79 precedent.

---

## 6. Blockers & Workarounds

**BLOCKER:** No browser automation tooling in this environment (unchanged
constraint since Kernel 61).

**WORKAROUND:** Same substitution used since Kernel 65/71/74: a real
disposable account + session + character driven through the actual
deployed HTTP API end-to-end (§3 above), plus 4 new dbtest integration
tests against a real database with real users/roles/publications for the
authority-sensitive parts (link-status hiding, dual-authority curation).

**OPERATOR ACTION REQUIRED:** Grant should do one visual pass when
convenient: open Library → Skill Directory, confirm the alphabetical
browse/search/attribute-filter UI reads cleanly and a few "View rule"
links land on the right section; open a character's Mechanics page in
Greenroom with a trained skill and confirm the same.

---

## 7. Deviations from Kernel

Confirmed with Grant before implementation: this pass is **Kernel 79A
only** (Skill Directory + Character skill rule-links), the narrowest slice
Kernel 79 Phase 1's reportback recommended, not the full remaining Kernel
79 spec. Everything else Kernel 79 Phase 1 already deferred remains
deferred: health-system links (still no data model to link to), tutorial/
Cue/index-card links, hierarchical export, ruleset-wide Library
navigation/search, Brevo verification.

Within the 79A slice itself, two further narrowings:

- **Character-mechanics links are skills only.** Spec §6.1 also lists
  attributes, values/resources, and actions/reactions — none of those have
  an existing per-Character data row equivalent to `character_skills` to
  hang a rule link off, and building that data model would have expanded
  this into the health-systems-sized deferral Kernel 79 Phase 1 already
  recorded. Skills were the one mechanic both already modeled
  (`character_skills`, Kernel 60) and explicitly named as the Directory's
  proving ground (spec §5.2).
- **Venue right-tray UI was left unwired.** The backend data
  (`CharacterSheetProjection.SkillRuleLinks`) is populated identically to
  the Greenroom path, but the tray's skill buttons are shared,
  tightly-tested interaction surfaces (`frontend/lib/stage-runtime/
  runtime.js`, styled per-venue in `first-theater/index.html` and
  `catharsis/index.html`, covered by `tests/stage-runtime/kernel74-
  tutorial.test.js` and `kernel75-completion.test.js`). Restructuring the
  flat button list into a row-with-link layout touches three files' worth
  of shared, tested click-handling for a supplementary affordance — the
  Greenroom Mechanics page already satisfies spec's core ask ("Does a
  Character skill open the exact rule?"). Judged not worth the regression
  risk for this pass; the data is ready whenever a future kernel wants the
  UI.

---

## 8. Known Issues

- **2 of 100 Skill Directory entries don't auto-link**, both real content
  drift between `backend/internal/characters/chapter4_skills.go` and the
  imported manuscript, not a matcher bug:
  - `"Breaking / Forcing"` (catalogue, spaces around the slash) vs.
    `"Breaking/Forcing"` (manuscript heading, no spaces).
  - `"Recovery"` (catalogue, Resolve attribute) has no corresponding
    level-4 heading in the manuscript's Resolve appendix at all (9 Resolve
    skills present there, not 10) — genuinely absent, not misnamed.
- `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`'s pre-existing
  flake (Kernel 79 Phase 1's Known Issues) reproduced once during this
  session's first (non-reset) test run, confirming it's still live and
  unrelated to this kernel — resolved by the documented fix (reset the
  test database before a clean run), not touched otherwise.

---

## 8A. Addendum: Kernel 79 §13 Bounded Brevo Check (same session, post-79A)

Performed at Grant's request immediately after 79A shipped, honoring his
recorded Kernel 77 note ("remind me in kernel 78+ to check if I have been
approved yet with Brevo"). Not part of 79A's own scope; recorded here
rather than as a separate reportback since it's a single bounded check,
not a kernel-sized change.

**Method:** a throwaway `go run` program (`backend/cmd/brevo-check-tmp/`,
deleted immediately after use, never committed) constructed the real
`mailer.SMTPMailer` from the live `.env` SMTP config and called
`SendPasswordReset` against one real recipient — isolating the check to
"can we authenticate to Brevo's SMTP relay and hand off a message" without
touching any live account, database row, or HTTP/auth session.

**Result: still blocked, identical failure to Kernel 77.**
```
SEND FAILED: 535 5.7.8 Authentication failed
```
Same error, same signature (an account-level gate, not a configuration
problem — Kernel 77 already ruled out config via three independently-
correct fixes: SMTP login format, a freshly-generated key, and an
IP-allowlist entry Brevo itself confirmed recognizing). No email was
delivered — the SMTP session never got past `AUTH`, so nothing reached
Brevo's outbound pipeline at all. Per spec §13 item 5 ("do not spend the
kernel debugging an external account indefinitely"), no further fixes were
attempted this pass — re-deriving the same three already-ruled-out
explanations would not be new information.

**OPERATOR ACTION REQUIRED (unchanged from Kernel 77):** This needs
resolution on Brevo's side — check the account's approval/review status
directly in the Brevo dashboard (something only Grant can do; Victory only
holds SMTP credentials, not an account-management API key). `victory-
recover` remains the fully-functional break-glass path for every account,
including Grant's own, in the meantime.

---

## 9. Next Recommended Step

Grant's choice from here: (a) hand-link the 2 drifted skills and decide
whether to reconcile the catalogue/manuscript naming generally, (b) wire
the venue right-tray "View rule" affordance now that the backend data
exists, or (c) pick up the next slice of Kernel 79 proper (ruleset-wide
Library navigation is the most self-contained of what's left, per Kernel
79 Phase 1's own deferral notes).

---

## 10. Files Changed / Created

- `backend/migrations/087_kernel79a_ewrite_skill_directory.sql` (new)
- `backend/internal/ewrite/directories.go` (new)
- `backend/internal/ewrite/directories_dbtest_test.go` (new)
- `backend/internal/ewrite/seed_skill_directory_dbtest_test.go` (new)
- `frontend/venues/library/skill-directory.html` (new)
- `backend/internal/ewrite/types.go` (modified — `Directory`,
  `DirectoryEntry`)
- `backend/internal/ewrite/seed.go` (modified — `EnsureSkillDirectory`,
  `SkillCatalogueEntry`)
- `backend/internal/ewrite/http.go` (modified — 3 new directory routes)
- `backend/cmd/victory/main.go` (modified — route registration, boot seed
  call, catalogue bridge)
- `backend/internal/characters/workbook_pages.go` (modified —
  `SkillRuleLinks` on the Mechanics page)
- `backend/internal/characters/character_sheet_projection.go` (modified —
  `SkillRuleLinks` on the shared projection)
- `frontend/venues/library/index.html` (modified — directories rendered
  under their Ruleset in the tree)
- `frontend/venues/greenroom/index.html` (modified — "View rule" link per
  trained skill)
