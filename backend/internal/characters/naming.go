package characters

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var romanNumeralPattern = regexp.MustCompile(`^M{0,4}(CM|CD|D?C{0,3})(XC|XL|L?X{0,3})(IX|IV|V?I{0,3})$`)

var romanNumeralValues = []struct {
	value  int
	symbol string
}{
	{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
	{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
	{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
}

func romanNumeral(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for _, entry := range romanNumeralValues {
		for n >= entry.value {
			b.WriteString(entry.symbol)
			n -= entry.value
		}
	}
	return b.String()
}

// isGeneratedRomanName reports whether a character name looks like one of
// our auto-assigned Roman-numeral placeholder names (e.g. "I", "IV", "XII").
// This is a pattern match, not a stored flag: a player who deliberately
// renames a character to a bare Roman numeral will also match.
func isGeneratedRomanName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && romanNumeralPattern.MatchString(name)
}

func parseRomanNumeral(name string) (int, bool) {
	if !isGeneratedRomanName(name) {
		return 0, false
	}
	remaining := name
	total := 0
	for _, entry := range romanNumeralValues {
		for strings.HasPrefix(remaining, entry.symbol) {
			total += entry.value
			remaining = remaining[len(entry.symbol):]
		}
	}
	return total, true
}

// nextUnusedRomanNumeralName returns the lowest positive integer (expressed
// as a Roman numeral) not already used as a name among the owner's
// non-deleted character cards. Renaming a character frees its number for
// reuse by the next default-named character.
func nextUnusedRomanNumeralName(ctx context.Context, pool *pgxpool.Pool, ownerUserID string) (string, error) {
	rows, err := pool.Query(ctx, `
		SELECT name FROM character_cards WHERE owner_user_id = $1 AND is_deleted = FALSE
	`, ownerUserID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	used := map[int]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if n, ok := parseRomanNumeral(strings.TrimSpace(name)); ok {
			used[n] = true
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	candidate := 1
	for used[candidate] {
		candidate++
	}
	return romanNumeral(candidate), nil
}
