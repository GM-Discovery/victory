package storysofar

import (
	"fmt"
	"sort"
	"strings"
)

// The template library (kernel-75 S5.3).
//
// Every template is a pure function returning (DraftEvent, bool). Returning
// false OMITS the clause -- it never emits placeholder text, an "unknown", or
// a hedged sentence. That is S5.2's "omit missing clauses" implemented as
// control flow, and it is also what lets a Character who played before
// Kernel 75 (and therefore has no durable stance or Haggle rows) produce a
// complete, honest, well-formed story with fewer clauses rather than a
// story full of blanks.
//
// The templates may only read Inputs, whose field set is exactly S5.1's
// permitted source list. There is no path from here to anything else.
//
// Names are third-person and use the Character's name, matching S5.4's
// worked examples. Pronouns come from the Character's own recorded
// pronouns; when none are recorded the templates use "they", which is also
// what S5.4's Example B does.

// Milestone keys this package tests against. Duplicated as literals rather
// than imported from the tutorial package to keep storysofar a leaf (see the
// package doc). They are CHECK-constrained in migrations 066 and 069, so a
// drift here fails loudly in the dbtests rather than silently omitting a
// clause.
const (
	milestoneKessaCompleted = "kessa_intro_completed"
	milestoneDoorSubmitted  = "door_intention_submitted"
	milestoneRaCompleted    = "ra_intro_completed"
	milestoneGateOpened     = "tutorial_gate_opened"
	milestoneTutorialDone   = "tutorial_completed"
)

type template func(Inputs, Rules) (DraftEvent, bool)

// templates is the ordered library. Order is the chronological order of the
// resulting Story So Far, and it is fixed rather than derived from
// timestamps so that two clauses recorded in the same second still render in
// a sensible narrative sequence.
var templates = []template{
	tmplArrival,
	tmplFaceSheetEcho,
	tmplMerchantMet,
	tmplStance,
	tmplHaggle,
	tmplEquipment,
	tmplDoorIntention,
	tmplDialogue,
	tmplGateOpened,
	tmplTutorialCompleted,
}

// Generate applies the whole library.
//
// PURE: no database, no clock, no randomness, no map-iteration-order
// dependence. Same Inputs -> byte-identical []DraftEvent in a stable order.
//
// That purity is the reason Continue is retry-safe. The dedupe UNIQUE index
// in migration 070 can only absorb a retry if the retry re-derives the same
// (event_type, source_kind, source_ref) tuples; if Generate ever became
// impure, retries would start writing second copies of a Player's history
// instead of colliding. generate_test.go asserts this directly.
func Generate(in Inputs, rules Rules) []DraftEvent {
	out := []DraftEvent{}
	for _, t := range templates {
		if draft, ok := t(in, rules); ok {
			if draft.OccurredAt.IsZero() {
				draft.OccurredAt = in.CompletedAt
			}
			out = append(out, draft)
		}
	}
	// Equipment, Fate awards, and Help resolutions each yield zero-to-many
	// clauses (one row per source item), unlike the single-clause templates
	// in the slice above.
	out = append(out, generateEquipmentLines(in)...)
	out = append(out, generateFateLines(in)...)
	out = append(out, generateHelpLines(in)...)
	return out
}

// generateFateLines emits one clause per Director-awarded Fate ledger
// entry (kernel-88 spec §4.6, §19: "meaningful Fate awards... should be
// recorded"). Player self-spends never reach Inputs.FateAwards -- the
// caller has already filtered to award-reason rows -- so there is nothing
// for this template to filter further.
func generateFateLines(in Inputs) []DraftEvent {
	if len(in.FateAwards) == 0 {
		return nil
	}
	name := characterName(in)
	out := make([]DraftEvent, 0, len(in.FateAwards))
	for _, a := range in.FateAwards {
		if a.Delta == 0 {
			continue
		}
		verb := "was awarded"
		if a.Delta < 0 {
			verb = "had a Fate correction of"
		}
		out = append(out, DraftEvent{
			EventType:  EventFatePointsAwarded,
			Title:      "Fate awarded",
			Summary:    fmt.Sprintf("%s %s %d Fate.", name, verb, a.Delta),
			SourceKind: SourceSocioFateLedger,
			SourceRef:  a.LedgerID,
			OccurredAt: a.CreatedAt,
		})
	}
	return out
}

