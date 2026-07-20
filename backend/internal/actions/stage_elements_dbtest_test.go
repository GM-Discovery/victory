package actions

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

// Kernel 72A: the former hardcoded isSingleVenueLegacySlug allowlist became
// the stage_elements_enabled venues.config flag (migration 055 seeds the same
// venues) — asserted against the real migrated test database, plus
// fail-closed behavior for venues without the flag and unknown slugs.
func TestStageElementsEnabled(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx := context.Background()

	cases := map[string]bool{
		"the-cave":            true,
		"first-theater":       true,
		"catharsis":           true,
		"middle-school-stage": false,
		"greenroom":           false,
		"unknown-venue":       false,
		"":                    false,
	}

	for slug, want := range cases {
		got, err := stageElementsEnabled(ctx, pool, slug)
		if err != nil {
			t.Fatalf("stageElementsEnabled(%q): %v", slug, err)
		}
		if got != want {
			t.Fatalf("stageElementsEnabled(%q) = %v, want %v", slug, got, want)
		}
	}
}
