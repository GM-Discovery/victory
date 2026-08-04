package ewrite

// Kernel 78 scale proof (spec 7.5, 18.2): the importer proven against the
// real Socio v1.1 manuscript already shipped in the repository -- not a
// 20-line fixture. Measures parse+render, full save (render + revision +
// section reconcile in one tx), and re-save, and pins the structural
// numbers the reportback records: explicit anchors preserved verbatim,
// spacer headings skipped, duplicate disambiguations warned.

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

const manuscriptPath = "../../../frontend/assets/rulesets/Sociov1_1.md"

func TestImportRealSocioManuscriptAtScale(t *testing.T) {
	raw, err := os.ReadFile(manuscriptPath)
	if err != nil {
		t.Fatalf("the Socio v1.1 manuscript must exist for the scale proof: %v", err)
	}
	source, err := normalizeSource(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	// Stage 1: pure render.
	renderStart := time.Now()
	res, err := Render(source)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	renderDur := time.Since(renderStart)

	// Structural truths of the v1.1 export. If the manuscript file is
	// replaced, re-measure and update -- these pin importer behavior, not
	// the book.
	if len(res.Outline) < 700 {
		t.Fatalf("expected 700+ outline headings, got %d", len(res.Outline))
	}
	explicit := 0
	for _, h := range res.Outline {
		if h.Explicit {
			explicit++
		}
	}
	if explicit != 137 {
		t.Fatalf("expected all 137 explicit {#anchors} preserved, got %d", explicit)
	}
	for _, want := range []string{"stories-of-us", "table-of-contents", "why-this-game?", "(an-open-source-collaborative-tabletop-role-playing-system)"} {
		found := false
		for _, h := range res.Outline {
			if h.Anchor == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("explicit anchor %q not preserved verbatim", want)
		}
	}
	if res.SpacerCount < 100 {
		t.Fatalf("expected 100+ spacer headings skipped, got %d", res.SpacerCount)
	}
	if res.WordCount < 40000 {
		t.Fatalf("expected 40k+ words, got %d", res.WordCount)
	}
	if strings.Contains(res.HTML, "{#") {
		t.Fatal("rendered HTML leaks literal {#anchor} text")
	}

	// Stage 2: full save path against the database.
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_scale_crew")
	grantRole(t, pool, loc, crew, "crew")
	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Socio Scale Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Socio v1.1 Scale Proof")

	saveStart := time.Now()
	r1 := mustSave(t, pool, crew, pub.ID, source, "")
	saveDur := time.Since(saveStart)

	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) != len(res.Outline) {
		t.Fatalf("expected %d section rows, got %d", len(res.Outline), len(sections))
	}

	// Stage 3: re-save with one heading touched -- the reconcile path over
	// 700+ existing rows, the everyday editing case.
	touched := strings.Replace(source, "## Overview {#overview}", "## Overview (Revised) {#overview}", 1)
	resaveStart := time.Now()
	mustSave(t, pool, crew, pub.ID, touched, r1.RevisionID)
	resaveDur := time.Since(resaveStart)

	budget := 15 * time.Second
	if renderDur > budget || saveDur > budget || resaveDur > budget {
		t.Fatalf("scale budget exceeded: render=%s save=%s resave=%s (budget %s each)", renderDur, saveDur, resaveDur, budget)
	}

	t.Logf("SCALE PROOF: bytes=%d words=%d headings=%d (explicit=%d spacers=%d) render=%s save=%s resave=%s warnings=%d html_bytes=%d",
		len(source), res.WordCount, len(res.Outline), explicit, res.SpacerCount,
		renderDur, saveDur, resaveDur, len(res.Warnings), len(res.HTML))
}