// generateHelpLines emits one clause per resolved Help/interrupt outcome
// (kernel-88 spec §19: "significant Help/interruption"). Only resolved
// entries reach Inputs.HelpResolutions -- opened-but-unresolved interrupts
// and cancellations are not meaningful history.
func generateHelpLines(in Inputs) []DraftEvent {
	if len(in.HelpResolutions) == 0 {
		return nil
	}
	out := make([]DraftEvent, 0, len(in.HelpResolutions))
	for _, h := range in.HelpResolutions {
		helper := strings.TrimSpace(h.HelperName)
		if helper == "" {
			helper = "A helper"
		}
		var summary string
		if h.Succeeded {
			primary := strings.TrimSpace(h.PrimaryActorName)
			if primary == "" {
				summary = fmt.Sprintf("%s helped, adding %d to the action underway.", helper, h.Overage)
			} else {
				summary = fmt.Sprintf("%s helped %s, adding %d to the action underway.", helper, primary, h.Overage)
			}
		} else {
			summary = fmt.Sprintf("%s tried to help, but the attempt fell short.", helper)
		}
		out = append(out, DraftEvent{
			EventType:  EventHelpResolved,
			Title:      "Help resolved",
			Summary:    summary,
			SourceKind: SourceSocioPendingAction,
			SourceRef:  h.PendingActionID,
			OccurredAt: h.ResolvedAt,
		})
	}
	return out
}

// --- helpers ---------------------------------------------------------------

func characterName(in Inputs) string {
	if n := strings.TrimSpace(in.CharacterName); n != "" {
		return n
	}
	return "This Character"
}

// subjectPronoun / possessivePronoun read the Character's own recorded
// pronouns. Never inferred from a name.
func possessivePronoun(in Inputs) string {
	p := strings.ToLower(strings.TrimSpace(in.Pronouns))
	switch {
	case strings.HasPrefix(p, "she"):
		return "her"
	case strings.HasPrefix(p, "he/") || p == "he" || strings.HasPrefix(p, "he "):
		return "his"
	default:
		return "their"
	}
}

func reflexivePronoun(in Inputs) string {
	p := strings.ToLower(strings.TrimSpace(in.Pronouns))
	switch {
	case strings.HasPrefix(p, "she"):
		return "herself"
	case strings.HasPrefix(p, "he/") || p == "he" || strings.HasPrefix(p, "he "):
		return "himself"
	default:
		return "themself"
	}
}

func archetypeLabel(in Inputs) string {
	if t := strings.TrimSpace(in.ArchetypeTitle); t != "" {
		return t
	}
	if k := strings.TrimSpace(in.ArchetypeKey); k != "" {
		return "the " + k
	}
	return ""
}

// lastAttempt returns the most recent attempt of a kind. "Most recent" is by
// recorded time with a total tie-break on ID-free fields, so ordering never
// depends on the slice arriving pre-sorted.
func lastAttempt(in Inputs, kind string) (Attempt, bool) {
	var found []Attempt
	for _, a := range in.Attempts {
		if a.Kind == kind {
			found = append(found, a)
		}
	}
	if len(found) == 0 {
		return Attempt{}, false
	}
	sort.SliceStable(found, func(i, j int) bool {
		if !found[i].CreatedAt.Equal(found[j].CreatedAt) {
			return found[i].CreatedAt.Before(found[j].CreatedAt)
		}
		if found[i].Total != found[j].Total {
			return found[i].Total < found[j].Total
		}
		return found[i].StanceKey < found[j].StanceKey
	})
	return found[len(found)-1], true
}

func successfulHaggle(in Inputs) (Attempt, bool) {
	var found []Attempt
	for _, a := range in.Attempts {
		if a.Kind == "haggle" && a.Success {
			found = append(found, a)
		}
	}
	if len(found) == 0 {
		return Attempt{}, false
	}
	sort.SliceStable(found, func(i, j int) bool {
		if !found[i].CreatedAt.Equal(found[j].CreatedAt) {
			return found[i].CreatedAt.Before(found[j].CreatedAt)
		}
		return found[i].Total < found[j].Total
	})
	return found[0], true
}

