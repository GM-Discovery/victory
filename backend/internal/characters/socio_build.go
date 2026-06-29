package characters

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

func seedCatharsisStarterDraft(input CharacterCardInput) CharacterCardInput {
	context := normalizeWorkbookContext(input.WorkbookContext)
	if strings.ToLower(strings.TrimSpace(stringValue(context["source"]))) != "catharsis" {
		return input
	}

	context = resolveCatharsisParentageContext(context, randomCatharsisD2)
	input.WorkbookContext = context

	entry := catharsisPrimaryParentageEntry(context)
	parentageRoll, _ := parseCatharsisRoll(context["socio_parentage_roll"])
	if strings.TrimSpace(input.Tagline) == "" {
		input.Tagline = fmt.Sprintf("Parentage roll %d · %s", parentageRoll, entry.SocialClass)
	}
	if strings.TrimSpace(input.PublicDescription) == "" {
		input.PublicDescription = entry.Description
	}
	if strings.TrimSpace(input.PrivateNotes) == "" {
		input.PrivateNotes = fmt.Sprintf(
			"Generated from Catharsis Socio %s. Wealth kind: %s. Starting credit: %d.",
			ParentageChartVersionV11,
			entry.WealthKind,
			entry.StartingCredit,
		)
	}
	if startingWealth, ok := parseCatharsisRoll(context["socio_starting_wealth"]); ok && startingWealth > 0 {
		sourceClass := strings.TrimSpace(stringValue(context["socio_starting_wealth_source_class"]))
		if sourceClass != "" {
			input.PrivateNotes = strings.TrimSpace(input.PrivateNotes + fmt.Sprintf(" Starting wealth resolved to %d from %s.", startingWealth, sourceClass))
		} else {
			input.PrivateNotes = strings.TrimSpace(input.PrivateNotes + fmt.Sprintf(" Starting wealth resolved to %d.", startingWealth))
		}
	}
	if strings.TrimSpace(input.Color) == "" {
		input.Color = "#d9c7a6"
	}

	return input
}

func resolveCatharsisParentageContext(context map[string]any, rollD2 func() int) map[string]any {
	out := normalizeWorkbookContext(context)
	rows := normalizeCatharsisParentageRows(out["socio_parentage_parents"])
	if len(rows) == 0 {
		roll := resolveCatharsisParentageRoll(out)
		entry, ok := ParentageChartEntryForRoll(roll)
		if !ok {
			roll = 3
			entry, _ = ParentageChartEntryForRoll(roll)
		}
		rows = []map[string]any{{
			"parent_index":    1,
			"roll_total":      roll,
			"social_class":    entry.SocialClass,
			"wealth_kind":     entry.WealthKind,
			"starting_credit": entry.StartingCredit,
			"description":     entry.Description,
		}}
	}

	enrichedRows, coinFlips, eligibleParents, startingWealth, inheritedParent := enrichCatharsisParentageRows(rows, rollD2)
	primaryEntry := catharsisPrimaryParentageEntry(out)
	out["socio_parentage_chart_version"] = ParentageChartVersionV11
	out["socio_parentage_parents"] = enrichedRows
	out["socio_parentage_coin_flips"] = coinFlips
	out["socio_wealth_eligible_parents"] = eligibleParents
	out["socio_starting_wealth"] = startingWealth
	out["socio_starting_wealth_source_parent_index"] = inheritedParent["parent_index"]
	out["socio_starting_wealth_source_roll"] = inheritedParent["roll_total"]
	out["socio_starting_wealth_source_class"] = inheritedParent["social_class"]
	out["socio_parentage_class"] = primaryEntry.SocialClass
	out["socio_parentage_starting_credit"] = primaryEntry.StartingCredit
	out["socio_parentage_description"] = primaryEntry.Description
	if roll, ok := parseCatharsisRoll(out["socio_parentage_roll"]); ok {
		out["socio_parentage_roll"] = roll
	} else {
		out["socio_parentage_roll"] = totalParentageRoll(enrichedRows)
	}
	if out["socio_parentage_first_roll"] == nil && len(enrichedRows) > 0 {
		out["socio_parentage_first_roll"] = enrichedRows[0]["roll_total"]
	}
	if out["socio_parentage_second_roll"] == nil && len(enrichedRows) > 1 {
		out["socio_parentage_second_roll"] = enrichedRows[1]["roll_total"]
	}
	out["socio_parentage_total_roll"] = totalParentageRoll(enrichedRows)
	return out
}

func catharsisPrimaryParentageEntry(context map[string]any) ParentageChartEntry {
	roll, ok := parseCatharsisRoll(context["socio_parentage_roll"])
	if !ok {
		if rows := normalizeCatharsisParentageRows(context["socio_parentage_parents"]); len(rows) > 0 {
			roll, _ = parseCatharsisRoll(rows[0]["roll_total"])
		}
	}
	entry, ok := ParentageChartEntryForRoll(roll)
	if !ok {
		entry, _ = ParentageChartEntryForRoll(3)
	}
	return entry
}

