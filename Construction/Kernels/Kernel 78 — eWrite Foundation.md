# Kernel 78 — eWrite Foundation

**Status:** READY FOR IMPLEMENTATION
**Type:** Major product-feature kernel
**Primary tracks:** Victory Core, Socio
**Secondary tracks:** Anthology, Education, Operational Integrity
**Planning authority:** `Victory_Canonical_Roadmap_v2.md`
**Sequence position:** After Kernel 77A
**Product name:** eWrite
**Content objects:** eWritings
**Authoring home:** Writer's Room
**Reading home:** Library

---

## 0. Kernel contract

Kernel 78 creates Victory's first rules-native writing, publication, and reading system.

The system is called **eWrite**. The works created and published through it are **eWritings**.

The Writer's Room is where authorized users create and edit. The Library is where users browse and read.

This is not:

- internal construction documentation;
- a replacement for Git-backed kernel and operator records;
- Google Docs;
- a generic office suite;
- a full learning-management system;
- a package marketplace;
- a live multi-cursor editor;
- a bulk data-entry project.

This kernel must produce one complete vertical:

```text
Author creates or imports Markdown
→ eWrite recognizes hierarchy and headings
→ draft is saved safely
→ revision history exists
→ publication is published with visibility rules
→ reader finds it in the Library
→ direct links open exact sections
→ existing Victory objects may link into exact rule sections
→ content exports without being trapped in Victory
```

The implementation must be safe enough for large real rules documents, not only tiny fixtures.

---

# 1. Locked product decisions

## 1.1 Product language

Use these terms in the interface:

- **eWrite** — the authoring and publication system;
- **eWriting** — one written work;
- **eWritings** — the collection of works;
- **Writer's Room** — creation and editing;
- **Library** — reading and browsing.

The database may use practical internal names, but the interface should not expose a long phrase such as "Rules, Scripts, and Publications."

## 1.2 Visible hierarchy

The visible hierarchy is:

```text
Ruleset
→ Series
→ Module
→ Publication
→ Section
→ Subsection
```

Examples:

```text
Socio: Stories of Us              Ruleset
Core Rulebook                     Series
Character Rules                   Module
Social Stances                    Publication
Defensive Stance                  Section
Taking Defensive Stance           Subsection
```

```text
Socio: Stories of Us              Ruleset
Niava                             Series
The Locked Courtyard              Module
Scene Script                      Publication
Kessa Introduction                Section
First Offer                       Subsection
```

The system must not require every integration to use all levels in every case, but the hierarchy must remain available and understandable.

## 1.3 Database flexibility

Do not create six rigid unrelated tables solely because the interface has six labels.

Preferred architectural direction:

```text
eWriteCollection
- id
- parent_id
- collection_type
- title
- slug
- sort_order
- visibility or inherited visibility
- owner / authority context
```

Collection types may include:

- ruleset;
- series;
- module.

Then use distinct publication and section structures for the authored content itself.

A practical schema may differ after repository inspection, but it must preserve:

- stable hierarchy;
- parent-child ordering;
- direct linking;
- future integration flexibility;
- no requirement that every game use every level;
- clean export.

## 1.4 Author permissions

Writing is a **Crew+** capability.

Base role rule:

```text
Crew
Director
Producer
Operator
```

may create or edit eWritings when authorized within the relevant Location, Production, or collection scope.

Cast and Audience are readers by default.

Role alone must not grant unrestricted editing across all content. The server must also verify the relevant scope or named-editor grant.

Named editors may be supported.

## 1.5 Visibility

Published content may be:

- public;
- authenticated;
- Production-only.

Defaults should be conservative.

Drafts are never public.

Visibility and edit authority are separate.

## 1.6 First real content

Grant may provide:

- Socio Core Rulebook;
- Socio Quickstart;
- Niava supplement.

Kernel 78 should support substantial real manuscripts.

The kernel must not require Grant to manually split every paragraph into records before import.

The importer should use Markdown headings to recognize sections and subsections.

The kernel must include:

1. a small controlled fixture for precise automated testing;
2. at least one substantial real Socio manuscript for scale and usability proof when available.

Importing and editorially perfecting all three books is not required for PASS.

## 1.7 Editing model

Initial editing includes:

- Markdown source editor;
- rendered preview;
- Save;
- draft/published state;
- revision history;
- save-conflict warning;
- no simultaneous live cursor editing;
- no tracked changes;
- no comments system unless repository evidence makes it trivial and bounded.

## 1.8 Internal project documentation

Construction documentation remains in Git-backed Markdown files.

Do not import kernels, reportbacks, operator notes, architecture records, or security records into eWrite merely because they are documents.

---

# 2. Product goals

Kernel 78 has nine goals.

## Goal A — Create eWrite hierarchy

Support:

```text
Ruleset
→ Series
→ Module
→ Publication
→ Section
→ Subsection
```

with stable IDs, slugs, ordering, and links.

## Goal B — Build the Writer's Room authoring surface

Authorized Crew+ users can:

- create collections;
- create publications;
- paste Markdown;
- upload `.md` files;
- edit source;
- preview;
- save drafts;
- publish;
- unpublish;
- inspect revisions;
- restore or copy from an older revision where safe;
- see conflict warnings.

## Goal C — Build the Library reading surface

Readers can:

- browse published eWritings;
- move through hierarchy;
- open long-form publications;
- open short publications;
- jump to sections and subsections;
- follow internal and cross-publication links;
- copy direct links;
- search or filter at a minimum useful level;
- see visibility-appropriate content only.

## Goal D — Safe Markdown

Support useful Markdown while preventing active content and unsafe execution.

## Goal E — Stable links

Links must survive normal edits and title changes where practical.

## Goal F — Import and export

Users can bring Markdown in and get Markdown back out.

## Goal G — Revision and conflict safety

Ordinary saves must not silently erase a newer revision.

## Goal H — Link Victory objects into eWritings

At least one existing Victory object must link directly to an exact publication section.

## Goal I — Prove real scale

The system must handle a substantial real Socio manuscript without collapsing into unusable performance or structure.

---

# 3. Required investigation

Before implementation, inspect:

- current Library Venue;
- current Writer's Room or closest existing writing venue;
- venue capability and route patterns;
- role and admission resolution;
- current Markdown use;
- frontend rendering practices;
- upload handling from Kernels 76 and 77;
- current asset system;
- current search/indexing capabilities;
- current object-link patterns;
- current migration count;
- current route and WebSocket security conventions;
- current backup/export implications;
- current account deletion behavior;
- current public/authenticated/Production-only visibility patterns;
- current Caddy/static routing;
- current Socio source formats available in the repository.

Do not ask Grant for code facts the repository can reveal.

---

# 4. Domain model

## 4.1 Collection nodes

A collection node represents an organizational level above a publication.

Required conceptual fields:

```text
id
parent_id
collection_type
title
slug
description
sort_order
scope_type
scope_id
visibility
created_by
created_at
updated_at
```

Collection types:

```text
ruleset
series
module
```

A module may contain publications.

A publication may exist directly under a higher-level collection only if the architecture allows omitted intermediate levels deliberately.

## 4.2 Publications

Required conceptual fields:

```text
id
collection_id
title
slug
summary
source_markdown
rendered_or_parsed_representation
status
visibility
current_revision_id
created_by
updated_by
published_at
created_at
updated_at
```

Status at minimum:

```text
draft
published
archived
```

Archived content must not disappear without trace or break links silently.

## 4.3 Sections and subsections

Sections derive from Markdown headings but require stable identity.

Required conceptual fields:

```text
id
publication_id
parent_section_id
heading_level
title
slug
stable_anchor
sort_order
source_start_or_structure_reference
created_at
updated_at
```

Kernel 78 must support at least:

- publication title;
- sections;
- subsections.

Deeper Markdown heading levels may be rendered, but the visible product language need not expose more than Section and Subsection initially.

## 4.4 Revisions

Each save that changes source content must create or reference a revision.

Required conceptual fields:

```text
id
publication_id
revision_number
source_markdown
created_by
created_at
base_revision_id
change_summary_optional
```

A revision must preserve enough information to:

- inspect history;
- compare metadata;
- restore or duplicate old content safely;
- export prior source when appropriate.

A full visual diff engine is not required.

## 4.5 Named editors

If implemented, editor grants require:

```text
publication_or_collection_id
user_id
permission
granted_by
created_at
```

Permissions may begin with:

```text
edit
publish
manage_editors
```

Role and named grants must be resolved server-side.

---

# 5. Writer's Room

## 5.1 Entry

