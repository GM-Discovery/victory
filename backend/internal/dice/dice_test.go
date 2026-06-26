package dice

import (
	"context"
	"errors"
	"testing"
)

type fakeSource struct {
	values []int
	err    error
	index  int
}

func (s *fakeSource) Intn(maxExclusive int) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	if s.index >= len(s.values) {
		return 0, errors.New("out of values")
	}
	value := s.values[s.index]
	s.index++
	if value < 0 || value >= maxExclusive {
		return 0, errors.New("value out of range")
	}
	return value, nil
}

func TestParseExpression(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantSpec  Spec
		wantExpr  string
		wantError string
	}{
		{name: "d20", raw: "d20", wantSpec: Spec{Count: 1, Sides: 20}, wantExpr: "d20"},
		{name: "two d6", raw: "2d6", wantSpec: Spec{Count: 2, Sides: 6}, wantExpr: "2d6"},
		{name: "modifier", raw: "2d6+3", wantSpec: Spec{Count: 2, Sides: 6, Modifier: 3}, wantExpr: "2d6+3"},
		{name: "negative modifier", raw: "4d8-2", wantSpec: Spec{Count: 4, Sides: 8, Modifier: -2}, wantExpr: "4d8-2"},
		{name: "explode", raw: "1d10!", wantSpec: Spec{Count: 1, Sides: 10, ExplodeOnMax: true}, wantExpr: "d10!"},
		{name: "uppercase and whitespace", raw: "  2 D 13  ", wantSpec: Spec{Count: 2, Sides: 13}, wantExpr: "2d13"},
		{name: "d100", raw: "d100", wantSpec: Spec{Count: 1, Sides: 100}, wantExpr: "d100"},
		{name: "invalid", raw: "2d6kh1", wantError: "invalid_expression"},
		{name: "too many dice", raw: "101d6", wantError: "dice_count_too_large"},
		{name: "too many sides", raw: "d1000001", wantError: "sides_too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, expr, err := ParseExpression(tt.raw)
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Fatalf("ParseExpression(%q) error = %v, want %q", tt.raw, err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseExpression(%q) unexpected error: %v", tt.raw, err)
			}
			if spec != tt.wantSpec {
				t.Fatalf("ParseExpression(%q) spec = %+v, want %+v", tt.raw, spec, tt.wantSpec)
			}
			if expr != tt.wantExpr {
				t.Fatalf("ParseExpression(%q) expr = %q, want %q", tt.raw, expr, tt.wantExpr)
			}
		})
	}
}

