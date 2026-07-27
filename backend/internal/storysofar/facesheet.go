package storysofar

import (
	"sort"
	"strings"
	"unicode"
)

// Lens is the Character's primary strength for the reflection, plus the
// Face Sheet line that already showed it, if one exists.
type Lens struct {
	// Attribute is the chosen primary lens. Empty only when the Character
	// has no recorded attributes at all.
	Attribute string
	// Score is the recorded value of Attribute.
	Score int
	// FromArchetype reports whether the archetype mapping chose Attribute
	// (S1.7's first rule) rather than the highest-score fallback.
	FromArchetype bool
	// Line is the matching Face Sheet history line. Valid only if HasLine.
	Line    StageLine
	HasLine bool
}

// SelectPrimaryLens implements S1.7's ordering:
//
//	Character archetype -> mapped primary attribute -> matching life-stage
//	or Face Sheet history line
//
// with the two documented fallbacks: if the archetype mapping is absent or
// invalid, use the highest attribute; if several attributes tie, prefer the
// attribute connected to the Character's archetype.
//
// If no Face Sheet line matches, HasLine is false and the caller omits the
// quotation entirely (S1.7: "omit that quotation rather than fabricating
// one"). There is no fallback text and there must never be one.
//
// DETERMINISM IS A CORRECTNESS REQUIREMENT, NOT A STYLE PREFERENCE.
// Inputs.Attributes is a Go map, and ranging over a map yields a random
// order on every run. If that order could reach the output, two Continue
// presses would generate different summaries, the dedupe UNIQUE index in
// migration 070 would stop absorbing retries, and a Player's history would
// silently duplicate. Every comparison below is therefore total, and
// rules.AttributeOrder -- never map order -- is the final tie-break.
func SelectPrimaryLens(in Inputs, rules Rules) Lens {
	order := rules.AttributeOrder
	if len(order) == 0 {
		// Defensive only. A caller that forgets Rules gets a stable
		// alphabetical order rather than a random one, so the failure is
		// visible in a diff instead of intermittent.
		for name := range in.Attributes {
			order = append(order, name)
		}
		sort.Strings(order)
	}
	rank := make(map[string]int, len(order))
	for i, name := range order {
		rank[name] = i
	}

	archetypeAttr := strings.TrimSpace(in.PrimaryAttribute)
	if archetypeAttr == "" && in.ArchetypeKey != "" && rules.Archetype != nil {
		if a, ok := rules.Archetype(in.ArchetypeKey); ok {
			archetypeAttr = strings.TrimSpace(a.PrimaryAttribute)
		}
	}
	// "absent or invalid": an archetype naming an attribute the Character
	// has no score for cannot be the lens, because the sentence it feeds
	// asserts a strength.
	archetypeValid := archetypeAttr != ""
	if archetypeValid {
		if _, ok := in.Attributes[archetypeAttr]; !ok {
			archetypeValid = false
		}
	}

	best := ""
	bestScore := 0
	for _, name := range order {
		score, ok := in.Attributes[name]
		if !ok {
			continue
		}
		if best == "" {
			best, bestScore = name, score
			continue
		}
		if score > bestScore {
			best, bestScore = name, score
			continue
		}
		if score == bestScore {
			// S1.7: "If several attributes tie, prefer the attribute
			// connected to the Character's archetype." Only the archetype
			// attribute can displace an equal-scoring incumbent; otherwise
			// canonical order wins, which keeps this total.
			if archetypeValid && name == archetypeAttr && best != archetypeAttr {
				best, bestScore = name, score
			}
		}
	}

	lens := Lens{}
	switch {
	case archetypeValid:
		lens.Attribute = archetypeAttr
		lens.Score = in.Attributes[archetypeAttr]
		lens.FromArchetype = true
	case best != "":
		lens.Attribute = best
		lens.Score = bestScore
	default:
		return lens
	}

	if line, ok := selectFaceSheetLine(lens.Attribute, in, rules, rank); ok {
		lens.Line = line
		lens.HasLine = true
	}
	return lens
}

