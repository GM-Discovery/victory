package actions

import (
	"context"
	"strconv"
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
		case *[]byte:
			switch v := r.values[i].(type) {
			case []byte:
				*d = append((*d)[:0], v...)
			case string:
				*d = []byte(v)
			}
		}
	}
	return nil
}

type fakeQuerier struct {
	role          string
	handle        string
	venueSlug     string
	venueEnabled  bool
	showingStatus string
	layoutFound   bool
	locked        bool
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
	case strings.Contains(sql, "FROM users") && strings.Contains(sql, "lower(COALESCE(NULLIF(handle, ''), ''))"):
		return fakeRow{values: []any{strings.ToLower(strings.TrimSpace(q.handle))}}
	case strings.Contains(sql, "JOIN venue_layout_elements"):
		if !q.layoutFound {
			return fakeRow{err: pgx.ErrNoRows}
		}
		visibility := `{"locked":` + strings.ToLower(strconv.FormatBool(q.locked)) + `,"nameplate_visible":true,"visible":true}`
		return fakeRow{values: []any{"venue-1", q.venueSlug, "card-1", "index-card-director", "index_card", "card", "stage", []byte(visibility)}}
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

func TestCanActDuplicateElement(t *testing.T) {
	tests := []struct {
		name         string
		role         string
		locked       bool
		venueEnabled bool
		want         bool
		wantReason   string
	}{
		{name: "producer allowed", role: "producer", locked: false, venueEnabled: true, want: true, wantReason: "allowed"},
		{name: "director allowed", role: "director", locked: false, venueEnabled: true, want: true, wantReason: "allowed"},
		{name: "cast denied", role: "cast", locked: false, venueEnabled: true, want: false, wantReason: "insufficient_role"},
		{name: "locked denied", role: "producer", locked: true, venueEnabled: true, want: false, wantReason: "locked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := fakeQuerier{role: tt.role, venueSlug: "the-cave", venueEnabled: tt.venueEnabled, layoutFound: true, locked: tt.locked}
			decision, err := CanAct(context.Background(), q, "user-1", "act/duplicate_element", "session-1", ActionTarget{
				Kind:      "element",
				ElementID: "card-1",
				VenueSlug: "the-cave",
				Layer:     "stage",
			})
			if err != nil {
				t.Fatalf("CanAct act/duplicate_element returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct act/duplicate_element allowed=%v, want %v", decision.Allowed, tt.want)
			}
			if decision.Reason != tt.wantReason {
				t.Fatalf("CanAct act/duplicate_element reason=%q, want %q", decision.Reason, tt.wantReason)
			}
		})
	}
}

func TestCanActDiceRoll(t *testing.T) {
	t.Setenv("OPERATOR_HANDLE", "straturli")

	tests := []struct {
		name string
		q    fakeQuerier
		want bool
	}{
		{name: "producer allowed", q: fakeQuerier{role: "producer"}, want: true},
		{name: "director allowed", q: fakeQuerier{role: "director"}, want: true},
		{name: "audience denied", q: fakeQuerier{role: "audience"}, want: false},
		{name: "operator allowed", q: fakeQuerier{role: "audience", handle: "straturli"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := CanAct(context.Background(), tt.q, "user-1", "roll/dice", "session-1", ActionTarget{Kind: "session"})
			if err != nil {
				t.Fatalf("CanAct roll/dice returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct roll/dice allowed=%v, want %v", decision.Allowed, tt.want)
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

func TestCanActLockedElementBlocksManipulation(t *testing.T) {
	q := fakeQuerier{role: "producer", venueSlug: "the-cave", venueEnabled: true, layoutFound: true, locked: true}

	decision, err := CanAct(context.Background(), q, "user-1", "act/place_element", "session-1", ActionTarget{
		Kind:      "element",
		ElementID: "card-1",
		VenueSlug: "the-cave",
		Layer:     "stage",
	})
	if err != nil {
		t.Fatalf("CanAct act/place_element returned error: %v", err)
	}
	if decision.Allowed || decision.Reason != "locked" {
		t.Fatalf("expected locked place_element denial, got %+v", decision)
	}

	decision, err = CanAct(context.Background(), q, "user-1", "act/set_nameplate_visibility", "session-1", ActionTarget{
		Kind:      "element",
		ElementID: "card-1",
	})
	if err != nil {
		t.Fatalf("CanAct act/set_nameplate_visibility returned error: %v", err)
	}
	if decision.Allowed || decision.Reason != "locked" {
		t.Fatalf("expected locked nameplate denial, got %+v", decision)
	}
}
