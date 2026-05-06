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
	grant    bool
	err      error
}

func (q draftQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	_ = ctx
	_ = args
	if q.err != nil {
		return fakeRow{err: q.err}
	}
	switch {
	case strings.Contains(sql, "FROM (") && strings.Contains(sql, "location_memberships") && strings.Contains(sql, "producer"):
		if q.implicit {
			return fakeRow{values: []any{1}}
		}
		return fakeRow{err: pgx.ErrNoRows}
	case strings.Contains(sql, "FROM permission_grants"):
		if q.grant {
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
		{name: "producer director implicit", querier: draftQuerier{implicit: true}, want: true},
		{name: "cast with grant", querier: draftQuerier{grant: true}, want: true},
		{name: "cast without grant denied", querier: draftQuerier{}, want: false},
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
