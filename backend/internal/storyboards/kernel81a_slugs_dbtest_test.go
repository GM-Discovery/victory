package storyboards

// Kernel 81A: deterministic serialized slugs for structural objects.
// Requires TEST_DATABASE_URL (internal/dbtest safety gate).

import (
	"context"
	"sync"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestSlugifyLabelDeterministicAndIdempotent(t *testing.T) {
	cases := map[string]string{
		"Scene":        "scene",
		"  Scene  ":    "scene",
		"Scene Two!!":  "scene-two",
		"Ünïcode Name": "n-code-name",
		"":             "untitled",
		"   ":          "untitled",
		"already-slug": "already-slug",
	}
	for input, want := range cases {
		got := SlugifyLabel(input)
		if got != want {
			t.Fatalf("SlugifyLabel(%q) = %q, want %q", input, got, want)
		}
		// Idempotent: slugifying an already-slug-shaped string is a no-op.
		if again := SlugifyLabel(got); again != got {
			t.Fatalf("SlugifyLabel not idempotent: SlugifyLabel(%q) = %q, want %q", got, again, got)
		}
	}
}

func TestDuplicateColumnLabelsSerializeUniquely(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Slug Board")

	// Column 1 already exists from board creation (default column).
	c2, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add Scene: %v", err)
	}
	c3, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add second Scene: %v", err)
	}
	c4, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add third Scene: %v", err)
	}

	if c2.Slug != "scene" {
		t.Fatalf("expected first Scene slug 'scene', got %q", c2.Slug)
	}
	if c3.Slug != "scene-2" {
		t.Fatalf("expected second Scene slug 'scene-2', got %q", c3.Slug)
	}
	if c4.Slug != "scene-3" {
		t.Fatalf("expected third Scene slug 'scene-3', got %q", c4.Slug)
	}
}

func TestSlugSkipsIndependentlyExistingCandidate(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Slug Skip Board")

	// "Scene 2" independently slugifies to "scene-2".
	independent, err := AddColumn(ctx, pool, owner, board.ID, "Scene 2")
	if err != nil {
		t.Fatalf("add Scene 2: %v", err)
	}
	if independent.Slug != "scene-2" {
		t.Fatalf("expected 'Scene 2' to slugify to 'scene-2', got %q", independent.Slug)
	}

	first, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add Scene: %v", err)
	}
	second, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add second Scene: %v", err)
	}

	if first.Slug != "scene" {
		t.Fatalf("expected 'scene', got %q", first.Slug)
	}
	// scene-2 is taken by the independent column -- must skip straight to
	// scene-3, never collide with it.
	if second.Slug != "scene-3" {
		t.Fatalf("expected collision-avoidance to skip taken 'scene-2' and land on 'scene-3', got %q", second.Slug)
	}
}

func TestDuplicateVisibleLabelsAreNeverRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Dup Label Board")

	b1, err := AddBand(ctx, pool, owner, board.ID, "Act")
	if err != nil {
		t.Fatalf("add band 1: %v", err)
	}
	b2, err := AddBand(ctx, pool, owner, board.ID, "Act")
	if err != nil {
		t.Fatalf("duplicate-labeled band should be accepted, got error: %v", err)
	}
	if b1.Label != b2.Label {
		t.Fatalf("expected both bands to keep the identical visible label 'Act', got %q and %q", b1.Label, b2.Label)
	}
	if b1.Slug == b2.Slug {
		t.Fatalf("expected distinct slugs despite identical labels, both were %q", b1.Slug)
	}
}

func TestSlugStableAcrossReloadsAndReorder(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Stability Board")

	created, err := AddColumn(ctx, pool, owner, board.ID, "Scene")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}
	originalSlug := created.Slug

	reloaded, err := loadColumn(ctx, pool, board.ID, created.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Slug != originalSlug {
		t.Fatalf("slug changed across reload: %q -> %q", originalSlug, reloaded.Slug)
	}

	// Renaming the human label must never touch the slug.
	renamed, err := RenameColumn(ctx, pool, owner, board.ID, created.ID, "Completely Different Title")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Slug != originalSlug {
		t.Fatalf("slug changed after rename: %q -> %q", originalSlug, renamed.Slug)
	}

	// Reordering must never touch any column's slug.
	cols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	ids := make([]string, len(cols))
	for i, c := range cols {
		ids[i] = c.ID
	}
	// Reverse the order.
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	if err := ReorderColumns(ctx, pool, owner, board.ID, ids); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	afterReorder, err := loadColumn(ctx, pool, board.ID, created.ID)
	if err != nil {
		t.Fatalf("reload after reorder: %v", err)
	}
	if afterReorder.Slug != originalSlug {
		t.Fatalf("slug changed after reorder: %q -> %q", originalSlug, afterReorder.Slug)
	}
}

