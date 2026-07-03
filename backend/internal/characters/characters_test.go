package characters

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
		case *int:
			if v, ok := r.values[i].(int); ok {
				*d = v
			}
		case *string:
			if v, ok := r.values[i].(string); ok {
				*d = v
			}
		}
	}
	return nil
}

type draftQuerier struct {
	implicit bool
	count    int
	hasCount bool
	err      error
}

func (q draftQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	_ = ctx
	_ = args
	if q.err != nil {
		return fakeRow{err: q.err}
	}
	switch {
	case strings.Contains(sql, "COUNT(*)::int") && strings.Contains(sql, "FROM character_cards"):
		return fakeRow{values: []any{q.count}}
	case strings.Contains(sql, "FROM (") && strings.Contains(sql, "location_memberships") && strings.Contains(sql, "producer"):
		if q.implicit {
			return fakeRow{values: []any{1}}
		}
		return fakeRow{err: pgx.ErrNoRows}
	default:
		return fakeRow{err: pgx.ErrNoRows}
	}
}

func TestCanDraftCharacterRoleInheritanceAndGrant(t *testing.T) {
	tests := []struct {
		name    string
		querier draftQuerier
		want    bool
	}{
		{name: "performer role implicit", querier: draftQuerier{implicit: true}, want: true},
		{name: "user without performer role denied", querier: draftQuerier{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanDraftCharacter(context.Background(), tt.querier, "user-1")
			if err != nil {
				t.Fatalf("CanDraftCharacter returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("CanDraftCharacter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanDraftCharacterRequiresUser(t *testing.T) {
	got, err := CanDraftCharacter(context.Background(), draftQuerier{implicit: true}, "")
	if err != nil {
		t.Fatalf("CanDraftCharacter returned error: %v", err)
	}
	if got {
		t.Fatalf("expected anonymous user to be denied")
	}
}

func TestSanitizeInputSheetLinks(t *testing.T) {
	links := []SheetLink{
		{
			ID:        "sheet-1",
			Ruleset:   "  story-first  ",
			SheetType: " blank ",
			Label:     "  Story Sheet  ",
			URL:       "  ",
			CreatedAt: "not-a-date",
		},
		{},
	}

	input := sanitizeInput(CharacterCardInput{
		Name:       "Hero",
		SheetLinks: &links,
	})
	if input.SheetLinks == nil {
		t.Fatalf("expected sheet links to remain present")
	}

	got := *input.SheetLinks
	if len(got) != 1 {
		t.Fatalf("got %d sheet links, want 1", len(got))
	}
	if got[0].Ruleset != "story-first" {
		t.Fatalf("ruleset = %q, want story-first", got[0].Ruleset)
	}
	if got[0].SheetType != "blank" {
		t.Fatalf("sheet_type = %q, want blank", got[0].SheetType)
	}
	if got[0].Label != "Story Sheet" {
		t.Fatalf("label = %q, want Story Sheet", got[0].Label)
	}
	if got[0].CreatedAt == "" || got[0].CreatedAt == "not-a-date" {
		t.Fatalf("expected invalid created_at to be replaced, got %q", got[0].CreatedAt)
	}
}

func TestCanCreateCharacterCardEnforcesLimit(t *testing.T) {
	previous := maxCharacterCardsPerAccount
	maxCharacterCardsPerAccount = 2
	defer func() { maxCharacterCardsPerAccount = previous }()

	ok, err := canCreateCharacterCard(context.Background(), draftQuerier{count: 1}, "user-1")
	if err != nil {
		t.Fatalf("canCreateCharacterCard returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected count below limit to be allowed")
	}

	ok, err = canCreateCharacterCard(context.Background(), draftQuerier{count: 2}, "user-1")
	if err != nil {
		t.Fatalf("canCreateCharacterCard returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected count at limit to be denied")
	}
}