The Writer's Room is the primary authoring venue.

It should offer:

- eWrite dashboard;
- recent drafts;
- recently edited publications;
- hierarchy browser;
- create Ruleset;
- create Series;
- create Module;
- create Publication;
- import Markdown;
- editor access.

The existing venue architecture should be reused rather than creating an unrelated admin app.

## 5.2 Create flow

An authorized user may:

1. choose or create parent collection;
2. create publication title;
3. choose visibility default;
4. paste Markdown or upload `.md`;
5. preview recognized structure;
6. resolve import warnings;
7. save as draft;
8. publish when ready.

## 5.3 Markdown editor

The first editor may be a textarea or code-style editor with preview.

Required:

- source remains visible and editable;
- preview updates deliberately or safely;
- large documents remain usable;
- save button is clear;
- current revision is shown;
- conflict state is visible;
- unsupported content warnings are shown;
- raw HTML does not execute.

Do not require a rich-text editor.

## 5.4 Structure preview

Before import or save, show recognized headings:

```text
Publication
  Section
    Subsection
```

Warnings should identify:

- duplicate headings;
- headings with no parent;
- unsupported raw HTML;
- broken internal links;
- unsafe URLs;
- images that cannot be resolved;
- document size or parser limits.

Warnings must not silently rewrite content without reporting what changed.

## 5.5 Save conflict

When User A opens revision 4 and User B saves revision 5 first:

- User A's save must not silently overwrite revision 5;
- return a conflict;
- preserve User A's submitted source;
- offer reload, copy, or create-new-revision resolution;
- show the current revision number.

No live merge editor is required.

## 5.6 Publish

Publishing must:

- verify publish authority;
- verify visibility;
- parse/sanitize content;
- generate or update stable section anchors;
- create a revision;
- update the Library index;
- preserve previous published revision until the new publish succeeds;
- avoid half-published state.

Unpublishing must not destroy the work.

---

# 6. Library

## 6.1 Entry

The Library is the main reading and browsing venue.

It should provide:

- hierarchy browser;
- Rulesets;
- Series;
- Modules;
- Publications;
- recently published works;
- search or filter;
- direct-link handling;
- reader navigation.

## 6.2 Reading view

A publication reading view must support:

- title;
- summary;
- table of contents;
- long-form scrolling;
- short-form article display;
- section highlighting on direct open;
- previous/next navigation where ordered;
- copy direct link;
- visibility indicator for authorized editors;
- revision or publication date where useful.

## 6.3 Direct section links

A link to a section must:

- open the correct publication;
- scroll/focus to the exact section;
- preserve access control;
- return a clear not-found or forbidden state;
- not expose unpublished titles through error messages.

## 6.4 Search

Minimum search may be:

- title;
- summary;
- headings;
- body text.

Results must respect visibility.

A simple PostgreSQL search is acceptable if it fits the current architecture.

Do not add an external search service in this kernel.

## 6.5 Public reading

Public publications may be reachable without login.

Public access must expose only:

- published content;
- public collections/publications;
- safe rendered content;
- public assets.

It must not expose:

- drafts;
- revision history;
- editor identities beyond deliberate byline fields;
- private source metadata;
- Production-only hierarchy;
- unpublished links.

---

# 7. Markdown import

## 7.1 Accepted input

Support:

- pasted Markdown;
- uploaded `.md` file;
- UTF-8 text;
- substantial document size within measured limits.

## 7.2 Heading recognition

Recommended mapping:

```text
# Publication title
## Section
### Subsection
####+ rendered nested heading inside subsection or normalized according to parser rules
```

The importer should not assume every file begins perfectly.

It should provide a preview and warnings.

## 7.3 Original source preservation

Preserve the original imported Markdown or a clearly normalized source copy.

Export must not depend solely on rendered HTML.

## 7.4 Import report

Record:

- source filename;
- byte size;
- heading count;
- section count;
- subsection count;
- links detected;
- images detected;
- unsupported constructs;
- unsafe content removed or rejected;
- duplicate-anchor resolutions.

## 7.5 Large manuscript behavior

The importer must be tested against a substantial real Socio manuscript when provided.

Measure:

- upload/import duration;
- parse duration;
- save duration;
- render duration;
- browser responsiveness;
- section count;
- database size increase.

Do not claim scale readiness from a 20-line fixture.

---

# 8. Safe Markdown rendering

