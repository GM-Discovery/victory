package access

import (
	"context"
	"testing"
)

// TestListRequestableVenuesReturnsAllEligibleVenuesMinusHiddenFixtures
// covers Kernel 85 §8.2: Audition Hall's venue picker must be driven by the
// real venues table (all canonical venues an authenticated user could
// plausibly ask for), not a hardcoded markup list -- and, unlike
// ResolveVisibleVenues, it must NOT be filtered down to venues this
// particular caller already has access to, since this is a *request* page.
func TestListRequestableVenuesReturnsAllEligibleVenuesMinusHiddenFixtures(t *testing.T) {
	pool := openVisibilityTestPool(t)
	// A brand-new account with no location membership and no grants at all
	// -- ResolveVisibleVenues would show this account almost nothing, but
	// ListRequestableVenues should still show the full canonical list.
	stranger := insertVisibilityTestUser(t, pool, "vis_requestable_stranger")

	venues, err := ListRequestableVenues(context.Background(), pool, stranger)
	if err != nil {
		t.Fatalf("ListRequestableVenues: %v", err)
	}
	if len(venues) == 0 {
		t.Fatal("expected at least one requestable venue")
	}

	slugs := visibleSlugSet(t, venues)

	// A handful of real, non-hidden venues that must appear regardless of
	// this caller's own access -- this is the whole point of the endpoint.
	for _, want := range []string{"catharsis", "trailers", "audition-hall"} {
		if !slugs[want] {
			t.Fatalf("expected requestable venue list to include %q, got %v", want, slugs)
		}
	}

	// Hidden/internal fixture slugs must never appear, same as the main map.
	for hiddenSlug := range hiddenMainMapVenueSlugs {
		if slugs[hiddenSlug] {
			t.Fatalf("expected hidden fixture slug %q to be excluded from requestable venues", hiddenSlug)
		}
	}

	for _, v := range venues {
		if v.VisibleBecause != "requestable" {
			t.Fatalf("expected visible_because=requestable for %q, got %q", v.Slug, v.VisibleBecause)
		}
	}
}

// TestListRequestableVenuesRequiresAuthentication mirrors
// ResolveVisibleVenues' own empty-userID contract: an anonymous caller gets
// an empty list, never an error and never the full venue list.
func TestListRequestableVenuesRequiresAuthentication(t *testing.T) {
	pool := openVisibilityTestPool(t)
	venues, err := ListRequestableVenues(context.Background(), pool, "")
	if err != nil {
		t.Fatalf("ListRequestableVenues(anonymous): %v", err)
	}
	if len(venues) != 0 {
		t.Fatalf("expected zero requestable venues for an anonymous caller, got %d", len(venues))
	}
}