func hasAttemptOfKind(in Inputs, kind string) bool {
	for _, a := range in.Attempts {
		if a.Kind == kind {
			return true
		}
	}
	return false
}

// inventoryFromThisShow is already filtered by the loader; this only sorts
// it into a stable order for rendering.
func inventoryFromThisShow(in Inputs) []InventoryLine {
	out := append([]InventoryLine(nil), in.Inventory...)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].AcquiredAt.Equal(out[j].AcquiredAt) {
			return out[i].AcquiredAt.Before(out[j].AcquiredAt)
		}
		if out[i].ItemName != out[j].ItemName {
			return out[i].ItemName < out[j].ItemName
		}
		return out[i].InventoryItemID < out[j].InventoryItemID
	})
	return out
}

// --- templates -------------------------------------------------------------

// Cases 1/2 of S5.3: archetype identified, and archetype missing.
func tmplArrival(in Inputs, _ Rules) (DraftEvent, bool) {
	if len(in.Milestones) == 0 {
		return DraftEvent{}, false
	}
	name := characterName(in)
	// Arrival deliberately does not assert a strength. The lens sentence is
	// tmplFaceSheetEcho's job, and it fires only when a real Face Sheet line
	// backs it up -- so a Character with no history gets "entered the
	// Courtyard as the Guardian" and nothing further, which is S5.4's
	// Example B exactly.
	arch := archetypeLabel(in)
	if arch == "" {
		return DraftEvent{
			EventType:  EventArrival,
			Title:      "Into the Courtyard",
			Summary:    fmt.Sprintf("%s entered the Locked Courtyard.", name),
			SourceKind: SourceNone,
		}, true
	}
	return DraftEvent{
		EventType:  EventArrival,
		Title:      "Into the Courtyard",
		Summary:    fmt.Sprintf("%s entered the Locked Courtyard as %s.", name, arch),
		SourceKind: SourceNone,
	}, true
}

// Cases 3/4/5/6 of S5.3: one highest attribute; tied highest attributes
// resolved through archetype; matching Face Sheet line present; matching
// line absent.
//
// When no line matches, this template returns false and the quotation is
// omitted entirely rather than replaced -- S1.7's explicit instruction.
func tmplFaceSheetEcho(in Inputs, rules Rules) (DraftEvent, bool) {
	lens := SelectPrimaryLens(in, rules)
	if lens.Attribute == "" || !lens.HasLine {
		return DraftEvent{}, false
	}
	quoted := strings.TrimSpace(lens.Line.Body)
	if quoted == "" {
		quoted = strings.TrimSpace(lens.Line.Title)
	}
	if quoted == "" {
		return DraftEvent{}, false
	}
	name := characterName(in)
	return DraftEvent{
		EventType: EventFaceSheetEcho,
		Title:     fmt.Sprintf("A pattern already visible: %s", lens.Attribute),
		Summary: fmt.Sprintf(
			"%s's strongest lens was %s, a pattern already visible in %s history: %q",
			name, lens.Attribute, possessivePronoun(in), quoted),
		SourceKind: SourceWorkbookEntry,
		SourceRef:  lens.Line.ID,
	}, true
}

func tmplMerchantMet(in Inputs, _ Rules) (DraftEvent, bool) {
	if !in.Milestones[milestoneKessaCompleted] {
		return DraftEvent{}, false
	}
	return DraftEvent{
		EventType:  EventMerchantMet,
		Title:      "At Kessa's stall",
		Summary:    fmt.Sprintf("%s stopped at Kessa's stall and heard what she had to offer.", characterName(in)),
		SourceKind: SourceTutorialMilestone,
		SourceRef:  milestoneKessaCompleted,
	}, true
}