func TestRoll(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
		src  *fakeSource
		want Result
	}{
		{
			name: "ordinary result",
			spec: Spec{Count: 1, Sides: 20},
			src:  &fakeSource{values: []int{9}},
			want: Result{Expression: "d20", Spec: Spec{Count: 1, Sides: 20}, Dice: []DieResult{{Index: 0, Chain: []int{10}, Subtotal: 10}}, ExplosionCount: 0, Modifier: 0, Total: 10, RollVersion: 1},
		},
		{
			name: "multiple dice",
			spec: Spec{Count: 2, Sides: 6},
			src:  &fakeSource{values: []int{3, 5}},
			want: Result{Expression: "2d6", Spec: Spec{Count: 2, Sides: 6}, Dice: []DieResult{{Index: 0, Chain: []int{4}, Subtotal: 4}, {Index: 1, Chain: []int{6}, Subtotal: 6}}, ExplosionCount: 0, Modifier: 0, Total: 10, RollVersion: 1},
		},
		{
			name: "odd sided die",
			spec: Spec{Count: 1, Sides: 13},
			src:  &fakeSource{values: []int{10}},
			want: Result{Expression: "d13", Spec: Spec{Count: 1, Sides: 13}, Dice: []DieResult{{Index: 0, Chain: []int{11}, Subtotal: 11}}, ExplosionCount: 0, Modifier: 0, Total: 11, RollVersion: 1},
		},
		{
			name: "positive modifier",
			spec: Spec{Count: 1, Sides: 6, Modifier: 3},
			src:  &fakeSource{values: []int{1}},
			want: Result{Expression: "d6+3", Spec: Spec{Count: 1, Sides: 6, Modifier: 3}, Dice: []DieResult{{Index: 0, Chain: []int{2}, Subtotal: 2}}, ExplosionCount: 0, Modifier: 3, Total: 5, RollVersion: 1},
		},
		{
			name: "negative modifier",
			spec: Spec{Count: 1, Sides: 6, Modifier: -2},
			src:  &fakeSource{values: []int{4}},
			want: Result{Expression: "d6-2", Spec: Spec{Count: 1, Sides: 6, Modifier: -2}, Dice: []DieResult{{Index: 0, Chain: []int{5}, Subtotal: 5}}, ExplosionCount: 0, Modifier: -2, Total: 3, RollVersion: 1},
		},
		{
			name: "one explosion",
			spec: Spec{Count: 1, Sides: 10, ExplodeOnMax: true},
			src:  &fakeSource{values: []int{9, 9, 3}},
			want: Result{Expression: "d10!", Spec: Spec{Count: 1, Sides: 10, ExplodeOnMax: true}, Dice: []DieResult{{Index: 0, Chain: []int{10, 10, 4}, Subtotal: 24}}, ExplosionCount: 2, Modifier: 0, Total: 24, RollVersion: 1},
		},
		{
			name: "independent explosion chains",
			spec: Spec{Count: 2, Sides: 6, ExplodeOnMax: true},
			src:  &fakeSource{values: []int{5, 2, 3}},
			want: Result{Expression: "2d6!", Spec: Spec{Count: 2, Sides: 6, ExplodeOnMax: true}, Dice: []DieResult{{Index: 0, Chain: []int{6, 3}, Subtotal: 9}, {Index: 1, Chain: []int{4}, Subtotal: 4}}, ExplosionCount: 1, Modifier: 0, Total: 13, RollVersion: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Roll(context.Background(), tt.spec, tt.src)
			if err != nil {
				t.Fatalf("Roll returned error: %v", err)
			}
			if got.Expression != tt.want.Expression || got.Spec != tt.want.Spec || got.ExplosionCount != tt.want.ExplosionCount || got.Modifier != tt.want.Modifier || got.Total != tt.want.Total || got.RollVersion != tt.want.RollVersion {
				t.Fatalf("Roll summary = %+v, want %+v", got, tt.want)
			}
			if len(got.Dice) != len(tt.want.Dice) {
				t.Fatalf("Roll dice len = %d, want %d", len(got.Dice), len(tt.want.Dice))
			}
			for i := range got.Dice {
				if got.Dice[i].Index != tt.want.Dice[i].Index || got.Dice[i].Subtotal != tt.want.Dice[i].Subtotal {
					t.Fatalf("die %d mismatch: got %+v want %+v", i, got.Dice[i], tt.want.Dice[i])
				}
				if len(got.Dice[i].Chain) != len(tt.want.Dice[i].Chain) {
					t.Fatalf("die %d chain len = %d, want %d", i, len(got.Dice[i].Chain), len(tt.want.Dice[i].Chain))
				}
				for j := range got.Dice[i].Chain {
					if got.Dice[i].Chain[j] != tt.want.Dice[i].Chain[j] {
						t.Fatalf("die %d chain[%d] = %d, want %d", i, j, got.Dice[i].Chain[j], tt.want.Dice[i].Chain[j])
					}
				}
			}
		})
	}
}

func TestRollAtomicCapAndSourceError(t *testing.T) {
	values := make([]int, MaxAtomicThrows+1)
	for i := range values {
		values[i] = 1
	}
	_, err := Roll(context.Background(), Spec{Count: 1, Sides: 2, ExplodeOnMax: true}, &fakeSource{values: values})
	if err == nil || err.Error() != "atomic_roll_cap_exceeded" {
		t.Fatalf("expected atomic roll cap error, got %v", err)
	}

	_, err = Roll(context.Background(), Spec{Count: 1, Sides: 6}, &fakeSource{err: errors.New("entropy failed")})
	if err == nil || err.Error() != "entropy failed" {
		t.Fatalf("expected source error, got %v", err)
	}
}
