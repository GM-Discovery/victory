package ewrite

// The bluemonday policy for eWrite rendered HTML, isolated here so the
// malicious-fixture tests in markdown_test.go pin its behavior directly.
//
// Built from an empty NewPolicy(), NOT UGCPolicy(): UGCPolicy allows images
// from any standard URL, and eWrite's image contract is Victory-managed
// assets only (kernel spec 8.4). bluemonday policies are additive with no
// way to narrow an inherited allowance, so the safe direction is to start
// from nothing and allow deliberately.
//
// Everything the kernel forbids (spec 8.2) -- script, iframe, object, embed,
// event handlers, inline JavaScript, style injection, forms, active SVG --
// is denied by construction: it was simply never allowed. The tests still
// prove each vector by name.

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// anchorIDPattern is the exact charset observed across Sociov1_1.md's 137
// Google-Docs-exported {#anchor} ids (& ' ( ) , - . / : ? plus
// alphanumerics), extended with '_' for generated anchors. Measured, not
// guessed: grep -o '{#[^}]*}' over the manuscript.
var anchorIDPattern = regexp.MustCompile(`^[A-Za-z0-9&'(),\-./:?_]+$`)

// assetImageSrcPattern: images may only reference Victory's asset read
// route. No external images -- no tracking pixels, no mixed content, no
// remote fetches from a rules page (spec 8.4 "prefer Victory-managed
// assets"; external URLs deliberately not allowed in Kernel 78).
var assetImageSrcPattern = regexp.MustCompile(`^/api/assets/[0-9a-fA-F-]{36}/content(\?variant=[a-z0-9]+)?$`)

var languageClassPattern = regexp.MustCompile(`^language-[a-zA-Z0-9+#-]+$`)
var footnoteIDPattern = regexp.MustCompile(`^fn(ref)?(%3A|:)[A-Za-z0-9-]+$`)
var footnoteClassPattern = regexp.MustCompile(`^footnote(s|-ref|-backref)?$`)
var footnoteRolePattern = regexp.MustCompile(`^doc-(endnotes|noteref|backlink)$`)
var textAlignPattern = regexp.MustCompile(`^(left|center|right)$`)

func newRenderPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	// Block structure.
	p.AllowElements(
		"h1", "h2", "h3", "h4", "h5", "h6",
		"p", "br", "hr",
		"blockquote",
		"ul", "ol", "li",
		"pre",
		"em", "strong", "del", "s",
		"table", "thead", "tbody", "tr", "th", "td",
		"sup", "section",
	)
	p.AllowAttrs("start").Matching(bluemonday.Integer).OnElements("ol")

	// Headings carry the stable section anchors.
	p.AllowAttrs("id").Matching(anchorIDPattern).OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	// Code blocks: goldmark emits <pre><code class="language-x">.
	p.AllowElements("code")
	p.AllowAttrs("class").Matching(languageClassPattern).OnElements("code")

	// GFM table cell alignment.
	p.AllowAttrs("align").Matching(textAlignPattern).OnElements("th", "td")
	p.AllowStyles("text-align").MatchingEnum("left", "center", "right").OnElements("th", "td")

	// Links: http/https/mailto plus relative (which includes the #fragment
	// section links this whole system exists to serve). External links get
	// rel=nofollow; javascript:, data:, vbscript:, file: die here.
	p.AllowAttrs("href").OnElements("a")
	p.AllowStandardURLs()
	p.AllowURLSchemes("http", "https", "mailto")
	p.AllowRelativeURLs(true)
	p.RequireNoFollowOnFullyQualifiedLinks(true)

	// Images: Victory assets only.
	p.AllowAttrs("src").Matching(assetImageSrcPattern).OnElements("img")
	p.AllowAttrs("alt", "title").OnElements("img")

	// GFM task list checkboxes render as disabled inputs -- inert, not a
	// form (no form element is ever allowed).
	p.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	p.AllowAttrs("checked", "disabled").Matching(regexp.MustCompile(`^$|^checked$|^disabled$`)).OnElements("input")

	// Footnotes (goldmark extension.Footnote): sup/section/li/a wiring with
	// fn:/fnref: ids -- bounded, well-known output shape.
	p.AllowAttrs("id").Matching(footnoteIDPattern).OnElements("li", "sup", "a", "section")
	p.AllowAttrs("class").Matching(footnoteClassPattern).OnElements("a", "section", "sup")
	p.AllowAttrs("role").Matching(footnoteRolePattern).OnElements("a", "section")

	return p
}