// Cases 10/11 of S5.3: no Kessa stance; stance attempted.
func tmplStance(in Inputs, _ Rules) (DraftEvent, bool) {
	name := characterName(in)
	if a, ok := lastAttempt(in, "stance"); ok {
		approach := strings.TrimSpace(a.StanceKey)
		if approach == "" {
			return DraftEvent{}, false
		}
		summary := fmt.Sprintf("At Kessa's stall, %s approached through %s.", name, approach)
		if d := strings.TrimSpace(a.Disposition); d != "" {
			summary = fmt.Sprintf("At Kessa's stall, %s approached through %s, and Kessa met it with %s.", name, approach, d)
		}
		return DraftEvent{
			EventType:  EventMerchantStance,
			Title:      "An approach at the stall",
			Summary:    summary,
			SourceKind: SourceInteractionAttempt,
			SourceRef:  a.StanceKey,
			OccurredAt: a.CreatedAt,
		}, true
	}
	// No stance attempted. Only worth a clause if they actually met Kessa;
	// otherwise there is nothing to say and nothing is said.
	if !in.Milestones[milestoneKessaCompleted] {
		return DraftEvent{}, false
	}
	return DraftEvent{
		EventType:  EventMerchantStance,
		Title:      "An approach at the stall",
		Summary:    fmt.Sprintf("%s let Kessa lead the conversation rather than pressing an approach.", name),
		SourceKind: SourceNone,
	}, true
}

// Case 12 of S5.3: Haggle attempted. Success, failure, and never-tried are
// three distinct clauses; exactly one can fire.
func tmplHaggle(in Inputs, _ Rules) (DraftEvent, bool) {
	name := characterName(in)
	if a, ok := successfulHaggle(in); ok {
		return DraftEvent{
			EventType:  EventMerchantHaggle,
			Title:      "A price talked down",
			Summary:    fmt.Sprintf("%s haggled with Kessa and got the better of it.", name),
			SourceKind: SourceInteractionAttempt,
			SourceRef:  "haggle_success",
			OccurredAt: a.CreatedAt,
		}, true
	}
	if a, ok := lastAttempt(in, "haggle"); ok {
		return DraftEvent{
			EventType:  EventMerchantHaggle,
			Title:      "A price held",
			Summary:    fmt.Sprintf("%s tried to haggle with Kessa. She held her price.", name),
			SourceKind: SourceInteractionAttempt,
			SourceRef:  "haggle_failure",
			OccurredAt: a.CreatedAt,
		}, true
	}
	if !in.Milestones[milestoneKessaCompleted] || hasAttemptOfKind(in, "haggle") {
		return DraftEvent{}, false
	}
	return DraftEvent{
		EventType:  EventMerchantHaggle,
		Title:      "A price accepted",
		Summary:    fmt.Sprintf("%s did not haggle.", name),
		SourceKind: SourceNone,
	}, true
}

// Cases 7/8/9 of S5.3: no purchase; one acquired item; several acquired
// items. The "no purchase" case is S5.4's Example C almost verbatim -- the
// kernel is explicit that leaving without buying is a real choice with a
// place in the Character's history, not an absence to apologize for.
func tmplEquipment(in Inputs, _ Rules) (DraftEvent, bool) {
	items := inventoryFromThisShow(in)
	if len(items) > 0 {
		return DraftEvent{}, false // handled by generateEquipmentLines
	}
	if !in.Milestones[milestoneKessaCompleted] {
		return DraftEvent{}, false
	}
	name := characterName(in)
	return DraftEvent{
		EventType: EventEquipmentAcquired,
		Title:     "Nothing taken from the stall",
		Summary: fmt.Sprintf(
			"%s spoke with Kessa but left the stall without taking new equipment. That choice is part of %s history; preparation did not require a purchase.",
			name, possessivePronoun(in)),
		SourceKind: SourceNone,
	}, true
}