// selectFaceSheetLine finds the single history line that best evidences the
// chosen lens, or reports false so the caller omits the quotation.
//
// Candidate terms, in preference order: the chosen attribute, the
// archetype's key skill, then the archetype's secondary attribute. A line
// matching an earlier term always beats a line matching a later one.
//
// Tie-break order, applied in full so the result is total:
//
//	(a) earlier candidate term
//	(b) higher recorded attribute score for the matched term
//	(c) earlier position in rules.AttributeOrder  <- closes map-order leakage
//	(d) higher StageNumber (later life stages are more characteristic)
//	(e) lower SortOrder
//	(f) lexicographically smaller ID
func selectFaceSheetLine(attribute string, in Inputs, rules Rules, rank map[string]int) (StageLine, bool) {
	if len(in.StageLines) == 0 {
		return StageLine{}, false
	}

	terms := []string{attribute}
	keySkill := strings.TrimSpace(in.KeySkill)
	secondary := strings.TrimSpace(in.SecondaryAttribute)
	if keySkill == "" || secondary == "" {
		if in.ArchetypeKey != "" && rules.Archetype != nil {
			if a, ok := rules.Archetype(in.ArchetypeKey); ok {
				if keySkill == "" {
					keySkill = strings.TrimSpace(a.KeySkill)
				}
				if secondary == "" {
					secondary = strings.TrimSpace(a.SecondaryAttribute)
				}
			}
		}
	}
	if keySkill != "" && !strings.EqualFold(keySkill, attribute) {
		terms = append(terms, keySkill)
	}
	if secondary != "" && !strings.EqualFold(secondary, attribute) && !strings.EqualFold(secondary, keySkill) {
		terms = append(terms, secondary)
	}

	type candidate struct {
		line     StageLine
		termIdx  int
		termName string
	}
	var found []candidate
	for termIdx, term := range terms {
		for _, line := range in.StageLines {
			if containsWholeWord(line.Title+" "+line.Body, term) {
				found = append(found, candidate{line: line, termIdx: termIdx, termName: term})
			}
		}
	}
	if len(found) == 0 {
		return StageLine{}, false
	}

	scoreOf := func(term string) int { return in.Attributes[term] }
	rankOf := func(term string) int {
		if r, ok := rank[term]; ok {
			return r
		}
		// A key skill is not an attribute and has no canonical rank. Sort
		// it after every attribute, deterministically.
		return len(rank) + 1
	}

	sort.SliceStable(found, func(i, j int) bool {
		a, b := found[i], found[j]
		if a.termIdx != b.termIdx {
			return a.termIdx < b.termIdx // (a)
		}
		if sa, sb := scoreOf(a.termName), scoreOf(b.termName); sa != sb {
			return sa > sb // (b)
		}
		if ra, rb := rankOf(a.termName), rankOf(b.termName); ra != rb {
			return ra < rb // (c)
		}
		if a.line.StageNumber != b.line.StageNumber {
			return a.line.StageNumber > b.line.StageNumber // (d)
		}
		if a.line.SortOrder != b.line.SortOrder {
			return a.line.SortOrder < b.line.SortOrder // (e)
		}
		return a.line.ID < b.line.ID // (f)
	})
	return found[0].line, true
}

// containsWholeWord reports whether needle appears in haystack as a whole
// word, case-insensitively. Whole-word matching matters: "Lore" must not
// match inside "explore", and "Craft" must not match inside "crafty",
// because a false match would attach a Face Sheet quotation to a strength it
// does not actually evidence -- exactly the unsupported claim S5.2 forbids.
func containsWholeWord(haystack, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	h := strings.ToLower(haystack)
	n := strings.ToLower(needle)

	from := 0
	for {
		idx := strings.Index(h[from:], n)
		if idx < 0 {
			return false
		}
		start := from + idx
		end := start + len(n)
		beforeOK := start == 0 || !isWordRune(rune(h[start-1]))
		afterOK := end == len(h) || !isWordRune(rune(h[end]))
		if beforeOK && afterOK {
			return true
		}
		from = start + 1
		if from >= len(h) {
			return false
		}
	}
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
