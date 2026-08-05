# Kernel 79 — eWrite Integration and Rules Navigation

**Status:** READY FOR IMPLEMENTATION  
**Type:** Product integration and navigation kernel  
**Primary tracks:** Victory Core, Socio  
**Secondary tracks:** Anthology, Operational Integrity  
**Planning authority:** `Victory_Canonical_Roadmap_v2.md`  
**Sequence position:** After Kernel 78  
**Feature authority:** Kernel 78 implementation and reportback  
**Authoring system:** eWrite  
**Reading system:** Library

---

## 0. Kernel contract

Kernel 79 turns the successful eWrite foundation into a practical in-play rules-reference system.

Kernel 78 proved that Victory can import substantial Markdown, create hierarchy, render safely, preserve stable anchors, save revisions, prevent silent conflicts, publish with visibility, browse and read in the Library, link equipment to exact sections, and export individual publications.

Kernel 79 now makes that foundation useful across a complete Ruleset and during actual play.

This kernel focuses on:

- organizing the live Socio books;
- ruleset-wide navigation;
- compact reusable directories for dense rule indexes;
- direct links from Character and play surfaces into exact rules;
- image access that follows eWriting visibility;
- Module, Series, and Ruleset export;
- preserving the current writer-focused UI baseline.

This is not another eWrite foundation kernel and not a new Writer's Room redesign.

---

## 1. Locked product decisions

### 1.1 Kernel purpose

> **Kernel 79 — eWrite Integration and Rules Navigation**

Its purpose is to make eWrite practical as an everyday rules-reference and play-assistance system.

### 1.2 Socio hierarchy

```text
Socio: Stories of Us -- Ruleset
├── Core Rulebook -- Series
├── Quickstart -- Series
└── Niava -- Series
```

Below each Series:

```text
Module
└── Publication
    └── Section
        └── Subsection
```

The importer may recommend structure, but it must not automatically turn every heading into a separate Publication without review.

### 1.3 Rules-link priority

Kernel 79 should extend exact-rule linking in this order:

1. Character mechanics and health systems
2. Tutorial actions and choices
3. Index cards
4. Cues
5. Scene elements

Equipment already proves the reusable object-link vertical.

### 1.4 Skill navigation

Socio skills must not require users to scan hundreds of headings in one giant index.

Kernel 79 must create a reusable directory pattern:

```text
Directory
└── compact index entry
    └── exact canonical eWrite section
```

The first implementation is a **Skill Directory**.

It should support:

- skill name;
- alphabetical grouping;
- search;
- filters by attribute, category, or existing Socio classification;
- aliases or alternate terms;
- compact description or metadata;
- exact eWrite destination;
- links from Character mechanics;
- no duplication of the canonical rules text.

The same directory pattern should be reusable later for actions, reactions, Social Stances, health systems, equipment, conditions, oracles, and other large rule indexes.

### 1.5 Image visibility

Images embedded in or attached to eWritings must follow the effective visibility of the publication:

```text
public publication        -> public image
authenticated publication -> authenticated image
Production publication    -> same Production boundary
draft                     -> creator/editors/authorized Crew+ only
```

A copied asset URL must not bypass the publication's access rule.

### 1.6 Hierarchical export

Kernel 79 must support export of:

- one Module;
- one Series;
- one Ruleset.

Exports must preserve hierarchy, Markdown, stable anchors, internal links, authorized images, metadata, and an import/reconstruction manifest.

### 1.7 Anonymous reading

Anonymous public reading remains deferred.

The `public` visibility value may remain in schema, but eWrite content remains login-gated until a separate deliberate kernel creates Victory's first unauthenticated content surface.

### 1.8 UI baseline

Grant has manually improved the Writer's Room index.

The accepted design direction is:

- focus on the writer and active work;
- collection structure remains secondary;
- current color changes are preserved;
- current panel/window arrangement is preserved;
- Library color adjustments may continue.

The builder must treat the current browser-approved state as the new baseline.

---

## 2. Kernel goals

Kernel 79 has eight goals.

### Goal A — Establish the live Socio Ruleset structure

Create or support the intended hierarchy for Core Rulebook, Quickstart, and Niava. Provide tools to curate imported manuscript structure without excessive manual labor.

### Goal B — Build reusable rules directories

Create the first reusable directory abstraction and prove it through Socio skills.

