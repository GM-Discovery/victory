package characters

import "testing"

func TestSeedCatharsisStarterDraftUsesRequestedRoll(t *testing.T) {
	input := CharacterCardInput{
		WorkbookContext: map[string]any{
			"source":               "catharsis",
			"socio_parentage_roll": 100,
		},
	}

	got := seedCatharsisStarterDraft(input)
	if got.Name != "" {
		t.Fatalf("Name = %q, want untouched (Roman-numeral default is assigned by CreateCard, not chart seeding)", got.Name)
	}
	if got.Tagline == "" || got.Tagline == input.Tagline {
		t.Fatalf("expected tagline to be seeded, got %q", got.Tagline)
	}
	entry, ok := ParentageChartEntryForRoll(100)
	if !ok {
		t.Fatalf("expected chart entry for roll 100")
	}
	if got.PublicDescription != entry.Description {
		t.Fatalf("expected public description to be seeded, got %q", got.PublicDescription)
	}
	if got.WorkbookContext["socio_parentage_roll"] != 100 {
		t.Fatalf("socio_parentage_roll = %v, want 100", got.WorkbookContext["socio_parentage_roll"])
	}
	if got.WorkbookContext["socio_parentage_class"] != "Merchant Paragon" {
		t.Fatalf("socio_parentage_class = %v, want Merchant Paragon", got.WorkbookContext["socio_parentage_class"])
	}

	rows := normalizeCatharsisParentageRows(got.WorkbookContext["socio_parentage_parents"])
	if len(rows) != 1 {
		t.Fatalf("expected one parent row, got %d", len(rows))
	}
	if rows[0]["coin_flip_result"] != "pending" {
		t.Fatalf("coin_flip_result = %v, want pending (coin flip must not auto-resolve at character creation)", rows[0]["coin_flip_result"])
	}
	if got.WorkbookContext["socio_starting_wealth"] != 0 {
		t.Fatalf("socio_starting_wealth = %v, want 0 (unresolved until the player flips)", got.WorkbookContext["socio_starting_wealth"])
	}
}

func TestSeedCatharsisStarterDraftLeavesNonCatharsisInputAlone(t *testing.T) {
	input := CharacterCardInput{Name: "", WorkbookContext: map[string]any{"source": "other"}}
	got := seedCatharsisStarterDraft(input)
	if got.Name != "" {
		t.Fatalf("expected non-Catharsis input to be unchanged, got %q", got.Name)
	}
}

func TestResolveCatharsisParentageContextRetentionBoundaries(t *testing.T) {
	tests := []struct {
		name           string
		startingCredit int
		d2Result       int
		wantEligible   bool
		wantRoll       int
		wantWealth     int
		wantCalls      int
	}{
		{name: "forty-nine no flip", startingCredit: 49, d2Result: 1, wantEligible: false, wantRoll: 0, wantWealth: 0, wantCalls: 0},
		{name: "fifty no flip", startingCredit: 50, d2Result: 2, wantEligible: false, wantRoll: 0, wantWealth: 0, wantCalls: 0},
		{name: "fifty-one lose", startingCredit: 51, d2Result: 1, wantEligible: true, wantRoll: 1, wantWealth: 0, wantCalls: 1},
		{name: "fifty-one retain", startingCredit: 51, d2Result: 2, wantEligible: true, wantRoll: 2, wantWealth: 51, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			got := resolveCatharsisParentageContext(map[string]any{
				"source": "catharsis",
				"socio_parentage_parents": []any{
					map[string]any{
						"parent_index":    1,
						"roll_total":      3,
						"starting_credit": tt.startingCredit,
					},
				},
			}, func() int {
				calls++
				return tt.d2Result
			})

			rows := normalizeCatharsisParentageRows(got["socio_parentage_parents"])
			if len(rows) != 1 {
				t.Fatalf("expected one resolved parent row, got %d", len(rows))
			}
			row := rows[0]

			toInt := func(value any) int {
				switch v := value.(type) {
				case int:
					return v
				case int32:
					return int(v)
				case int64:
					return int(v)
				case float64:
					return int(v)
				case float32:
					return int(v)
				default:
					return 0
				}
			}

			if gotEligible := row["coin_flip_eligible"] == true; gotEligible != tt.wantEligible {
				t.Fatalf("coin_flip_eligible = %v, want %v", gotEligible, tt.wantEligible)
			}
			if gotRoll := toInt(row["coin_flip_roll"]); gotRoll != tt.wantRoll {
				t.Fatalf("coin_flip_roll = %d, want %d", gotRoll, tt.wantRoll)
			}
			if gotWealth, _ := parseCatharsisInt(row["inherited_wealth"]); gotWealth != tt.wantWealth {
				t.Fatalf("inherited_wealth = %d, want %d", gotWealth, tt.wantWealth)
			}
			if gotWealth, _ := parseCatharsisInt(got["socio_starting_wealth"]); gotWealth != tt.wantWealth {
				t.Fatalf("socio_starting_wealth = %d, want %d", gotWealth, tt.wantWealth)
			}
			if gotCalls := calls; gotCalls != tt.wantCalls {
				t.Fatalf("d2 calls = %d, want %d", gotCalls, tt.wantCalls)
			}
		})
	}
}

func TestResolveCatharsisParentageContextUsesHigherParentRoll(t *testing.T) {
	got := resolveCatharsisParentageContext(map[string]any{
		"source": "catharsis",
		"socio_parentage_parents": []any{
			map[string]any{
				"parent_index":    1,
				"roll_total":      52,
				"starting_credit": 220,
			},
			map[string]any{
				"parent_index":    2,
				"roll_total":      31,
				"starting_credit": 45,
			},
		},
	}, nil)

	if roll, ok := parseCatharsisRoll(got["socio_parentage_roll"]); !ok || roll != 52 {
		t.Fatalf("socio_parentage_roll = %v, want 52", got["socio_parentage_roll"])
	}
	if roll, ok := parseCatharsisRoll(got["socio_parentage_total_roll"]); !ok || roll != 83 {
		t.Fatalf("socio_parentage_total_roll = %v, want 83", got["socio_parentage_total_roll"])
	}
}