// generateEquipmentLines emits one clause per acquired item, plus a summary
// sentence when there are several. One row per item keeps each acquisition
// individually attributable to its inventory row, which is what makes the
// dedupe key stable across retries.
func generateEquipmentLines(in Inputs) []DraftEvent {
	items := inventoryFromThisShow(in)
	if len(items) == 0 {
		return nil
	}
	name := characterName(in)
	out := make([]DraftEvent, 0, len(items)+1)

	if len(items) == 1 {
		out = append(out, DraftEvent{
			EventType: EventEquipmentAcquired,
			Title:     items[0].ItemName,
			Summary: fmt.Sprintf("%s equipped %s with %s.",
				name, reflexivePronoun(in), items[0].ItemName),
			SourceKind: SourceInventoryItem,
			SourceRef:  items[0].InventoryItemID,
			OccurredAt: items[0].AcquiredAt,
		})
		return out
	}

	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.ItemName)
		out = append(out, DraftEvent{
			EventType:  EventEquipmentAcquired,
			Title:      it.ItemName,
			Summary:    fmt.Sprintf("%s took %s from Kessa's stall.", name, it.ItemName),
			SourceKind: SourceInventoryItem,
			SourceRef:  it.InventoryItemID,
			OccurredAt: it.AcquiredAt,
		})
	}
	out = append(out, DraftEvent{
		EventType: EventEquipmentAcquired,
		Title:     "Equipped for what came next",
		Summary: fmt.Sprintf("%s equipped %s with %s.",
			name, reflexivePronoun(in), joinList(names)),
		SourceKind: SourceNone,
	})
	return out
}

func joinList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

// Case 13 of S5.3: door intention present.
//
// The Player's words are stored VERBATIM and are never wrapped in authored
// narration. Kernel 74 S1.4 settled this: a Player may type a complete
// sentence, so "You start to <input>" is unsafe grammar, and rewriting
// someone's own words back at them is worse than unsafe. The quotation marks
// here are the only decoration, and every renderer inserts the text through
// textContent so markup in it stays literal.
func tmplDoorIntention(in Inputs, _ Rules) (DraftEvent, bool) {
	intention := strings.TrimSpace(in.DoorIntention)
	if intention == "" {
		if !in.Milestones[milestoneDoorSubmitted] {
			return DraftEvent{}, false
		}
		return DraftEvent{
			EventType:  EventDoorIntention,
			Title:      "At the locked gate",
			Summary:    fmt.Sprintf("%s chose an approach of %s own before Ra interrupted.", characterName(in), possessivePronoun(in)),
			SourceKind: SourceNone,
		}, true
	}
	return DraftEvent{
		EventType: EventDoorIntention,
		Title:     "At the locked gate",
		Summary: fmt.Sprintf("At the gate, %s intended to: %q",
			characterName(in), intention),
		SourceKind: SourceFreeformSubmission,
		SourceRef:  "door_intention",
	}, true
}

func tmplDialogue(in Inputs, _ Rules) (DraftEvent, bool) {
	if !in.Milestones[milestoneRaCompleted] {
		return DraftEvent{}, false
	}
	optional := 0
	for _, t := range in.Topics {
		if !t.Required {
			optional++
		}
	}
	name := characterName(in)
	summary := "Ra interrupted before the attempt resolved and introduced the Crown Bet."
	if optional > 0 {
		summary = fmt.Sprintf(
			"Ra interrupted before the attempt resolved and introduced the Crown Bet. %s stayed to ask more than %s had to.",
			name, possessivePronoun(in))
	}
	return DraftEvent{
		EventType:  EventDialogueLearned,
		Title:      "The Crown Bet",
		Summary:    summary,
		SourceKind: SourceDialogueTopic,
		SourceRef:  "required",
	}, true
}

func tmplGateOpened(in Inputs, _ Rules) (DraftEvent, bool) {
	if !in.Milestones[milestoneGateOpened] {
		return DraftEvent{}, false
	}
	return DraftEvent{
		EventType:  EventGateOpened,
		Title:      "The gate opened",
		Summary:    "Ra worked the concealed lock, the bolt withdrew, and the gate swung open.",
		SourceKind: SourceTutorialMilestone,
		SourceRef:  milestoneGateOpened,
	}, true
}

// Case 14 of S5.3: tutorial completed. The closing anchor.
func tmplTutorialCompleted(in Inputs, _ Rules) (DraftEvent, bool) {
	if !in.Milestones[milestoneTutorialDone] {
		return DraftEvent{}, false
	}
	return DraftEvent{
		EventType: EventTutorialCompleted,
		Title:     "The guided beginning is complete",
		Summary: fmt.Sprintf(
			"%s has completed the guided Socio tutorial. These events are now part of %s continuing history.",
			characterName(in), possessivePronoun(in)),
		SourceKind: SourceTutorialMilestone,
		SourceRef:  milestoneTutorialDone,
	}, true
}
