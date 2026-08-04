# eWrite Link and Anchor Contract (Kernel 78)

Implementation: `backend/internal/ewrite/sections.go` (reconcile),
`markdown.go` (anchor generation). The contract exists so a title edit never
destroys a link (kernel spec 25.7).

## Anchor generation

1. An explicit `{#id}` on a heading is preserved **verbatim** (charset
   `[A-Za-z0-9&'(),\-./:?_]+` — the measured Sociov1_1.md set). Explicit
   ids win collisions.
2. Headings without explicit ids get `slugifyAnchor(title)`: lowercase,
   spaces→hyphens, drop everything outside `[a-z0-9-]`, collapse runs.
3. Duplicates get deterministic `-2`, `-3`… suffixes, reported as import
   warnings.
4. Spacer headings (`## ` with no title — the Google Docs export idiom) are
   rendered for spacing but get no id, no section row, no TOC entry.

## Section identity across saves (reconcile, in the save transaction)

1. Match existing section rows by **anchor** first → row UUID survives,
   object links keep pointing at the same section.
2. Leftovers match by **(title, level)** in document order — the "same
   heading, new anchor" case (e.g. an explicit id was added). The row is
   updated in place and the OLD anchor becomes an `ewrite_anchor_aliases`
   row pointing at the same section.
3. Unmatched new headings insert; unmatched old sections delete (their
   aliases cascade).
4. An alias colliding with a live anchor is dropped — live wins; nothing
   silently redirects to unrelated content (spec 9.4).

## Reader resolution order

`exact section anchor → alias → publication top with a visible
"section not found" notice`. Implemented in `ResolveAnchor` + the reader
payload's `anchor_found`/`resolved_anchor` fields;
`frontend/venues/library/read.html` rewrites the URL hash to the resolved
anchor so copied links converge on the live spelling.

## Link forms

- Same-publication: ordinary Markdown `[text](#anchor)`.
- Cross-publication / external surface: route URL
  `/venues/library/read.html?pub=<publication-id>#<anchor>`. The UI
  generates these (reader hover-¶ buttons, editor link tool) — users never
  hand-write internal syntax.
- Victory object → section: `ewrite_object_links` row (typed FK binding).
  Resolution (`RuleLinksForEquipmentItems`) only returns **published**
  targets, so a draft's title can never leak into a Player payload; a
  removed heading degrades the binding to the publication top
  (`section_id` SET NULL), never a dangle.

## Broken links

Import/preview reports count internal links and warn on unsupported
constructs; unresolved fragments are preserved in source for correction,
never rewritten (spec 9.4, 5.4).
