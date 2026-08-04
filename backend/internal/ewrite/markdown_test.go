package ewrite

// Markdown safety and anchor-contract tests (kernel spec 17.4, 8.2, 9.1).
// Pure -- no database. Each forbidden vector from the spec gets a named
// fixture proving no active content survives the pipeline; the positive
// tests prove the hostile-but-legitimate Google-Docs anchor charset and
// fragment links DO survive, because a sanitizer that strips the manuscript's
// own ids would fail the kernel just as surely as one that passes script.

import (
	"strings"
	"testing"
)

func renderOrFail(t *testing.T, source string) *RenderResult {
	t.Helper()
	res, err := Render(source)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return res
}

func mustNotContain(t *testing.T, html string, needles ...string) {
	t.Helper()
	lower := strings.ToLower(html)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			t.Fatalf("sanitized HTML must not contain %q\nhtml: %s", n, html)
		}
	}
}

func mustContain(t *testing.T, html string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(html, n) {
			t.Fatalf("sanitized HTML must contain %q\nhtml: %s", n, html)
		}
	}
}

// --- Malicious fixtures: one test per spec 17.4 vector ---

func TestSanitizeScriptTag(t *testing.T) {
	res := renderOrFail(t, "hello\n\n<script>alert(1)</script>\n\nworld")
	mustNotContain(t, res.HTML, "<script", "alert(1)</script>")
	if res.RawHTMLCount == 0 {
		t.Fatal("expected raw HTML to be counted and reported")
	}
}

func TestSanitizeInlineScriptInParagraph(t *testing.T) {
	res := renderOrFail(t, "before <script>document.cookie</script> after")
	mustNotContain(t, res.HTML, "<script")
}

func TestSanitizeEventHandlerAttributes(t *testing.T) {
	res := renderOrFail(t, `<p onclick="steal()">x</p>`+"\n\n"+`<img src="/api/assets/00000000-0000-0000-0000-000000000000/content" onerror="alert(1)">`)
	mustNotContain(t, res.HTML, "onclick", "onerror", "steal()")
}

func TestSanitizeJavascriptURL(t *testing.T) {
	res := renderOrFail(t, `[click me](javascript:alert(document.cookie))`)
	mustNotContain(t, res.HTML, "javascript:")
	mustContain(t, res.HTML, "click me")
}

func TestSanitizeVbscriptAndFileURL(t *testing.T) {
	res := renderOrFail(t, `[a](vbscript:msgbox) and [b](file:///etc/passwd)`)
	mustNotContain(t, res.HTML, "vbscript:", "file:")
}

func TestSanitizeIframe(t *testing.T) {
	res := renderOrFail(t, `<iframe src="https://evil.example/"></iframe>`)
	mustNotContain(t, res.HTML, "<iframe")
}

func TestSanitizeObjectAndEmbed(t *testing.T) {
	res := renderOrFail(t, "<object data=\"x\"></object>\n\n<embed src=\"x\">")
	mustNotContain(t, res.HTML, "<object", "<embed")
}

func TestSanitizeForm(t *testing.T) {
	res := renderOrFail(t, `<form action="https://evil.example/phish"><input name="password" type="password"></form>`)
	mustNotContain(t, res.HTML, "<form", "password")
}

func TestSanitizeActiveSVG(t *testing.T) {
	res := renderOrFail(t, `<svg onload="alert(1)"><script>alert(2)</script></svg>`)
	mustNotContain(t, res.HTML, "<svg", "onload", "<script")
}

func TestSanitizeDataURI(t *testing.T) {
	// A data: URI must never survive as a live href/src attribute. (It may
	// remain visible as inert escaped text when the markdown around it
	// fails to parse -- text is not a vector.)
	res := renderOrFail(t, `[x](data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==) ![y](data:image/svg+xml,<svg onload=alert(1)>)`)
	mustNotContain(t, res.HTML, `href="data:`, `src="data:`, "<svg", "onload")
}

func TestSanitizeStyleInjection(t *testing.T) {
	res := renderOrFail(t, `<p style="background:url(javascript:alert(1));position:fixed">x</p>`+"\n\n<style>body{display:none}</style>")
	mustNotContain(t, res.HTML, "position:fixed", "<style", "display:none")
}

func TestSanitizeMetaRefresh(t *testing.T) {
	res := renderOrFail(t, `<meta http-equiv="refresh" content="0;url=https://evil.example/">`)
	mustNotContain(t, res.HTML, "<meta", "http-equiv")
}

func TestSanitizeMalformedHTML(t *testing.T) {
	res := renderOrFail(t, "<scr<script>ipt>alert(1)</scr</script>ipt>\n\n<<img src=x onerror=alert(1)//>")
	mustNotContain(t, res.HTML, "<script", "onerror")
}

func TestSanitizeExternalImageStripped(t *testing.T) {
	res := renderOrFail(t, `![tracker](https://evil.example/pixel.gif)`)
	mustNotContain(t, res.HTML, "evil.example")
	if len(res.ImageRefs) != 1 {
		t.Fatalf("expected the external image to be reported, got %v", res.ImageRefs)
	}
}

func TestSanitizeNestedCodeBlockNotExecuted(t *testing.T) {
	// Script inside a fenced code block must render as escaped text --
	// visible, not active.
	res := renderOrFail(t, "```html\n<script>alert(1)</script>\n```")
	mustNotContain(t, res.HTML, "<script>")
	mustContain(t, res.HTML, "&lt;script&gt;")
}

// --- Positive contract: what MUST survive ---

