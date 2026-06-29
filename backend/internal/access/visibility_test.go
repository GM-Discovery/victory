package access

import "testing"

func TestIsHiddenMainMapVenueSlug(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{name: "gateway thread venue", slug: "gateway-thread-venue", want: true},
		{name: "gateway thread fixture", slug: "gateway-thread-fixture", want: true},
		{name: "gateway thread", slug: "gateway-thread", want: true},
		{name: "normal venue", slug: "audition-hall", want: false},
		{name: "trimmed case", slug: " Gateway-Thread-Venue ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHiddenMainMapVenueSlug(tt.slug); got != tt.want {
				t.Fatalf("isHiddenMainMapVenueSlug(%q) = %t, want %t", tt.slug, got, tt.want)
			}
		})
	}
}