### Goal C — Extend exact-rule links

Connect additional Victory objects to exact eWrite sections.

### Goal D — Enforce image visibility

Close the Kernel 78 embedded-image visibility gap.

### Goal E — Add hierarchical export

Export Module, Series, and complete Ruleset packages without flattening structure.

### Goal F — Improve rules navigation

Make Library navigation useful across a large Ruleset.

### Goal G — Preserve UI and security baselines

Keep the current writer-focused UI and all Kernel 76–78 security properties.

### Goal H — Perform bounded operational checks

Check Brevo approval and attempt one real recovery-email delivery without allowing that external dependency to derail the kernel.

---

## 3. Required preflight

Before implementation:

- read the canonical roadmap;
- read Kernel 78 spec and reportback;
- read all eWrite architecture notes;
- inspect the current Writer's Room and Library UI;
- inspect the Socio manuscripts;
- inspect equipment rule links;
- inspect Character mechanics, health systems, tutorial actions, index cards, Cues, Scene Elements, asset access, account export/deletion, security notes, and recovery-email configuration;
- confirm Kernel 79 and migration numbers are unused;
- record the current UI baseline so Grant's changes are not overwritten.

Record:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}	{{.Status}}	{{.Ports}}'
```

---

## 4. Socio Ruleset organization

### 4.1 Required hierarchy

Provide or verify:

```text
Ruleset: Socio: Stories of Us

Series:
- Core Rulebook
- Quickstart
- Niava
```

Each Series may contain Modules, Publications, Sections, and Subsections.

### 4.2 Import curation

Large manuscripts must not become one unusable wall of headings.

Provide curation support such as:

- heading outline;
- proposed Module boundaries;
- proposed Publication boundaries;
- reorder or reassignment;
- split Publication at heading;
- merge adjacent Publications;
- rename without breaking anchors;
- preserve source lineage;
- warn before destructive restructuring.

Do not require Grant to cut and paste hundreds of sections manually.

### 4.3 Source lineage

Track enough metadata to know:

- source manuscript;
- source filename;
- import date;
- imported heading path;
- derived Publication;
- whether content was later edited;
- whether export can reconstruct hierarchy.

### 4.4 Duplicate content

The Core Rulebook and Quickstart may contain overlapping rules.

Do not automatically deduplicate substantive text without operator review.

The system may flag likely duplicates, suggest canonical links, or preserve separate contextual versions.

---

## 5. Reusable directory system

### 5.1 Purpose

A directory provides compact navigational metadata for a dense rules domain. It does not duplicate full rule text.

Conceptual model:

```text
Directory
- id
- ruleset_id
- directory_type
- title
- slug
- visibility
- scope
- sort/filter configuration

