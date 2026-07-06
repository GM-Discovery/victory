package actions

import (
	"context"
	"testing"
)

func TestSkillRolledThisSessionShortCircuitsOnEmptyArgs(t *testing.T) {
	// nil pool is safe here: empty session/actor/skill IDs short-circuit
	// before any database access.
	cases := [][3]string{
		{"", "user-1", "SKILL_X"},
		{"session-1", "", "SKILL_X"},
		{"session-1", "user-1", ""},
	}
	for _, c := range cases {
		ok, err := SkillRolledThisSession(context.Background(), nil, c[0], c[1], c[2])
		if err != nil {
			t.Fatalf("unexpected error for %v: %v", c, err)
		}
		if ok {
			t.Fatalf("expected false for %v", c)
		}
	}
}