func TestRowSlugScopedPerBandNotBoard(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Row Scope Board")

	bands, err := ListBands(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list bands: %v", err)
	}
	bandA := bands[0].ID
	bandB, err := AddBand(ctx, pool, owner, board.ID, "Second Band")
	if err != nil {
		t.Fatalf("add second band: %v", err)
	}

	rowA, err := AddRow(ctx, pool, owner, board.ID, bandA, "Beat")
	if err != nil {
		t.Fatalf("add row in band A: %v", err)
	}
	rowB, err := AddRow(ctx, pool, owner, board.ID, bandB.ID, "Beat")
	if err != nil {
		t.Fatalf("add row in band B: %v", err)
	}

	// Different bands: identical label should be allowed to reuse the
	// same base slug, since row uniqueness is scoped to the band.
	if rowA.Slug != "beat" || rowB.Slug != "beat" {
		t.Fatalf("expected both rows to independently get slug 'beat' in their own band, got %q and %q", rowA.Slug, rowB.Slug)
	}

	// Within the SAME band, a duplicate label must still collision-avoid.
	rowA2, err := AddRow(ctx, pool, owner, board.ID, bandA, "Beat")
	if err != nil {
		t.Fatalf("add second row in band A: %v", err)
	}
	if rowA2.Slug != "beat-2" {
		t.Fatalf("expected second same-band 'Beat' to get 'beat-2', got %q", rowA2.Slug)
	}
}

