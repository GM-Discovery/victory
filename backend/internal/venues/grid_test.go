package venues

import (
	"net/http/httptest"
	"testing"
)

func TestResolveVenueGridSlug(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/venues/first-theater/grid", nil)
	if got := resolveVenueGridSlug(req); got != firstTheaterSlug {
		t.Fatalf("resolveVenueGridSlug() = %q, want %q", got, firstTheaterSlug)
	}

	req = httptest.NewRequest("PUT", "/api/venues/catharsis/grid", nil)
	if got := resolveVenueGridSlug(req); got != catharsisSlug {
		t.Fatalf("resolveVenueGridSlug() = %q, want %q", got, catharsisSlug)
	}
}
