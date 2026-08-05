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

	req = httptest.NewRequest("POST", "/api/venues/catharsis/map", nil)
	if got := resolveVenueMapSlug(req); got != catharsisSlug {
		t.Fatalf("resolveVenueMapSlug() = %q, want %q", got, catharsisSlug)
	}
}

func TestDefaultCourtyardMapStateIsCenteredAndStageFitted(t *testing.T) {
	state := defaultCourtyardMapState()
	if state.Asset == nil || state.Asset.ContentURL != builtInCourtyardMapURL {
		t.Fatalf("default courtyard asset = %#v, want %q", state.Asset, builtInCourtyardMapURL)
	}
	if state.Fit != "contain" || state.CropX != 0.5 || state.CropY != 0.5 || state.Scale != 1 || state.DisplayMode != "theater" {
		t.Fatalf("default courtyard placement = %#v, want centered contain/theater placement", state)
	}
}