Kernel 76 required Markdown security design before eWrite.

Kernel 78 must implement it.

## 8.1 Allowed baseline

Support safely:

- headings;
- paragraphs;
- emphasis;
- strong text;
- ordered/unordered lists;
- blockquotes;
- code blocks;
- inline code;
- tables;
- horizontal rules;
- safe links;
- safe images;
- footnotes if library support is bounded and safe.

## 8.2 Raw HTML

Default:

```text
Raw HTML is not executed.
```

Either:

- strip raw HTML;
- escape it;
- reject it with a warning.

Do not allow:

- script;
- iframe;
- object;
- embed;
- event-handler attributes;
- inline JavaScript;
- arbitrary style injection;
- forms;
- active SVG.

## 8.3 URLs

Reject unsafe schemes such as:

```text
javascript:
data: (except tightly controlled image handling if explicitly safe)
vbscript:
file:
```

Allow only deliberate schemes.

External links should use safe attributes where appropriate.

## 8.4 Images

Images may reference:

- Victory assets;
- uploaded safe images;
- approved external URLs if policy allows.

Prefer Victory-managed assets.

Do not proxy or fetch arbitrary URLs server-side without a separate safe design.

## 8.5 Rendering architecture

Use a mature Markdown parser and sanitizer where appropriate.

Victory does not need to hand-write a Markdown parser.

Any dependency added must be:

- current enough;
- license-compatible;
- documented;
- pinned;
- tested against malicious fixtures.

---

# 9. Stable anchors and links

## 9.1 Anchor generation

Each Section and Subsection receives a stable anchor.

Do not rely only on regenerating `slugify(title)` on every save.

Preferred behavior:

- initial anchor generated from heading;
- stable internal ID retained;
- title may change without breaking old link immediately;
- prior anchor may redirect or alias when practical;
- duplicate headings receive deterministic disambiguation.

## 9.2 Internal links

Support:

- link within same publication;
- link to another publication;
- link to exact section/subsection;
- link from Victory object to publication section.

## 9.3 Link syntax

Victory may support ordinary Markdown links plus a stable internal link form.

Example conceptual forms:

```text
[Defensive Stance](victory://ewrite/publication-id#section-id)
```

or repository-consistent route URLs.

The UI should generate links for users rather than requiring them to know internal syntax.

## 9.4 Broken-link handling

On save or publish:

- detect broken internal links where feasible;
- warn author;
- do not silently redirect to unrelated content;
- preserve unresolved source for correction.

---

# 10. Visibility and authority

## 10.1 Visibility levels

Required:

```text
public
authenticated
production
```

A future private/editor-only state may exist through draft status.

## 10.2 Inheritance

Collections may define default visibility, but publications must have clear effective visibility.

Do not create ambiguous inheritance that authors cannot understand.

## 10.3 Editing authority

Server must verify:

- authenticated user;
- Crew+ role;
- relevant Location/Production scope;
- named editor grant when required;
- publish authority separately when applicable.

Client-supplied role or user IDs are not authority.

## 10.4 Reading authority

Server must verify effective visibility.

A user in Production A must not read Production B material merely because both use the same Ruleset title.

## 10.5 Draft privacy

Drafts are visible only to:

- creator;
- named editors;
- authorized higher roles within correct scope;
- operator only through deliberate exceptional tooling, not ordinary browsing.

---

# 11. Existing-object linking proof

At least one existing Victory object must gain an eWrite link.

Preferred proof object:

- equipment item;
- Character mechanic;
- Cue;
- index card;
- tutorial action.

The link must open an exact rule section.

Recommended first proof:

```text
A Socio Character or tutorial object links directly to the Social Stances rule section.
```

If the large Socio import is not yet available, use a controlled Socio fixture and replace it later.

The implementation should create a reusable link model rather than hard-code one page.

---

# 12. Export

## 12.1 Publication export

Export one publication as Markdown plus metadata.

Include:

- title;
- hierarchy path;
- source Markdown;
- visibility;
- publication status where authorized;
- section map;
- internal-link manifest;
- image references.

## 12.2 Collection export

A bounded collection export should include:

- hierarchy manifest;
- publications;
- Markdown files;
- assets where authorized;
- link map.

A complete ruleset export may be deferred if too large, but the schema must not trap content.

## 12.3 User account export integration

