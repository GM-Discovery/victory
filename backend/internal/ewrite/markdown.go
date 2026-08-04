package ewrite

// The eWrite render pipeline: Markdown source in, sanitized HTML + heading
// outline + plain search text out. One pure function with no database
// access; the store calls it inside the same transaction as every source
// write, so rendered_html can never go stale relative to source_markdown.
//
// Explicit {#anchor} ids are extracted by a line-level pre-pass rather than
// goldmark's parser.WithHeadingAttribute. Decision recorded from a measured
// spike (see Construction/eWrite/ewrite-markdown-security.md): goldmark's
// attribute lexer rejects the ( ) ? characters that Sociov1_1.md's
// Google-Docs-exported ids use throughout, leaving the literal "{#...}"
// text visible in the rendered heading. The pre-pass strips the id from the
// heading line (keeping line count identical so AST line positions still
// map), records line -> id, and the AST walk reattaches ids to heading
// nodes by source line number.

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Heading is one outline entry: a rendered heading that received an anchor.
// Spacer headings (no title text -- the Google Docs export uses empty "## "
// lines as vertical whitespace) are rendered but never appear here.
type Heading struct {
	Level    int    `json:"level"`
	Title    string `json:"title"`
	Anchor   string `json:"anchor"`
	Explicit bool   `json:"explicit"`
}

// RenderResult carries everything a save/import needs in one pass.
type RenderResult struct {
	HTML          string
	Outline       []Heading
	PlainText     string
	WordCount     int
	SpacerCount   int
	RawHTMLCount  int
	InternalLinks []string
	ExternalLinks []string
	ImageRefs     []string
	Warnings      []string
}

var explicitAnchorLine = regexp.MustCompile(`^(#{1,6})([ \t]+.*?)?[ \t]+\{#([^}]+)\}[ \t]*$`)
var fenceLine = regexp.MustCompile("^(```|~~~)")

var slugStrip = regexp.MustCompile(`[^a-z0-9\- _]`)
var slugSpaces = regexp.MustCompile(`[ _]+`)
var slugDashes = regexp.MustCompile(`-{2,}`)

// slugifyAnchor generates an anchor for a heading without an explicit id.
// Deterministic and documented (ewrite-link-anchor-contract.md): lowercase,
// spaces to hyphens, drop everything outside [a-z0-9-], collapse runs.
func slugifyAnchor(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugStrip.ReplaceAllString(s, "")
	s = slugSpaces.ReplaceAllString(s, "-")
	s = slugDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// stripExplicitAnchors is the pre-pass. It walks source line by line,
// tracking fenced code blocks so a "## heading {#id}" inside a fence is
// left alone, and returns the rewritten source (same line count), a map of
// zero-based line index -> explicit id, and warnings for ids it refused.
func stripExplicitAnchors(source string) (string, map[int]string, []string) {
	lines := strings.Split(source, "\n")
	ids := make(map[int]string)
	var warnings []string
	inFence := false
	fenceMark := ""

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if m := fenceLine.FindString(trimmed); m != "" {
			if !inFence {
				inFence = true
				fenceMark = m
			} else if strings.HasPrefix(trimmed, fenceMark) {
				inFence = false
			}
			continue
		}
		if inFence {
			continue
		}
		m := explicitAnchorLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		title := strings.TrimSpace(m[2])
		id := m[3]
		if title == "" {
			warnings = append(warnings, fmt.Sprintf("line %d: heading has an {#%s} anchor but no title; anchor ignored", i+1, id))
			continue
		}
		if !anchorIDPattern.MatchString(id) {
			warnings = append(warnings, fmt.Sprintf("line %d: explicit anchor {#%s} contains unsupported characters; a generated anchor was used instead", i+1, id))
			lines[i] = m[1] + " " + title
			continue
		}
		ids[i] = id
		lines[i] = m[1] + " " + title
	}
	return strings.Join(lines, "\n"), ids, warnings
}

func newGoldmark() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(),
	)
	// Deliberately no html.WithUnsafe(): goldmark escapes raw HTML by
	// default (first defense layer); bluemonday sanitizes the rendered
	// output (second layer).
}

// lineIndex maps a byte offset in the rewritten source to its zero-based
// line number via the precomputed starts table.
func lineIndex(starts []int, offset int) int {
	i := sort.Search(len(starts), func(i int) bool { return starts[i] > offset })
	return i - 1
}