DirectoryEntry
- id
- directory_id
- canonical_name
- aliases
- compact_summary
- category metadata
- target_publication_id
- target_section_id
- sort_key
```

The exact schema may vary if eWrite object links can support this cleanly.

### 5.2 Skill Directory

The Skill Directory must support:

- all Socio skills represented in the canonical rules;
- exact section link;
- alphabetical browse;
- search;
- filter by attribute;
- filter by category/classification where source data supports it;
- aliases;
- compact display;
- Character-sheet links.

### 5.3 Directory authority

Directory entries may be created or managed by Crew+ users with correct scope.

Reading follows the target publication's visibility. A directory must not reveal the existence or title of a hidden target publication.

### 5.4 Canonical source

A directory entry points to the canonical eWrite section. Do not copy the full rule into the directory.

Compact summary metadata may exist but must be clearly distinct from the canonical rule.

### 5.5 Future reuse

Design so later directories can represent actions, reactions, stances, health systems, equipment, conditions, and oracles.

Do not hard-code the database exclusively for skills.

---

## 6. Character mechanics and health-system links

### 6.1 Character-sheet integration

Character mechanics should link to exact eWrite rules.

At minimum support links from:

- skills;
- attributes where rules exist;
- values/resources;
- health systems;
- actions or reactions displayed on Character surfaces.

### 6.2 Eight health systems

Socio's health systems should be linkable through a compact directory or structured mechanics index.

The UI must not require users to search manually through the full book.

### 6.3 Link behavior

A click should:

- open Library reader;
- open the exact publication;
- focus the exact section;
- preserve access control;
- provide a safe denial if unavailable;
- avoid exposing hidden titles.

### 6.4 Missing links

The UI should distinguish linked, unlinked, removed target, hidden target, and alias redirect states.

Crew+ users may receive editing affordances to repair missing links.

---

## 7. Tutorial actions and choices

Kernel 79 should allow tutorial interactions to link to exact rules.

Examples include stance choices, actions, reactions, health consequences, equipment interactions, and social mechanics.

The link should be optional and non-blocking.

A player should not be forced out of the tutorial flow merely because they opened a rule. Preserve return context through a side panel, modal, new reader route, or another repository-consistent method.

Do not hard-code Catharsis-only link behavior when the same model can serve future tutorials.

---

## 8. Index cards, Cues, and Scene Elements

### 8.1 Index cards

Add optional exact-rule link metadata to index cards.

Cards should be able to link to a Publication, Section, Subsection, or Directory entry.

### 8.2 Cues

Cues may optionally link to a script section, rule section, rehearsal note, or procedure.

The Cue remains a theatrical operation; the link is reference metadata.

### 8.3 Scene Elements

Scene Elements may optionally link to a rule, script, location note, or item description.

This kernel need not redesign Scene Elements.

### 8.4 Reusable link model

Prefer extending the existing `ewrite_object_links` model. Do not create separate unrelated link systems for each object type.

---

## 9. Ruleset-wide Library navigation

### 9.1 Ruleset landing page

Show:

- title;
- description;
- Series;
- major Modules;
- directories;
- recent or featured Publications;
- search within Ruleset;
- export where authorized.

### 9.2 Series landing page

Show Series description, Modules, Publications, directory links, ordered navigation, and source metadata.

### 9.3 Module landing page

Show Module description, Publications, ordered reading path, linked directories, and export.

### 9.4 Publication reader

Improve cross-publication navigation with:

- hierarchy breadcrumb;
- previous/next Publication;
- return to Module;
- related rules;
- directory context;
- exact-section highlighting;
- return-to-source-object context where practical.

### 9.5 Search

Support Ruleset-scoped search.

Results should include Publication title, section heading, snippet, hierarchy breadcrumb, directory matches, and visibility-safe results only.

### 9.6 Dense indexes

Do not render hundreds of skill headings in the primary Ruleset index. Use compact directory surfaces.

---

## 10. Image visibility

### 10.1 Problem

Kernel 78 recorded that embedded images were link-knowable because asset reads gated only some asset types.

A direct asset URL must not bypass eWriting visibility.

### 10.2 Required behavior

When an image is used by eWrite:

- resolve effective publication visibility;
- verify requester;
- verify Production when applicable;
- verify editor authority for drafts;
- deny unauthorized direct requests;
- avoid revealing hidden publication metadata.

### 10.3 Shared assets

An image may be reused by multiple publications with different visibility.

The implementation must document and enforce a model that avoids accidental exposure. Acceptable approaches include publication-bound asset references, permission-aware reference lookup, or another model that prevents a private use from becoming public accidentally.

### 10.4 Caching

Ensure browser and proxy caching does not serve restricted images after access changes.

Use appropriate cache headers for private content.

### 10.5 Export

Authorized hierarchical exports may include images. Unauthorized or inaccessible images must be omitted with a manifest warning.

---

## 11. Hierarchical export

### 11.1 Supported levels

Provide export for:

- Module;
- Series;
- Ruleset.

### 11.2 Package structure

Recommended:

```text
ruleset-export/
  README.md
  manifest.json
  ruleset.json
  directories/
    skills.json
  series/
    core-rulebook/
      series.json
      modules/
        character-rules/
          module.json
          publications/
            social-stances.md
            social-stances.metadata.json
    quickstart/
    niava/
  assets/
  links.json