Kernel 77 account export must include user-owned eWrite data appropriately.

Kernel 78 must extend account export behavior and tests.

## 12.4 Account deletion integration

On account deletion:

- drafts owned solely by user are deleted or transferred according to ownership rules;
- published shared material is transferred or anonymized safely;
- revision authorship is anonymized where required;
- no ownerless publication remains;
- shared published rules are not destroyed casually.

---

# 13. Images and media

Kernel 78 requires safe image references but not a full media-management redesign.

Support:

- selecting an existing Victory asset;
- inserting safe image Markdown/reference;
- alt text;
- caption if practical;
- thumbnail/full display;
- permission-aware retrieval.

Do not include:

- arbitrary active embeds;
- video hosting;
- remote scraping;
- unrestricted iframe embeds.

---

# 14. Revision history

## 14.1 Required

For each revision show:

- revision number;
- author display identity where permitted;
- timestamp;
- optional summary;
- draft/published state;
- restore/copy action for authorized editors.

## 14.2 Restore behavior

Restoring an old revision should create a new revision.

Do not rewrite history.

## 14.3 Deleted author

If revision author deletes account:

- preserve revision;
- anonymize author;
- do not expose deleted identity;
- preserve publication history.

---

# 15. Required routes and interfaces

Exact names may follow repository conventions.

Suggested API surface:

## Collections

```text
GET    /api/ewrite/collections
POST   /api/ewrite/collections
GET    /api/ewrite/collections/{id}
PATCH  /api/ewrite/collections/{id}
DELETE /api/ewrite/collections/{id}
```

## Publications

```text
GET    /api/ewrite/publications
POST   /api/ewrite/publications
GET    /api/ewrite/publications/{id}
PATCH  /api/ewrite/publications/{id}
DELETE /api/ewrite/publications/{id}
POST   /api/ewrite/publications/{id}/publish
POST   /api/ewrite/publications/{id}/unpublish
POST   /api/ewrite/publications/import
GET    /api/ewrite/publications/{id}/export
```

## Revisions

```text
GET    /api/ewrite/publications/{id}/revisions
GET    /api/ewrite/publications/{id}/revisions/{revision}
POST   /api/ewrite/publications/{id}/revisions/{revision}/restore
```

## Library/read

```text
GET    /api/library
GET    /api/library/publications/{slug-or-id}
GET    /api/library/publications/{id}/sections/{anchor}
GET    /api/library/search
```

## Editor grants

```text
GET    /api/ewrite/.../editors
POST   /api/ewrite/.../editors
DELETE /api/ewrite/.../editors/{user_id}
```

Only implement what is needed for the complete vertical.

---

# 16. Required migrations

Create the next verified migration numbers.

Likely structures:

- eWrite collections;
- publications;
- revisions;
- sections;
- anchor aliases;
- editor grants;
- object links;
- search support/indexes.

Requirements:

- clean-install safe;
- additive;
- idempotent where applicable;
- compatible with Kernel 77A canonical venue state;
- no dependency on manually created live rows;
- account deletion integrated;
- account export integrated;
- backup/restore covered automatically through database/assets;
- explicit foreign-key behavior;
- no raw HTML stored as trusted rendered output without sanitization guarantees.

---

# 17. Required tests

## 17.1 Hierarchy tests

- create Ruleset;
- create Series beneath Ruleset;
- create Module beneath Series;
- create Publication beneath Module;
- invalid parent type rejected;
- ordering stable;
- omitted level behavior explicit;
- duplicate slug handled safely.

## 17.2 Authority tests

- Crew with correct scope can create/edit;
- Crew outside scope denied;
- Cast denied authoring;
- Audience denied authoring;
- named editor works;
- editor without publish permission cannot publish;
- client-supplied role ignored;
- Production A content denied to Production B.

## 17.3 Visibility tests

- public publication readable anonymously;
- authenticated publication denied anonymously;
- authenticated publication readable after login;
- Production publication restricted correctly;
- draft never public;
- unpublished content not discoverable through search.

## 17.4 Markdown safety tests

Include malicious fixtures:

- script tags;
- event-handler attributes;
- `javascript:` links;
- iframe;
- object/embed;
- form;
- active SVG;
- malformed HTML;
- unsafe data URI;
- nested parser edge cases.

Prove no script or active content executes.

## 17.5 Import tests