func normalizeCatharsisParentageRows(value any) []map[string]any {
	entries := anySlice(value)
	if len(entries) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(entries))
	for idx, raw := range entries {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		normalized := normalizeWorkbookContext(row)
		if normalized == nil {
			normalized = map[string]any{}
		}
		if _, ok := parseCatharsisRoll(normalized["parent_index"]); !ok {
			normalized["parent_index"] = idx + 1
		}
		out = append(out, normalized)
	}
	return out
}

func enrichCatharsisParentageRows(rows []map[string]any, rollD2 func() int) ([]map[string]any, []map[string]any, []any, int, map[string]any) {
	enrichedRows := make([]map[string]any, 0, len(rows))
	coinFlips := make([]map[string]any, 0, len(rows))
	eligibleParents := make([]any, 0, len(rows))
	startingWealth := 0
	inheritedParent := map[string]any{}

	for _, row := range rows {
		normalized := normalizeWorkbookContext(row)
		if normalized == nil {
			normalized = map[string]any{}
		}

		parentIndex, _ := parseCatharsisRoll(normalized["parent_index"])
		if parentIndex == 0 {
			parentIndex = len(enrichedRows) + 1
		}
		roll, _ := parseCatharsisRoll(normalized["roll_total"])
		entry, ok := ParentageChartEntryForRoll(roll)
		if !ok {
			entry, _ = ParentageChartEntryForRoll(3)
		}

		startCredit := entry.StartingCredit
		if explicitCredit, ok := parseCatharsisRoll(normalized["starting_credit"]); ok {
			startCredit = explicitCredit
		}
		coinFlipRoll := 0
		coinFlipResult := "not eligible"
		inheritancePassed := false
		inheritedWealth := 0
		if startCredit > 50 {
			coinFlipRoll = 1
			if rollD2 != nil {
				coinFlipRoll = rollD2()
			}
			if coinFlipRoll != 1 {
				coinFlipRoll = 2
			}
			if coinFlipRoll == 2 {
				coinFlipResult = "retain"
				inheritancePassed = true
				inheritedWealth = startCredit
				if inheritedWealth > startingWealth {
					startingWealth = inheritedWealth
					inheritedParent = map[string]any{
						"parent_index":    parentIndex,
						"roll_total":      roll,
						"social_class":    entry.SocialClass,
						"starting_credit": startCredit,
					}
				}
			} else {
				coinFlipResult = "lose"
			}
			eligibleParents = append(eligibleParents, parentIndex)
		}

		normalized["parent_index"] = parentIndex
		normalized["roll_total"] = roll
		normalized["roll_range"] = formatParentageRollRange(entry)
		normalized["social_class"] = entry.SocialClass
		normalized["wealth_kind"] = entry.WealthKind
		normalized["starting_credit"] = startCredit
		normalized["description"] = entry.Description
		normalized["coin_flip_roll"] = coinFlipRoll
		normalized["coin_flip_result"] = coinFlipResult
		normalized["coin_flip_eligible"] = startCredit > 50
		normalized["inheritance_passed"] = inheritancePassed
		normalized["inherited_wealth"] = inheritedWealth

		enrichedRows = append(enrichedRows, normalized)
		if startCredit > 50 {
			coinFlips = append(coinFlips, map[string]any{
				"parent_index":       parentIndex,
				"roll_total":         roll,
				"coin_flip_roll":     coinFlipRoll,
				"coin_flip_result":   coinFlipResult,
				"inheritance_passed": inheritancePassed,
				"inherited_wealth":   inheritedWealth,
			})
		}
	}

	return enrichedRows, coinFlips, eligibleParents, startingWealth, inheritedParent
}

func totalParentageRoll(rows []map[string]any) int {
	total := 0
	for _, row := range rows {
		if roll, ok := parseCatharsisRoll(row["roll_total"]); ok {
			total += roll
		}
	}
	return total
}

func formatParentageRollRange(entry ParentageChartEntry) string {
	if entry.RollMin == entry.RollMax {
		return strconv.Itoa(entry.RollMin)
	}
	return fmt.Sprintf("%d-%d", entry.RollMin, entry.RollMax)
}

func resolveCatharsisParentageRoll(context map[string]any) int {
	if roll, ok := parseCatharsisRoll(context["socio_parentage_roll"]); ok {
		return roll
	}
	return randomCatharsisRoll()
}

func parseCatharsisRoll(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return clampCatharsisRoll(v)
	case int32:
		return clampCatharsisRoll(int(v))
	case int64:
		return clampCatharsisRoll(int(v))
	case float64:
		return clampCatharsisRoll(int(v))
	case float32:
		return clampCatharsisRoll(int(v))
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed); err == nil {
			return clampCatharsisRoll(parsed)
		}
	}
	return 0, false
}

func clampCatharsisRoll(value int) (int, bool) {
	if value < 3 || value > 120 {
		return 0, false
	}
	return value, true
}

func randomCatharsisRoll() int {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 3
	}
	n := binary.LittleEndian.Uint64(buf[:])
	return 3 + int(n%118)
}

func randomCatharsisD2() int {
	var buf [1]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 1
	}
	return 1 + int(buf[0]%2)
}