func TestExplicitAnchorHostileCharsetSurvives(t *testing.T) {
	src := strings.Join([]string{
		"# Socio- {#socio-}",
		"",
		"## Why This Game? {#why-this-game?}",
		"",
		"## Core Concepts: What Makes This Game Unique {#core-concepts:-what-makes-this-game-unique}",
		"",
		"## (An Open-Source Collaborative Tabletop Role Playing System) {#(an-open-source-collaborative-tabletop-role-playing-system)}",
		"",
		"## Generational Wealth Chart (3d20 Exploding up to Once each) {#generational-wealth-chart-(3d20-exploding-up-to-once-each)}",
	}, "\n")
	res := renderOrFail(t, src)

	mustContain(t, res.HTML,
		`id="socio-"`,
		`id="why-this-game?"`,
		`id="core-concepts:-what-makes-this-game-unique"`,
		`id="(an-open-source-collaborative-tabletop-role-playing-system)"`,
		`id="generational-wealth-chart-(3d20-exploding-up-to-once-each)"`,
	)
	// The literal {#...} text must NOT leak into the rendered heading.
	mustNotContain(t, res.HTML, "{#")

	if len(res.Outline) != 5 {
		t.Fatalf("expected 5 outline entries, got %d: %+v", len(res.Outline), res.Outline)
	}
	for _, h := range res.Outline {
		if !h.Explicit {
			t.Fatalf("expected all anchors explicit, got %+v", h)
		}
	}
}

func TestFragmentLinksWithHostileCharsSurvive(t *testing.T) {
	res := renderOrFail(t, `See [Why This Game?](#why-this-game?) and [Wealth](#generational-wealth-chart-(3d20-exploding-up-to-once-each))`)
	mustContain(t, res.HTML, `href="#why-this-game?"`)
	if len(res.InternalLinks) != 2 {
		t.Fatalf("expected 2 internal links, got %v", res.InternalLinks)
	}
}

func TestGeneratedAnchors(t *testing.T) {
	res := renderOrFail(t, "# My Title\n\n## A Section Name\n\n### Sub-Section: One!")
	mustContain(t, res.HTML, `id="my-title"`, `id="a-section-name"`, `id="sub-section-one"`)
	for _, h := range res.Outline {
		if h.Explicit {
			t.Fatalf("expected generated anchors, got explicit %+v", h)
		}
	}
}

func TestDuplicateHeadingsDisambiguated(t *testing.T) {
	res := renderOrFail(t, "## Overview\n\n## Overview\n\n## Overview")
	mustContain(t, res.HTML, `id="overview"`, `id="overview-2"`, `id="overview-3"`)
	if len(res.Warnings) < 2 {
		t.Fatalf("expected disambiguation warnings, got %v", res.Warnings)
	}
}

func TestExplicitAnchorWinsCollision(t *testing.T) {
	// A generated anchor colliding with a later explicit one: the explicit
	// id keeps the canonical spelling, the generated one gets the suffix.
	res := renderOrFail(t, "## Overview\n\n## The Overview Chapter {#overview}")
	mustContain(t, res.HTML, `id="overview">The Overview Chapter`, `id="overview-2">Overview`)
}

func TestSpacerHeadingsRenderedButNotInOutline(t *testing.T) {
	res := renderOrFail(t, "# Title {#title}\n\n## \n\n## Real Section\n\n## ")
	if res.SpacerCount != 2 {
		t.Fatalf("expected 2 spacer headings, got %d", res.SpacerCount)
	}
	if len(res.Outline) != 2 {
		t.Fatalf("expected 2 outline entries (title + real section), got %+v", res.Outline)
	}
}

func TestAnchorInsideCodeFenceIgnored(t *testing.T) {
	res := renderOrFail(t, "```\n## Not A Heading {#not-an-anchor}\n```\n\n## Real {#real}")
	mustNotContain(t, res.HTML, `id="not-an-anchor"`)
	mustContain(t, res.HTML, `id="real"`, "{#not-an-anchor}")
}

func TestVictoryAssetImageSurvives(t *testing.T) {
	res := renderOrFail(t, `![map](/api/assets/1b4e28ba-2fa1-11d2-883f-0016d3cca427/content)`)
	mustContain(t, res.HTML, `src="/api/assets/1b4e28ba-2fa1-11d2-883f-0016d3cca427/content"`, `alt="map"`)
}

func TestGFMTableSurvives(t *testing.T) {
	res := renderOrFail(t, "| a | b |\n|---|---|\n| 1 | 2 |")
	mustContain(t, res.HTML, "<table>", "<td>1</td>")
}

func TestExternalLinkGetsNofollow(t *testing.T) {
	res := renderOrFail(t, `[ext](https://example.com/) and [frag](#local)`)
	mustContain(t, res.HTML, `rel="nofollow"`)
	if !strings.Contains(res.HTML, `href="#local"`) {
		t.Fatalf("fragment link must survive without nofollow damage: %s", res.HTML)
	}
}

func TestWordCountAndPlainText(t *testing.T) {
	res := renderOrFail(t, "# Title\n\nOne two three.\n\n- four\n- five")
	if res.WordCount != 6 {
		t.Fatalf("expected 6 words, got %d (plain: %q)", res.WordCount, res.PlainText)
	}
	if strings.Contains(res.PlainText, "#") {
		t.Fatalf("plain text must not contain markdown syntax: %q", res.PlainText)
	}
}

func TestEscapedPunctuationRendered(t *testing.T) {
	// Google Docs exports backslash-escape punctuation; the renderer must
	// unescape it for display and search text.
	res := renderOrFail(t, `Socio\- is an RPG\.`)
	mustContain(t, res.HTML, "Socio- is an RPG.")
	if strings.Contains(res.PlainText, `\-`) {
		t.Fatalf("plain text must unescape punctuation: %q", res.PlainText)
	}
}