func lineStarts(source string) []int {
	starts := []int{0}
	for i := 0; i < len(source); i++ {
		if source[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func nodeText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			buf.Write(t.Segment.Value(source))
		case *ast.String:
			buf.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

// Render is the single conversion path from Markdown source to safe HTML.
func Render(source string) (*RenderResult, error) {
	res := &RenderResult{}

	rewritten, explicitIDs, warnings := stripExplicitAnchors(source)
	res.Warnings = append(res.Warnings, warnings...)

	src := []byte(rewritten)
	starts := lineStarts(rewritten)

	md := newGoldmark()
	doc := md.Parser().Parse(text.NewReader(src))

	// Pass 1: collect headings in document order, resolve anchors.
	type headingRef struct {
		node  *ast.Heading
		title string
		id    string // explicit id or ""
	}
	var headings []headingRef
	var plain bytes.Buffer

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Heading:
			title := strings.TrimSpace(nodeText(t, src))
			ref := headingRef{node: t, title: title}
			if t.Lines().Len() > 0 {
				line := lineIndex(starts, t.Lines().At(0).Start)
				if id, ok := explicitIDs[line]; ok {
					ref.id = id
				}
			}
			headings = append(headings, ref)
			if title == "" {
				res.SpacerCount++
			}
		case *ast.RawHTML, *ast.HTMLBlock:
			res.RawHTMLCount++
		case *ast.Link:
			dest := string(t.Destination)
			if strings.HasPrefix(dest, "#") || (!strings.Contains(dest, "://") && !strings.HasPrefix(dest, "mailto:")) {
				res.InternalLinks = append(res.InternalLinks, dest)
			} else {
				res.ExternalLinks = append(res.ExternalLinks, dest)
			}
		case *ast.AutoLink:
			res.ExternalLinks = append(res.ExternalLinks, string(t.URL(src)))
		case *ast.Image:
			res.ImageRefs = append(res.ImageRefs, string(t.Destination))
		case *ast.Text:
			plain.Write(t.Segment.Value(src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				plain.WriteByte('\n')
			}
		case *ast.String:
			plain.Write(t.Value)
		}
		if n.Type() == ast.TypeBlock && n.Kind() != ast.KindDocument {
			plain.WriteByte('\n')
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}

	// Anchor assignment: explicit ids first (they win collisions), then
	// generated ones fill in around them. Duplicates get deterministic
	// -2/-3 suffixes with a warning (spec 9.1).
	used := make(map[string]bool)
	disambiguate := func(anchor string) string {
		if !used[anchor] {
			return anchor
		}
		for i := 2; ; i++ {
			cand := fmt.Sprintf("%s-%d", anchor, i)
			if !used[cand] {
				return cand
			}
		}
	}
	anchors := make([]string, len(headings))
	for i, h := range headings {
		if h.id == "" {
			continue
		}
		anchor := disambiguate(h.id)
		if anchor != h.id {
			res.Warnings = append(res.Warnings, fmt.Sprintf("duplicate explicit anchor {#%s}; heading %q was disambiguated to %q", h.id, h.title, anchor))
		}
		used[anchor] = true
		anchors[i] = anchor
	}
	for i, h := range headings {
		if anchors[i] != "" || h.title == "" {
			continue
		}
		base := slugifyAnchor(h.title)
		if base == "" {
			base = fmt.Sprintf("section-%d", i+1)
		}
		anchor := disambiguate(base)
		if anchor != base {
			res.Warnings = append(res.Warnings, fmt.Sprintf("duplicate heading %q; anchor disambiguated to %q", h.title, anchor))
		}
		used[anchor] = true
		anchors[i] = anchor
	}

	for i, h := range headings {
		if anchors[i] == "" {
			continue // spacer: rendered, but no id, no outline entry
		}
		h.node.SetAttributeString("id", []byte(anchors[i]))
		res.Outline = append(res.Outline, Heading{
			Level:    h.node.Level,
			Title:    h.title,
			Anchor:   anchors[i],
			Explicit: h.id != "",
		})
	}

	if res.RawHTMLCount > 0 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%d raw HTML fragment(s) found; raw HTML is never executed (escaped or removed)", res.RawHTMLCount))
	}

	var rendered bytes.Buffer
	if err := md.Renderer().Render(&rendered, src, doc); err != nil {
		return nil, err
	}
	res.HTML = renderPolicy.Sanitize(rendered.String())

	// ast.Text segments carry raw source bytes, so CommonMark backslash
	// escapes (the Google Docs export escapes most punctuation) are still
	// present; unescape them so search text matches what readers see.
	res.PlainText = backslashEscape.ReplaceAllString(plain.String(), "$1")
	res.WordCount = len(strings.Fields(res.PlainText))
	return res, nil
}

// CommonMark's escapable set: any ASCII punctuation may be backslash-escaped.
var backslashEscape = regexp.MustCompile("\\\\([!-/:-@\\[-`{-~])")

var renderPolicy = newRenderPolicy()