- paste Markdown;
- upload `.md`;
- headings become sections/subsections;
- duplicate headings receive stable unique anchors;
- unsupported content reported;
- original source preserved;
- large fixture imports within measured limits.

## 17.6 Revision tests

- save creates revision;
- no-change save behavior defined;
- conflict rejected;
- user source preserved on conflict;
- restore creates new revision;
- revision history remains append-forward.

## 17.7 Link tests

- same-publication section link;
- cross-publication link;
- direct section route;
- title change does not break stable anchor;
- old alias works where implemented;
- broken link warning;
- existing Victory object opens exact section.

## 17.8 Export tests

- Markdown reopens;
- hierarchy manifest valid;
- source preserved;
- no secrets;
- unauthorized export denied;
- account export includes user-owned eWrite data.

## 17.9 Deletion tests

- author deletion anonymizes revision authorship;
- sole-owned draft handled;
- published shared work transferred or blocked;
- no ownerless collection;
- links continue where shared publication remains.

## 17.10 Regression tests

- full Go suite passes;
- clean test database passes;
- live database untouched by tests;
- Kernel 76/77 security controls remain;
- Kernel 77A Audition Hall remains migration-backed and visible;
- `straturli` access remains;
- backup includes eWrite data automatically;
- restore includes eWrite data.

---

# 18. Real Socio proof

## 18.1 Controlled fixture

Create a small Socio fixture containing:

- one Ruleset;
- one Series;
- one Module;
- one Publication;
- at least three Sections;
- at least two Subsections;
- internal links;
- cross-publication link;
- one image;
- one existing Victory-object link.

This fixture supports repeatable automated proof.

## 18.2 Substantial manuscript

When Grant supplies a real manuscript, import at least one substantial source from:

- Core Rulebook;
- Quickstart;
- Niava supplement.

The reportback must record:

- source file;
- byte/word count;
- heading count;
- sections/subsections;
- warnings;
- import time;
- render time;
- browser behavior;
- export proof.

Do not require editorial perfection.

If no real manuscript is available during execution, the kernel may PASS only if the importer is proven against a substantial synthetic Markdown fixture and the reportback explicitly records the missing live manuscript proof as a follow-up.

---

# 19. Required frontend surfaces

## Writer's Room

At minimum:

- hierarchy browser;
- recent drafts;
- create/import;
- Markdown editor;
- preview;
- save;
- revision status;
- publish controls;
- visibility selector;
- conflict state;
- editor grants if implemented.

## Library

At minimum:

- browse hierarchy;
- search/filter;
- publication reader;
- table of contents;
- direct section links;
- public/authenticated/Production access states;
- safe image display.

Avoid overdesign. Function first, but browser-visible use is mandatory.

---

# 20. Required artifacts

## Kernel specification

```text
Construction/Kernels/Kernel 78 — eWrite Foundation.md
```

## Architecture note

```text
Construction/Domains/eWrite/ewrite-domain-model.md
```

## Markdown security note

```text
Construction/Domains/eWrite/ewrite-markdown-security.md
```

## Import/export note

```text
Construction/Domains/eWrite/ewrite-import-export.md
```

## Permissions matrix

```text
Construction/Domains/eWrite/ewrite-permissions.md
```

## Link and anchor contract

```text
Construction/Domains/eWrite/ewrite-link-anchor-contract.md
```

## Reportback

Use the standard reportback template.

---

# 21. Required evidence

## Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

