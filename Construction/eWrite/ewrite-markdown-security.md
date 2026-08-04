# eWrite Markdown Security (Kernel 78)

The security design Kernel 76 required before eWrite existed. Implementation:
`backend/internal/ewrite/markdown.go` (pipeline) and `markdown_policy.go`
(sanitizer policy). Tests: `markdown_test.go` — one named test per attack
vector, plus positive tests proving the hostile-but-legitimate manuscript
charset survives.

## Architecture: render server-side, sanitize server-side, always

```text
source markdown
→ explicit-anchor pre-pass (regex, fence-aware)
→ goldmark (GFM + Footnote; raw HTML ESCAPED — no html.WithUnsafe, layer 1)
→ AST walk (outline, links, images, plain text, word count)
→ heading id attachment (explicit id or generated slug, deduped -2/-3)
→ bluemonday sanitize (layer 2)
→ rendered_html cache, written only inside the save transaction
```

The client never renders Markdown to DOM. The editor preview POSTs to
`/api/ewrite/preview` and injects server-sanitized HTML; the Library reader
injects the cached server-sanitized HTML. The client-side outline parser
(`frontend/lib/ewrite/outline.js`) is display-only convenience and is
documented as such in its header.

## Dependencies (first Markdown deps in the repo)

- `github.com/yuin/goldmark v1.8.5` (MIT) — parser/renderer.
- `github.com/microcosm-cc/bluemonday v1.0.27` (BSD-3) — sanitizer
  (pulls douceur, gorilla/css, x/net).

Pinned in `backend/go.mod`; tested against malicious fixtures per kernel
spec 8.5.

## The spike that chose the pre-pass (recorded decision)

Sociov1_1.md's Google-Docs-exported anchors use `( ) : ? & ' , . /` —
measured, not guessed (`grep -o '{#[^}]*}'` over the manuscript, 137
anchors). A spike test proved goldmark's `parser.WithHeadingAttribute()`
**fails** on `?` and `(` ids: it leaves the literal `{#...}` text visible in
the heading and generates a mangled doubled auto-id (colons pass). So
explicit anchors are extracted by a line-level regex pre-pass that is
fence-aware, keeps line count identical (AST line positions still map), and
records line → id; the AST walk reattaches ids to heading nodes.

## Policy (built from empty `NewPolicy()`, not `UGCPolicy()`)

UGCPolicy allows images from any URL; bluemonday policies are additive with
no way to narrow, so eWrite starts from nothing and allows deliberately:

- Structure: h1–h6, p, br, hr, blockquote, ul/ol/li, pre/code (+
  `language-*` class), em/strong/del/s, tables (+ align), sup/section.
- Heading `id` matching `^[A-Za-z0-9&'(),\-./:?_]+$` — exactly the measured
  manuscript charset plus `_`.
- Links: http/https/mailto + relative (fragments included — the whole
  system exists to serve `#section` links); `rel=nofollow` on fully
  qualified external links only.
- Images: `src` must match `^/api/assets/<uuid>/content(\?variant=...)?$` —
  Victory assets only, no external images, no tracking pixels. Accepted
  caveat, recorded: the asset read path gates only `map`-type assets, so
  embedded images are link-knowable; acceptable for Kernel 78.
- GFM task-list checkboxes: `input[type=checkbox][checked][disabled]` only —
  inert; no form element is ever allowed.
- Footnotes: goldmark's bounded output shape (fn:/fnref: ids, footnote-*
  classes, doc-* roles).

Denied by construction (never allowed, each proven by a named test):
script, iframe, object, embed, event handlers, `javascript:`/`data:`/
`vbscript:`/`file:` URLs, style injection/`<style>`, forms, active SVG,
meta refresh, malformed-HTML smuggling.

## Limits

- Source cap 2 MiB per publication (~6× the full Socio rulebook), inside
  the global 4 MiB body cap; import endpoint has its own MaxBytesReader.
- `search_tsv` guards the 1 MB tsvector input limit via `left(…, 800000)`.
- Strict UTF-8 on import; CRLF normalized; BOM stripped.
