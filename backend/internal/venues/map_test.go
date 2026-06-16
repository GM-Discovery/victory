package venues

import (
	"net/http/httptest"
	"testing"
)

func TestResolveVenueMapSlug(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/venues/first-theater/map", nil)
	if got := resolveVenueMapSlug(req); got != firstTheaterSlug {
		t.Fatalf("resolveVenueMapSlug() = %q, want %q", got, firstTheaterSlug)
	}
}