## Tests

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 ./...
```

Use isolated `TEST_DATABASE_URL`.

## Static checks

```bash
git diff --check
```

Run `node --check` for changed inline scripts.

## Browser proof

Provide evidence for:

- Crew author opens Writer's Room;
- Cast cannot author;
- Markdown import;
- structure preview;
- draft save;
- revision creation;
- conflict warning;
- publish;
- Library browse;
- anonymous public read;
- authenticated read;
- Production-only denial;
- exact section direct link;
- existing Victory object opens exact rule section;
- export;
- unsafe Markdown does not execute.

## Scale proof

Record real or substantial fixture performance.

---

# 22. Pass criteria

Kernel 78 passes when:

- eWrite hierarchy exists;
- Writer's Room authoring works;
- Library reading works;
- Crew+ authority is enforced;
- Cast/Audience authoring is denied by default;
- Markdown paste and `.md` upload work;
- safe rendering is proven;
- sections/subsections are recognized;
- stable direct links work;
- revisions are append-forward;
- save conflicts do not overwrite newer work silently;
- published visibility works;
- drafts remain private;
- export works;
- account export/deletion integrate correctly;
- one existing Victory object links to an exact section;
- substantial manuscript or substantial fixture proof exists;
- full tests pass;
- no Kernel 76/77/77A regression exists.

---

# 23. Partial and fail rules

## PASS WITH FOLLOW-UP

Allowed when:

- complete feature vertical works;
- substantial synthetic manuscript passes;
- Grant has not yet supplied the real Socio manuscript.

The missing real import must be the only meaningful follow-up.

## PARTIAL

Use when:

- authoring works but Library does not;
- Library works but import does not;
- stable anchors are incomplete;
- unsafe Markdown remains;
- revisions or conflict protection are incomplete;
- visibility is not fully enforced;
- export is incomplete;
- no existing object link exists.

## FAIL

Use when:

- unsafe script/content executes;
- Production-private content leaks;
- Cast/Audience can author through client manipulation;
- drafts become public;
- save conflict destroys newer work;
- import loses source without warning;
- clean migration fails;
- account deletion or export regresses;
- Grant is locked out;
- full test suite is not credibly assessed.

---

# 24. Non-goals

Kernel 78 does not:

- import all Socio books perfectly;
- perform editorial restructuring for Grant;
- provide live simultaneous editing;
- provide comments or tracked changes;
- build the LMS;
- build storyboards;
- build vector drawing;
- build a marketplace;
- import construction documentation;
- create a full media platform;
- support arbitrary executable HTML;
- build public signup;
- replace Git-backed project records;
- build universal rules automation;
- add every possible document format.

---

# 25. Kernel-maker guidance

## 25.1 Build a publishing system, not a textarea page

The kernel succeeds only when writing, publishing, reading, linking, and exporting form one coherent system.

## 25.2 Do not confuse hierarchy with rigid schema

The interface hierarchy is fixed.

The database may use a more flexible typed tree where that improves durability.

## 25.3 Do not turn Grant into the data-entry worker

Large Markdown import must reduce manual splitting, not create it.

## 25.4 Do not trap the rules inside Victory

Source Markdown and export are mandatory.

## 25.5 Do not weaken theatrical permissions

Crew+ may write, but only in the correct scope.

## 25.6 Do not trust rendered HTML

Sanitize and test.

## 25.7 Do not let title edits destroy links

Stable section identity matters more than pretty slugs.

---

# 26. Immediate operator outcome

At completion, Grant must be able to answer:

1. Can I upload or paste a large Markdown rulebook?
2. Does eWrite recognize its sections and subsections?
3. Can Crew author without giving Cast writing access?
4. Can I save drafts safely?
5. Can two editors avoid silently overwriting each other?
6. Can I publish one work publicly and another only to a Production?
7. Can a player open the exact rule section from a Character, card, Cue, or item?
8. Can readers browse it in the Library?
9. Can unsafe Markdown execute?
10. Can I export the source back out?
11. Does account export include my eWritings?
12. Does account deletion handle authorship safely?
13. Can a substantial Socio manuscript load and remain usable?
14. Did the system preserve all security and recovery gains from Kernels 76–77A?

Kernel 78 succeeds when Victory becomes a practical place to write, publish, read, and directly reference rules and scripts without becoming a generic office suite.

---

## Operator amendments (recorded at execution start, 2026-08-02)

Locked with Grant before implementation:

1. **Two venues**: Library adopts the existing seeded `library` venue slot; Writer's Room is a new venue visible to Crew+ only. Grant supplies the Writer's Room map icon.
2. **Anonymous public reading deferred**: the `public/authenticated/production` visibility enum ships in schema and API, but `public` behaves as authenticated-only in this kernel. The §21 "anonymous public read" browser proof is deliberately deferred to a future kernel (first unauthenticated content API deserves its own security review). This is an operator-approved deviation, not a PARTIAL.
3. **Substantial manuscript**: `frontend/assets/rulesets/Sociov1_1.md` (already in the repository) is the real manuscript for §18.2.
4. **Editor/reader polish in scope**: formatting strip + Ctrl+S + word count; localStorage draft autosave/restore; reader scrollspy TOC + hover-¶ section links; FTS search snippets; print CSS.
