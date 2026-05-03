package actions

import "testing"

func TestCanRoleRevealHide(t *testing.T) {
	tests := []struct {
		name            string
		role            string
		actorsCanReveal bool
		want            bool
	}{
		{name: "producer allowed", role: "producer", actorsCanReveal: false, want: true},
		{name: "director allowed", role: "director", actorsCanReveal: false, want: true},
		{name: "cast denied when policy off", role: "cast", actorsCanReveal: false, want: false},
		{name: "actor denied when policy off", role: "actor", actorsCanReveal: false, want: false},
		{name: "cast allowed when policy on", role: "cast", actorsCanReveal: true, want: true},
		{name: "crew denied", role: "crew", actorsCanReveal: true, want: false},
		{name: "audience denied", role: "audience", actorsCanReveal: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canRoleRevealHide(tt.role, tt.actorsCanReveal); got != tt.want {
				t.Fatalf("canRoleRevealHide(%q, %v) = %v, want %v", tt.role, tt.actorsCanReveal, got, tt.want)
			}
		})
	}
}

func TestIsRevealableCaveTarget(t *testing.T) {
	tests := []struct {
		name   string
		target ActionTarget
		want   bool
	}{
		{
			name: "first fire allowed",
			target: ActionTarget{
				Kind:        "element",
				ElementSlug: "first-fire",
			},
			want: true,
		},
		{
			name: "other element denied",
			target: ActionTarget{
				Kind:        "element",
				ElementSlug: "second-fire",
			},
			want: false,
		},
		{
			name: "empty slug denied",
			target: ActionTarget{
				Kind:        "element",
				ElementID:   "abc",
				ElementSlug: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRevealableCaveTarget(tt.target); got != tt.want {
				t.Fatalf("isRevealableCaveTarget(%+v) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}
