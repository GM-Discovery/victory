package shows

import (
	"context"
	"testing"
)

// TestCreateShowAssignsUniqueShortCode proves every new Show gets a code
// automatically (spec §1.7/§8.1), with no client input, unique within its
// location, and normalized to the confusable-excluding charset.
func TestCreateShowAssignsUniqueShortCode(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sc_producer")
	_, showRunID, showID := insertShowFixture(t, pool, producer)

	s, err := LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	if len(s.ShortCode) != shortCodeLength {
		t.Fatalf("expected a %d-character short code, got %q", shortCodeLength, s.ShortCode)
	}
	for _, r := range s.ShortCode {
		if !contains(shortCodeAlphabet, r) {
			t.Fatalf("short code %q contains a character outside the confusable-excluding alphabet", s.ShortCode)
		}
	}

	second, err := CreateShow(context.Background(), pool, producer, showRunID, CreateShowInput{
		Title: "Second Show", Slug: "second-show-" + testSuffix(t),
	})
	if err != nil {
		t.Fatalf("create second show: %v", err)
	}
	if second.ShortCode == s.ShortCode {
		t.Fatalf("expected the second show's code to differ from the first, both got %q", s.ShortCode)
	}
}

func contains(alphabet string, r rune) bool {
	for _, a := range alphabet {
		if a == r {
			return true
		}
	}
	return false
}

// TestUpdateShortCodeValidatesNormalizesAndRejectsConflicts covers spec
// §8.1: case-insensitive, safe charset, length bounds, clear conflict
// response, and authority-gated.
func TestUpdateShortCodeValidatesNormalizesAndRejectsConflicts(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sc_edit_producer")
	stranger := insertShowsTestUser(t, pool, "sc_edit_stranger")
	_, showRunID, showID := insertShowFixture(t, pool, producer)

	second, err := CreateShow(context.Background(), pool, producer, showRunID, CreateShowInput{
		Title: "Second Show", Slug: "second-show-" + testSuffix(t),
	})
	if err != nil {
		t.Fatalf("create second show: %v", err)
	}

	updated, err := UpdateShortCode(context.Background(), pool, producer, showID, "ra01")
	if err != nil {
		t.Fatalf("update short code: %v", err)
	}
	if updated.ShortCode != "RA01" {
		t.Fatalf("expected case-normalized RA01, got %q", updated.ShortCode)
	}

	// Re-setting the same code (any case) is not a conflict with itself.
	if _, err := UpdateShortCode(context.Background(), pool, producer, showID, "RA01"); err != nil {
		t.Fatalf("expected re-setting the same code to succeed, got %v", err)
	}

	if _, err := UpdateShortCode(context.Background(), pool, producer, second.ID, "ra01"); err == nil || err.Error() != "short_code_taken" {
		t.Fatalf("expected short_code_taken for a colliding code, got %v", err)
	}

	if _, err := UpdateShortCode(context.Background(), pool, producer, showID, "a"); err == nil || err.Error() != "short_code_invalid_length" {
		t.Fatalf("expected short_code_invalid_length for a too-short code, got %v", err)
	}

	if _, err := UpdateShortCode(context.Background(), pool, producer, showID, "ra-01"); err == nil || err.Error() != "short_code_invalid_characters" {
		t.Fatalf("expected short_code_invalid_characters for punctuation, got %v", err)
	}

	if _, err := UpdateShortCode(context.Background(), pool, stranger, showID, "zz99"); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for a non-manager, got %v", err)
	}
}

// TestLoadShowByCodeResolvesCaseInsensitivelyAndRejectsUnknown covers
// lookup used by both /showtime and Audition Hall's Join-a-Show flow, and
// the "no backstage leak" HTTP-layer requirement is proven separately at
// the handler/response-shape level.
func TestLoadShowByCodeResolvesCaseInsensitivelyAndRejectsUnknown(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sc_lookup_producer")
	_, _, showID := insertShowFixture(t, pool, producer)

	if _, err := UpdateShortCode(context.Background(), pool, producer, showID, "kessa"); err != nil {
		t.Fatalf("set short code: %v", err)
	}

	found, err := LoadShowByCode(context.Background(), pool, producer, "KeSsA")
	if err != nil {
		t.Fatalf("load by code (mixed case): %v", err)
	}
	if found.ID != showID {
		t.Fatalf("expected to resolve show %q, got %q", showID, found.ID)
	}

	if _, err := LoadShowByCode(context.Background(), pool, producer, "nonexistent"); err == nil || err.Error() != "show_code_not_found" {
		t.Fatalf("expected show_code_not_found for an unknown code, got %v", err)
	}
}
