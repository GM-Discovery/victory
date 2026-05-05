package actions

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		if i >= len(r.values) {
			break
		}
		switch d := dest[i].(type) {
		case *string:
			if v, ok := r.values[i].(string); ok {
				*d = v
				continue
			}
		case *bool:
			if v, ok := r.values[i].(bool); ok {
				*d = v
				continue
			}
		}
	}
	return nil
}

type fakeQuerier struct {
	role          string
	venueSlug     string
	venueEnabled  bool
	showingStatus string
}

func (q fakeQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	_ = ctx
	_ = args
	switch {
	case strings.Contains(sql, "FROM showings"):
		status := q.showingStatus
		if status == "" {
			status = "live"
		}
		return fakeRow{values: []any{status}}
	case strings.Contains(sql, "COALESCE((v.config ->> 'index_cards_enabled')::boolean, FALSE)"):
		return fakeRow{values: []any{q.venueSlug, q.venueEnabled}}
	case strings.Contains(sql, "FROM session_participants sp") && strings.Contains(sql, "SELECT sp.role::text"):
		return fakeRow{values: []any{q.role}}
	case strings.Contains(sql, "SELECT EXISTS"):
		return fakeRow{values: []any{true}}
	default:
		return fakeRow{values: []any{"the-cave", q.role, true, "venue-1", "user-1"}}
	}
}

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
		{name: "element slug allowed", target: ActionTarget{Kind: "element", ElementSlug: "first-fire"}, want: true},
		{name: "other element allowed", target: ActionTarget{Kind: "element", ElementSlug: "second-fire"}, want: true},
		{name: "element id allowed", target: ActionTarget{Kind: "element", ElementID: "abc"}, want: true},
		{name: "empty denied", target: ActionTarget{Kind: "element"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRevealableCaveTarget(tt.target); got != tt.want {
				t.Fatalf("isRevealableCaveTarget(%+v) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestCanActIndexCardCreateUpdate(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{name: "producer allowed", role: "producer", want: true},
		{name: "director allowed", role: "director", want: true},
		{name: "cast denied", role: "cast", want: false},
		{name: "crew denied", role: "crew", want: false},
		{name: "audience denied", role: "audience", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := CanAct(context.Background(), fakeQuerier{role: tt.role}, "user-1", "create/index_card", "session-1", ActionTarget{Kind: "index_card"})
			if err != nil {
				t.Fatalf("CanAct create/index_card returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct create/index_card allowed=%v, want %v", decision.Allowed, tt.want)
			}

			decision, err = CanAct(context.Background(), fakeQuerier{role: tt.role}, "user-1", "update/index_card", "session-1", ActionTarget{Kind: "index_card", ElementID: "card-1"})
			if err != nil {
				t.Fatalf("CanAct update/index_card returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct update/index_card allowed=%v, want %v", decision.Allowed, tt.want)
			}
		})
	}
}

func TestCanActPlaceElement(t *testing.T) {
	tests := []struct {
		name         string
		role         string
		venueEnabled bool
		want         bool
	}{
		{name: "producer allowed", role: "producer", venueEnabled: true, want: true},
		{name: "director allowed", role: "director", venueEnabled: true, want: true},
		{name: "cast denied", role: "cast", venueEnabled: true, want: false},
		{name: "crew denied", role: "crew", venueEnabled: true, want: false},
		{name: "audience denied", role: "audience", venueEnabled: true, want: false},
		{name: "venue disabled", role: "producer", venueEnabled: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := CanAct(context.Background(), fakeQuerier{role: tt.role, venueSlug: "the-cave", venueEnabled: tt.venueEnabled}, "user-1", "act/place_element", "session-1", ActionTarget{
				Kind:      "index_card",
				ElementID: "card-1",
				VenueSlug: "the-cave",
				Layer:     "tray",
			})
			if err != nil {
				t.Fatalf("CanAct act/place_element returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct act/place_element allowed=%v, want %v", decision.Allowed, tt.want)
			}
		})
	}
}

func TestCanActBlockedWhenShowingClosed(t *testing.T) {
	decision, err := CanAct(context.Background(), fakeQuerier{role: "producer", venueSlug: "the-cave", venueEnabled: true, showingStatus: "closed"}, "user-1", "chat/message", "session-1", ActionTarget{Kind: "session"})
	if err != nil {
		t.Fatalf("CanAct chat/message returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected chat/message to be blocked when showing is closed")
	}
}