```

Exact layout may vary, but hierarchy must be reconstructable.

### 11.3 Manifest

Include:

- export format version;
- Ruleset ID/title;
- Series;
- Modules;
- Publications;
- stable anchors;
- aliases;
- internal links;
- authorized object-link references;
- directory entries;
- assets;
- visibility metadata;
- source lineage;
- checksums.

### 11.4 Importability

Kernel 79 does not need full Ruleset re-import if that would expand scope significantly.

However, export structure must allow a future importer to reconstruct hierarchy without guessing.

### 11.5 Security

Production-private content must not appear in another Production's export. Drafts require editor authority.

---

## 12. UI preservation and refinement

### 12.1 Writer's Room

Preserve the writer-focused layout, current colors, rearranged windows, collection hierarchy as secondary context, and current keyboard/editor behavior.

Kernel 79 may add ruleset navigation, directory management, hierarchy curation, export controls, and object-link management.

Do not recenter the UI on collection administration.

### 12.2 Library

Grant may continue color adjustments.

The builder should preserve current changes, improve rules navigation, maintain readable contrast, avoid overloading the index with hundreds of headings, and verify responsive behavior.

### 12.3 Accessibility

Check contrast, keyboard navigation, visible focus, semantic headings, link purpose, search/filter behavior, and color-independent status communication.

Do not turn this into a full accessibility-certification kernel.

---

## 13. Brevo verification

Kernel 79 must perform one bounded check:

1. inspect current Brevo approval/account status;
2. attempt one controlled real recovery email;
3. record delivery result;
4. preserve the tested recovery implementation;
5. do not spend the kernel debugging an external account indefinitely.

If still blocked, record the error, preserve break-glass recovery, and continue Kernel 79.

---

## 14. Required schema and migrations

Likely additions:

- rules directories;
- directory entries;
- directory aliases/categories;
- expanded object-link support;
- source lineage;
- hierarchical export metadata if needed;
- image-reference permission data if needed.

Requirements:

- next verified migration numbers;
- clean-install safe;
- additive;
- compatible with Kernel 78 data;
- no duplicate Socio hierarchy;
- account deletion integrated;
- account export integrated;
- backup/restore automatically covers new data;
- stable foreign-key behavior;
- visibility enforced server-side.

---

## 15. Required tests

### Socio hierarchy

- Ruleset exists;
- Core Rulebook, Quickstart, and Niava Series exist;
- Modules/Publication assignment works;
- hierarchy reordering is stable;
- no duplicate creation;
- source lineage preserved.

### Directory

- create Skill Directory;
- create entry;
- alias search;
- alphabetical browse;
- attribute/category filter;
- exact section link;
- hidden target does not leak;
- deleted target degrades safely;
- Production scope enforced;
- no full-rule duplication.

### Character links

- skill link opens exact rule;
- health-system link opens exact rule;
- unauthorized target denied;
- missing link visible to authorized editor;
- client cannot forge target authority.

### Tutorial links

- tutorial action carries optional rule link;
- opening rule preserves return context;
- inaccessible rule gives safe result;
- no Catharsis-only authority bypass.

### Object links

- index card link;
- Cue link;
- Scene Element link;
- shared link model;
- deleted/renamed target behavior;
- stable anchor alias behavior.

### Image visibility

- authenticated publication image denied anonymously;
- Production image denied to another Production;
- draft image denied to reader;
- editor can read draft image;
- copied URL cannot bypass access;
- cache headers appropriate;
- export includes authorized images only.

### Export

- Module export;
- Series export;
- Ruleset export;
- hierarchy manifest valid;
- Markdown, anchors, links, directories, and images preserved;
- unauthorized export denied;
- checksums valid.

### Navigation

- Ruleset, Series, and Module landing pages;
- scoped search;
- directory results;
- breadcrumbs;
- previous/next;
- exact-section links;
- no hidden-title leakage.

### UI regression

- Writer's Room baseline preserved;
- current colors/layout not overwritten;
- writer remains primary focus;
- Library remains readable;
- Cast/Audience authoring still denied;
- save/revision/conflict behavior remains.

### Kernel regression

- full Go suite passes;
- node tests pass;
- fresh-install smoke passes;
- Kernel 77A venues remain;
- eWrite safety remains;
- account deletion/export remain;
- backup/restore covers new data;
- `straturli` access remains.

---

## 16. Required browser proof

Manually prove:

- Socio Ruleset landing page;
- Core Rulebook, Quickstart, and Niava Series;
- Skill Directory alphabetical browse;
- skill search;
- attribute/category filter;
- Character skill opens exact rule;
- Character health system opens exact rule;
- tutorial action opens exact rule and returns;
- index card rule link;
- Cue rule/script link;
- Scene Element link;
- restricted image copied URL denied;
- Module export;
- Series export;
- Ruleset export;
- Writer's Room layout preserved;
- Library colors/layout remain usable;
- hidden Production content does not appear in search;
- Brevo delivery result recorded.

---

## 17. Required artifacts

```text
Construction/Kernels/Kernel 79 — eWrite Integration and Rules Navigation.md
Construction/eWrite/socio-ruleset-organization.md
Construction/eWrite/ewrite-directory-contract.md
Construction/eWrite/ewrite-object-link-contract.md
Construction/eWrite/ewrite-image-visibility.md
Construction/eWrite/ewrite-hierarchical-export.md
```

Use the standard reportback template.

---

## 18. Required evidence

### Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}	{{.Status}}	{{.Ports}}'
```