// TestConcurrentDuplicateCreationNeverProducesSameSlug proves the slug
// allocator specifically is race-free under real concurrent callers.
//
// AddColumn/AddBand/AddRow carry a separate, pre-existing (Kernel 80, not
// Kernel 81A) race: sort_order is computed via a plain `SELECT COUNT(*)`
// then used in the INSERT, with no locking between the two -- two
// concurrent calls can compute the same count and one loses to the
// table's own UNIQUE(storyboard_id, sort_order) constraint. That is a
// real, separate, out-of-scope bug (see operator-notes.md); this kernel's
// job is only to make slug allocation itself race-safe, not to fix it.
// So this test does what a real client already has to do against that
// pre-existing race -- retry the whole call on a transient failure --
// and then asserts the thing actually in scope: across every eventually-
// successful creation, no two ever got the same slug.
func TestConcurrentDuplicateCreationNeverProducesSameSlug(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Concurrency Slug Board")

	const n = 12
	const maxClientRetries = 20
	var wg sync.WaitGroup
	results := make([]*StoryboardColumn, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var c *StoryboardColumn
			var err error
			for attempt := 0; attempt < maxClientRetries; attempt++ {
				c, err = AddColumn(ctx, pool, owner, board.ID, "Scene")
				if err == nil {
					break
				}
			}
			results[i] = c
			errs[i] = err
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent AddColumn %d failed even after retries: %v", i, err)
		}
		slug := results[i].Slug
		if seen[slug] {
			t.Fatalf("duplicate slug %q produced by concurrent creation", slug)
		}
		seen[slug] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct slugs, got %d", n, len(seen))
	}

	// Confirm against the database directly too -- the real correctness
	// guarantee is the unique index, not just this process's bookkeeping.
	var distinctCount, totalCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(DISTINCT slug), COUNT(*) FROM storyboard_columns WHERE storyboard_id = $1 AND title = 'Scene'`, board.ID).Scan(&distinctCount, &totalCount); err != nil {
		t.Fatalf("count distinct slugs: %v", err)
	}
	if distinctCount != totalCount || totalCount != n {
		t.Fatalf("expected %d rows with %d distinct slugs in the DB, got total=%d distinct=%d", n, n, totalCount, distinctCount)
	}
}

func TestExportPreservesDistinctSlugsForDuplicateLabels(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Export Slug Board")

	if _, err := AddColumn(ctx, pool, owner, board.ID, "Scene"); err != nil {
		t.Fatalf("add column: %v", err)
	}
	if _, err := AddColumn(ctx, pool, owner, board.ID, "Scene"); err != nil {
		t.Fatalf("add duplicate column: %v", err)
	}
	if _, err := AddBand(ctx, pool, owner, board.ID, "Act"); err != nil {
		t.Fatalf("add band: %v", err)
	}

	doc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	sceneCols := 0
	slugsSeen := map[string]bool{}
	for _, c := range doc.Columns {
		if c.Title == "Scene" {
			sceneCols++
			if c.Slug == "" {
				t.Fatalf("exported column %q has an empty slug", c.ID)
			}
			if slugsSeen[c.Slug] {
				t.Fatalf("export contains a duplicate slug %q across distinct columns", c.Slug)
			}
			slugsSeen[c.Slug] = true
		}
		if c.ID == "" {
			t.Fatalf("exported column missing its stable UUID")
		}
	}
	if sceneCols != 2 {
		t.Fatalf("expected 2 exported 'Scene' columns, got %d", sceneCols)
	}

	for _, b := range doc.Bands {
		if b.Slug == "" {
			t.Fatalf("exported band %q has an empty slug", b.ID)
		}
	}
	for _, r := range doc.Rows {
		if r.Slug == "" {
			t.Fatalf("exported row %q has an empty slug", r.ID)
		}
	}
}

// TestExistingStoryboardsBackfillWithoutMigrationDamage simulates a board
// created before migration 094: columns/bands/rows inserted with slug
// left NULL, exactly what a pre-Kernel-81A row looks like. Runs the same
// backfill bootstrap main.go calls on every boot and proves every row
// gets a deterministic slug with zero change to id/title/label/sort
// order/description -- "existing Storyboards continue loading without
// migration damage."
func TestExistingStoryboardsBackfillWithoutMigrationDamage(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Pre-Migration Board")

	// Directly set the default column/band/row's slug back to NULL,
	// simulating a row that predates migration 094, and add a second,
	// duplicate-labeled column the same way -- also with no slug.
	if _, err := pool.Exec(ctx, `UPDATE storyboard_columns SET slug = NULL WHERE storyboard_id = $1`, board.ID); err != nil {
		t.Fatalf("simulate pre-migration column: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE storyboard_bands SET slug = NULL WHERE storyboard_id = $1`, board.ID); err != nil {
		t.Fatalf("simulate pre-migration band: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE storyboard_rows SET slug = NULL WHERE storyboard_id = $1`, board.ID); err != nil {
		t.Fatalf("simulate pre-migration row: %v", err)
	}

	bandsBefore, err := ListBands(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list bands before: %v", err)
	}
	var secondColID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_columns (storyboard_id, title, sort_order)
		VALUES ($1, $2, $3) RETURNING id::text
	`, board.ID, "Column 1", 1).Scan(&secondColID); err != nil {
		t.Fatalf("insert raw duplicate-titled legacy column: %v", err)
	}

	// Loading must already work fine even with NULL slugs present (the
	// COALESCE(slug, '') read path), before any backfill has run.
	preBackfillCols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("board must still load with NULL slugs present: %v", err)
	}
	for _, c := range preBackfillCols {
		if c.Slug != "" {
			t.Fatalf("expected empty slug before backfill, got %q", c.Slug)
		}
	}

	if err := EnsureKernel81AStoryboardSlugsSurface(ctx, pool); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	afterCols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list columns after backfill: %v", err)
	}
	if len(afterCols) != len(preBackfillCols) {
		t.Fatalf("backfill changed the number of columns: %d -> %d", len(preBackfillCols), len(afterCols))
	}
	seenSlugs := map[string]bool{}
	titleByID := map[string]string{}
	for _, before := range preBackfillCols {
		titleByID[before.ID] = before.Title
	}
	for _, after := range afterCols {
		if after.Slug == "" {
			t.Fatalf("column %q still has no slug after backfill", after.ID)
		}
		if seenSlugs[after.Slug] {
			t.Fatalf("backfill produced a duplicate slug %q", after.Slug)
		}
		seenSlugs[after.Slug] = true
		if after.Title != titleByID[after.ID] {
			t.Fatalf("backfill changed column %q's title: %q -> %q", after.ID, titleByID[after.ID], after.Title)
		}
	}
	// Both "Column 1" entries must have distinct slugs (one is "column-1",
	// the other collision-avoided).
	var col1Slug, col1DupSlug string
	for _, c := range afterCols {
		if c.ID == secondColID {
			col1DupSlug = c.Slug
		} else if c.Title == "Column 1" {
			col1Slug = c.Slug
		}
	}
	if col1Slug == "" || col1DupSlug == "" || col1Slug == col1DupSlug {
		t.Fatalf("expected two distinct slugs for duplicate 'Column 1' titles, got %q and %q", col1Slug, col1DupSlug)
	}

	afterBands, err := ListBands(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list bands after backfill: %v", err)
	}
	if len(afterBands) != len(bandsBefore) {
		t.Fatalf("backfill changed band count: %d -> %d", len(bandsBefore), len(afterBands))
	}
	for _, b := range afterBands {
		if b.Slug == "" {
			t.Fatalf("band %q still has no slug after backfill", b.ID)
		}
	}

	afterRows, err := ListRows(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list rows after backfill: %v", err)
	}
	for _, r := range afterRows {
		if r.Slug == "" {
			t.Fatalf("row %q still has no slug after backfill", r.ID)
		}
	}

	// Idempotent: running the backfill again must not error or change
	// anything already-populated.
	if err := EnsureKernel81AStoryboardSlugsSurface(ctx, pool); err != nil {
		t.Fatalf("second backfill run: %v", err)
	}
	againCols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list columns after second backfill: %v", err)
	}
	for i, c := range againCols {
		if c.Slug != afterCols[i].Slug {
			t.Fatalf("second backfill run changed an already-set slug: %q -> %q", afterCols[i].Slug, c.Slug)
		}
	}
}
