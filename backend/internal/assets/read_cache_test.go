package assets

import "testing"

// Grant, 2026-09-29: "cache the maps for cast+, audience stays uncached."
// This pins that policy as a matrix. It is a pure-function test on purpose
// -- the role lookup needs a database, but the decision it feeds must not
// be able to drift without failing here.
func TestAssetContentCacheControlFor(t *testing.T) {
	const uncached = "private, max-age=0, must-revalidate"
	const maxAge = 300

	tests := []struct {
		name      string
		assetType string
		backstage bool
		want      string
	}{
		{name: "backstage map is cached", assetType: "map", backstage: true, want: "private, max-age=300"},
		{name: "audience map is not cached", assetType: "map", backstage: false, want: uncached},

		// A token image appearing before the Director reveals it is the
		// failure this endpoint exists to prevent, so tokens never cache --
		// not even backstage, where the picker's thumbnails come from.
		{name: "backstage token is not cached", assetType: "token", backstage: true, want: uncached},
		{name: "audience token is not cached", assetType: "token", backstage: false, want: uncached},

		{name: "backstage generic asset is not cached", assetType: "generic", backstage: true, want: uncached},
		{name: "empty asset type is not cached", assetType: "", backstage: true, want: uncached},

		// Asset types are stored lowercase but nothing enforces it.
		{name: "asset type match is case-insensitive", assetType: "Map", backstage: true, want: "private, max-age=300"},
		{name: "asset type match tolerates whitespace", assetType: " map ", backstage: true, want: "private, max-age=300"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assetContentCacheControlFor(tt.assetType, tt.backstage, maxAge, uncached)
			if got != tt.want {
				t.Fatalf("assetContentCacheControlFor(%q, backstage=%v) = %q, want %q", tt.assetType, tt.backstage, got, tt.want)
			}
		})
	}
}

// A misconfigured or zero max-age must degrade to uncached rather than
// emitting "max-age=0" without must-revalidate, which is a weaker promise
// than the uncached string makes.
func TestAssetContentCacheControlForRejectsNonPositiveMaxAge(t *testing.T) {
	const uncached = "private, max-age=0, must-revalidate"
	for _, maxAge := range []int{0, -1} {
		if got := assetContentCacheControlFor("map", true, maxAge, uncached); got != uncached {
			t.Fatalf("maxAge=%d gave %q, want %q", maxAge, got, uncached)
		}
	}
}

// Every cacheable answer must stay private: this is authorized, per-viewer
// content and no shared cache may store it whatever the role.
func TestAssetContentCacheControlIsNeverPublic(t *testing.T) {
	const uncached = "private, max-age=0, must-revalidate"
	for _, assetType := range []string{"map", "token", "generic", ""} {
		for _, backstage := range []bool{true, false} {
			got := assetContentCacheControlFor(assetType, backstage, 300, uncached)
			if got[:len("private")] != "private" {
				t.Fatalf("assetContentCacheControlFor(%q, %v) = %q, which is not private", assetType, backstage, got)
			}
		}
	}
}