### Tests

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 ./...
```

Run frontend/node suites and fresh-install smoke.

### Static checks

```bash
git diff --check
```

Run `node --check` on changed scripts.

### Export proof

Record Module, Series, and Ruleset export sizes; Publication count; section/anchor count; directory entry count; image count; manifest/checksum result; and reconstruction-readiness assessment.

### Image proof

Record authorization results without exposing private asset content.

### Brevo proof

Record approval status, delivery attempt result, message arrival result, and provider error if blocked. Do not record credentials or recovery tokens.

---

## 19. Pass criteria

Kernel 79 passes when:

- Socio Ruleset hierarchy is practical and live;
- Core Rulebook, Quickstart, and Niava are organized as Series;
- Skill Directory works without duplicating full rules text;
- skills link to exact sections;
- Character mechanics and health systems link to exact rules;
- tutorial actions support exact-rule links;
- index cards, Cues, and Scene Elements support reusable links;
- ruleset-wide navigation is usable;
- embedded images honor eWriting visibility;
- copied URLs do not bypass restrictions;
- Module, Series, and Ruleset export work;
- hierarchy and anchors survive export;
- current Writer's Room UI is preserved;
- Library remains usable;
- anonymous reading remains deferred;
- Kernel 76–78 security and lifecycle guarantees remain intact;
- full tests and browser proof pass.

---

## 20. Partial and fail rules

### PASS WITH EXTERNAL FOLLOW-UP

Allowed only if all Kernel 79 feature work passes and Brevo remains externally blocked.

### PARTIAL

Use when only the Skill Directory ships, image visibility remains bypassable, whole-Ruleset export is incomplete, Socio hierarchy remains one giant manuscript, or the Writer's Room baseline is reverted.

### FAIL

Use when private images leak, Production-private rules appear in another Production, exports leak hidden content, copied URLs bypass authority, stable anchors break, Writer's Room direction is lost, Kernel 78 security regresses, or Grant is locked out.

---

## 21. Non-goals

Kernel 79 does not:

- open anonymous public reading;
- build public signup;
- redesign Writer's Room from scratch;
- editorially rewrite Socio;
- automatically deduplicate all rulebooks;
- build an LMS;
- build storyboards;
- build Microscope;
- build vector drawing;
- create a public rules website;
- solve Brevo beyond one bounded verification attempt.

---

## 22. Kernel-maker guidance

- Directories are navigation, not copies.
- Rules links must extend the reusable object-link model.
- Dense indexes need purpose-built navigation.
- Image access must follow content access.
- Export must preserve hierarchy and meaning.
- Grant's current Writer's Room arrangement is product direction.

---

## 23. Immediate operator outcome

At completion, Grant must be able to answer:

1. Can I browse Socio as Core Rulebook, Quickstart, and Niava?
2. Can I find a skill without scrolling through hundreds of headings?
3. Can I filter skills by meaningful categories?
4. Does a Character skill open the exact rule?
5. Do health systems open their exact rules?
6. Can tutorial choices link to rules without disrupting play?
7. Can cards, Cues, and Scene Elements link to eWritings?
8. Can a copied private image URL bypass access?
9. Can I export one Module?
10. Can I export one Series?
11. Can I export the entire Ruleset without flattening it?
12. Did the Writer's Room keep its writer-focused design?
13. Did the Library remain usable after color/layout changes?
14. Is anonymous reading still closed?
15. Did Brevo approve real recovery delivery?
16. Did all Kernel 76–78 protections remain intact?

Kernel 79 succeeds when eWrite becomes a practical rules-navigation system during actual play, rather than merely a place where large Markdown files can be stored and read.
