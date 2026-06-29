package characters

import "testing"

func TestValidateParentageChartV11(t *testing.T) {
	if err := ValidateParentageChartV11(); err != nil {
		t.Fatalf("ValidateParentageChartV11 returned error: %v", err)
	}
}

func TestParentageChartEntryForRoll(t *testing.T) {
	tests := []struct {
		name       string
		roll       int
		wantName   string
		wantKind   string
		wantCredit int
		wantOK     bool
	}{
		{name: "lowest personal row", roll: 3, wantName: "Abandoned", wantKind: "personal", wantCredit: 0, wantOK: true},
		{name: "mid personal row", roll: 47, wantName: "Professional", wantKind: "personal", wantCredit: 150, wantOK: true},
		{name: "merchant paragon", roll: 100, wantName: "Merchant Paragon", wantKind: "organizational", wantCredit: ParentageOrganizationalCreditFloor, wantOK: true},
		{name: "organizational band", roll: 120, wantName: "Organizational Wealth", wantKind: "organizational", wantCredit: ParentageOrganizationalCreditFloor, wantOK: true},
		{name: "out of range low", roll: 2, wantOK: false},
		{name: "out of range high", roll: 121, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParentageChartEntryForRoll(tt.roll)
			if ok != tt.wantOK {
				t.Fatalf("ParentageChartEntryForRoll(%d) ok = %v, want %v", tt.roll, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.SocialClass != tt.wantName {
				t.Fatalf("ParentageChartEntryForRoll(%d).SocialClass = %q, want %q", tt.roll, got.SocialClass, tt.wantName)
			}
			if got.WealthKind != tt.wantKind {
				t.Fatalf("ParentageChartEntryForRoll(%d).WealthKind = %q, want %q", tt.roll, got.WealthKind, tt.wantKind)
			}
			if got.StartingCredit != tt.wantCredit {
				t.Fatalf("ParentageChartEntryForRoll(%d).StartingCredit = %d, want %d", tt.roll, got.StartingCredit, tt.wantCredit)
			}
		})
	}
}

func TestParentageChartV11CoversEveryRollOnce(t *testing.T) {
	seen := make(map[int]bool)
	for roll := 3; roll <= 120; roll++ {
		entry, ok := ParentageChartEntryForRoll(roll)
		if !ok {
			t.Fatalf("ParentageChartEntryForRoll(%d) returned no entry", roll)
		}
		if roll < entry.RollMin || roll > entry.RollMax {
			t.Fatalf("ParentageChartEntryForRoll(%d) returned out-of-range entry %d-%d", roll, entry.RollMin, entry.RollMax)
		}
		if seen[roll] {
			t.Fatalf("roll %d was matched more than once", roll)
		}
		seen[roll] = true
	}

	if len(seen) != 118 {
		t.Fatalf("matched %d rolls, want 118", len(seen))
	}
}
