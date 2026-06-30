package characters

import "fmt"

const Chapter2Version = "0.3"
const Chapter2StartingFP = 12
const Chapter2EnhancementCost = 3
const Chapter2AttributeCap = 10

type Chapter2BonusChoice struct {
	ID              string `json:"id"`
	ChoiceKey       string `json:"choice_key"` // "A","B","C","D"
	Name            string `json:"name"`
	TargetAttribute string `json:"target_attribute"`
	Modifier        int    `json:"modifier"`
}

type Chapter2Companion struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CostFP      int    `json:"cost_fp"`
	Description string `json:"description"`
}

type Chapter2Enhancement struct {
	ID             string `json:"id"`
	CostFP         int    `json:"cost_fp"`
	RollExpression string `json:"roll_expression"`
	FinalResultMin int    `json:"final_result_min"`
	FinalResultMax int    `json:"final_result_max"`
}

type Chapter2Trait struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CostFP      int    `json:"cost_fp"`
	Description string `json:"description"`
}

type Chapter2Stage struct {
	ID               string                `json:"id"`
	StageNumber      int                   `json:"stage_number"`
	Name             string                `json:"name"`
	PrimaryAttribute string                `json:"primary_attribute"`
	Purpose          string                `json:"purpose"`
	CanonicalIntro   string                `json:"canonical_intro"`
	BonusChoices     []Chapter2BonusChoice `json:"bonus_choices"`
	Companion        *Chapter2Companion    `json:"companion,omitempty"`
	Enhancement      Chapter2Enhancement   `json:"enhancement"`
	Traits           []Chapter2Trait       `json:"traits"`
	// Stage 9 only: requires special 1-to-Craft allocation
	HasSpecialAllocation bool `json:"has_special_allocation,omitempty"`
	// Stage 2 only: optional freeform representation field
	HasRepresentationField bool `json:"has_representation_field,omitempty"`
}

// Chapter2Rules contains the complete versioned Chapter 2 stage data.
var Chapter2Rules = []Chapter2Stage{
	{
		ID: "STAGE_01", StageNumber: 1, Name: "Preconception", PrimaryAttribute: "Spirit",
		Purpose:        "Soul's essence before birth",
		CanonicalIntro: "Folic acid and how well your parents were able to take care of themselves.",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S01_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S01_A", ChoiceKey: "A", Name: "Early Spiritual Awakening", TargetAttribute: "Spirit", Modifier: 1},
			{ID: "BONUS_S01_B", ChoiceKey: "B", Name: "Destined Purpose", TargetAttribute: "Resolve", Modifier: 1},
			{ID: "BONUS_S01_C", ChoiceKey: "C", Name: "Divine Favor", TargetAttribute: "Presence", Modifier: 1},
			{ID: "BONUS_S01_D", ChoiceKey: "D", Name: "Mystical Insight", TargetAttribute: "Intellect", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S01", Name: "Entwined Fate", CostFP: 1, Description: "Share mystical destiny with another character"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S01_01_DIVINE_TOUCHED", Name: "Divine Touched", CostFP: 5, Description: "Spiritual authority, blessing abilities, divine insight"},
			{ID: "TRAIT_S01_02_ANCESTRAL_CONNECTION", Name: "Ancestral Connection", CostFP: 4, Description: "Commune with past generations, inherited wisdom"},
		},
	},
	{
		ID: "STAGE_02", StageNumber: 2, Name: "Genetics", PrimaryAttribute: "Might",
		Purpose:                "Physical foundation and constitution",
		CanonicalIntro:         "Our body's might allow our spirit to assert its will in the universe. We were given no other instrument, but made tools anyway.",
		HasRepresentationField: true,
		Enhancement:            Chapter2Enhancement{ID: "ENH_S02_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S02_A", ChoiceKey: "A", Name: "Warrior Heritage", TargetAttribute: "Might", Modifier: 1},
			{ID: "BONUS_S02_B", ChoiceKey: "B", Name: "Graceful Bearing", TargetAttribute: "Grace", Modifier: 1},
			{ID: "BONUS_S02_C", ChoiceKey: "C", Name: "Keen Senses", TargetAttribute: "Awareness", Modifier: 1},
			{ID: "BONUS_S02_D", ChoiceKey: "D", Name: "Family Wisdom", TargetAttribute: "Lore", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S02", Name: "Familial Bond", CostFP: 1, Description: "Establish family connection with another character"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S02_01_BERSERKER_HERITAGE", Name: "Berserker Heritage", CostFP: 5, Description: "Controlled fury, damage bonuses, intimidation presence"},
			{ID: "TRAIT_S02_02_ATHLETIC_BLOODLINE", Name: "Athletic Bloodline", CostFP: 4, Description: "Natural physical prowess, endurance bonuses"},
		},
	},
	{
		ID: "STAGE_03", StageNumber: 3, Name: "Infancy", PrimaryAttribute: "Empathy",
		Purpose:        "Bonding, care, emotional foundation",
		CanonicalIntro: "Our first struggle, and the foundation of our thought, how we relate to the first others, our folks, says a lot about our relationships and what we need to overcome in ourselves.",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S03_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S03_A", ChoiceKey: "A", Name: "Beloved Child", TargetAttribute: "Empathy", Modifier: 1},
			{ID: "BONUS_S03_B", ChoiceKey: "B", Name: "Alert Baby", TargetAttribute: "Awareness", Modifier: 1},
			{ID: "BONUS_S03_C", ChoiceKey: "C", Name: "Healthy Constitution", TargetAttribute: "Might", Modifier: 1},
			{ID: "BONUS_S03_D", ChoiceKey: "D", Name: "Peaceful Presence", TargetAttribute: "Spirit", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S03", Name: "Shared Caretaker", CostFP: 1, Description: "Share upbringing with another character"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S03_01_EMOTIONAL_INTUITION", Name: "Emotional Intuition", CostFP: 5, Description: "Read true emotions, detect lies, provide comfort"},
			{ID: "TRAIT_S03_02_ANIMAL_WHISPERER", Name: "Animal Whisperer", CostFP: 4, Description: "Communicate with creatures, animal allies"},
		},
	},
	{
		ID: "STAGE_04", StageNumber: 4, Name: "Toddlerhood", PrimaryAttribute: "Grace",
		Purpose:        "Motor development, early coordination",
		CanonicalIntro: "You know what they say about grace under pressure. Toddlerhood is the early sandbox of skill refinement.",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S04_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S04_A", ChoiceKey: "A", Name: "Natural Dancer", TargetAttribute: "Grace", Modifier: 1},
			{ID: "BONUS_S04_B", ChoiceKey: "B", Name: "Fearless Explorer", TargetAttribute: "Awareness", Modifier: 1},
			{ID: "BONUS_S04_C", ChoiceKey: "C", Name: "Strong Toddler", TargetAttribute: "Might", Modifier: 1},
			{ID: "BONUS_S04_D", ChoiceKey: "D", Name: "Charming Child", TargetAttribute: "Presence", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S04", Name: "Childhood Friend", CostFP: 1, Description: "Establish lifelong friendship from toddlerhood"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S04_01_PERFECT_BALANCE", Name: "Perfect Balance", CostFP: 4, Description: "Never fall, impossible acrobatics, poise under pressure"},
			{ID: "TRAIT_S04_02_UNSEEN_MOVEMENT", Name: "Unseen Movement", CostFP: 5, Description: "Enhanced stealth, environmental blending"},
		},
	},
	{
		ID: "STAGE_05", StageNumber: 5, Name: "Childhood", PrimaryAttribute: "Awareness",
		Purpose:        "Exploration, curiosity about the world",
		CanonicalIntro: "With new found freedoms at walking, talking, and thinking, it follows that a child might have questions and start seeking answers. What's over there? Can I throw far? Who approaches?",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S05_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S05_A", ChoiceKey: "A", Name: "Eagle Eyed", TargetAttribute: "Awareness", Modifier: 1},
			{ID: "BONUS_S05_B", ChoiceKey: "B", Name: "Quick Learner", TargetAttribute: "Intellect", Modifier: 1},
			{ID: "BONUS_S05_C", ChoiceKey: "C", Name: "Helper", TargetAttribute: "Craft", Modifier: 1},
			{ID: "BONUS_S05_D", ChoiceKey: "D", Name: "Physically Aware", TargetAttribute: "Grace", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S05", Name: "Exploration Partner", CostFP: 1, Description: "Establish childhood adventure companion"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S05_01_DANGER_SENSE", Name: "Danger Sense", CostFP: 5, Description: "Predict threats, ambush immunity, survival instincts"},
			{ID: "TRAIT_S05_02_TRACKER_S_GIFT", Name: "Tracker's Gift", CostFP: 4, Description: "Follow any trail, environmental reading, hunting mastery"},
		},
	},
	{
		ID: "STAGE_06", StageNumber: 6, Name: "School Age", PrimaryAttribute: "Intellect",
		Purpose:        "Formal learning, reasoning development",
		CanonicalIntro: "There are many kinds of intelligences: visual intelligence, social intelligence, cooking intelligence, common intelligence, reasoning intelligence, proprio-motor intelligence…",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S06_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S06_A", ChoiceKey: "A", Name: "Star Pupil", TargetAttribute: "Intellect", Modifier: 1},
			{ID: "BONUS_S06_B", ChoiceKey: "B", Name: "Outcast", TargetAttribute: "Resolve", Modifier: 1},
			{ID: "BONUS_S06_C", ChoiceKey: "C", Name: "Helper", TargetAttribute: "Empathy", Modifier: 1},
			{ID: "BONUS_S06_D", ChoiceKey: "D", Name: "Leader", TargetAttribute: "Presence", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S06", Name: "Study Buddy", CostFP: 1, Description: "Establish academic partnership"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S06_01_EIDETIC_MEMORY", Name: "Eidetic Memory", CostFP: 5, Description: "Perfect recall, research mastery, information advantages"},
			{ID: "TRAIT_S06_02_ANALYTICAL_GENIUS", Name: "Analytical Genius", CostFP: 4, Description: "Pattern recognition, problem-solving, deduction bonuses"},
		},
	},
	{
		ID: "STAGE_07", StageNumber: 7, Name: "Upper School", PrimaryAttribute: "Lore",
		Purpose:        "Specialized knowledge, cultural education",
		CanonicalIntro: "You may be smart, but without exposure to the basic understanding of gravity, your attempts at rocket science are doomed to failure.",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S07_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S07_A", ChoiceKey: "A", Name: "Living Library", TargetAttribute: "Lore", Modifier: 1},
			{ID: "BONUS_S07_B", ChoiceKey: "B", Name: "Master Craftsman", TargetAttribute: "Craft", Modifier: 1},
			{ID: "BONUS_S07_C", ChoiceKey: "C", Name: "Natural Philosopher", TargetAttribute: "Intellect", Modifier: 1},
			{ID: "BONUS_S07_D", ChoiceKey: "D", Name: "Naturally Graceful", TargetAttribute: "Grace", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S07", Name: "Academic Rival", CostFP: 2, Description: "Competitive relationship that pushes both to excel"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S07_01_LIVING_LIBRARY", Name: "Living Library", CostFP: 5, Description: "Vast knowledge, cultural fluency, historical insight"},
			{ID: "TRAIT_S07_02_LINGUISTIC_PRODIGY", Name: "Linguistic Prodigy", CostFP: 4, Description: "Learn languages near instantly, communication bonuses"},
		},
	},
	{
		ID: "STAGE_08", StageNumber: 8, Name: "Teen/Middle", PrimaryAttribute: "Presence",
		Purpose:        "Social identity, charisma development",
		CanonicalIntro: "Does your character know who they are? Who are they?",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S08_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S08_A", ChoiceKey: "A", Name: "Extreme Training", TargetAttribute: "Might", Modifier: 1},
			{ID: "BONUS_S08_B", ChoiceKey: "B", Name: "Spiritual Awakening", TargetAttribute: "Spirit", Modifier: 1},
			{ID: "BONUS_S08_C", ChoiceKey: "C", Name: "Crafting Away", TargetAttribute: "Craft", Modifier: 1},
			{ID: "BONUS_S08_D", ChoiceKey: "D", Name: "Steady Heart", TargetAttribute: "Resolve", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S08", Name: "Coming of Age Pact", CostFP: 1, Description: "Make solemn vow with another character"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S08_01_SILVER_TONGUE", Name: "Silver Tongue", CostFP: 5, Description: "Master persuasion, social manipulation, influence networks"},
			{ID: "TRAIT_S08_02_NATURAL_LEADER", Name: "Natural Leader", CostFP: 4, Description: "Command respect, inspire others, rally abilities"},
		},
	},
	{
		ID: "STAGE_09", StageNumber: 9, Name: "High School", PrimaryAttribute: "Craft",
		Purpose:              "Practical skills, hands-on learning",
		HasSpecialAllocation: true,
		CanonicalIntro:       "Everyone has some Craft in them, but people hone their craft in different domains. Some children study craft, grace, or lore throughout their lives. At this stage, consciously determine how your character spent that time. One point from the final roll must go to Craft. Allocate every remaining point to other attributes through the activities that developed those capacities.",
		Enhancement:          Chapter2Enhancement{ID: "ENH_S09_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S09_A", ChoiceKey: "A", Name: "Master Student", TargetAttribute: "Craft", Modifier: 1},
			{ID: "BONUS_S09_B", ChoiceKey: "B", Name: "Caring Mentor", TargetAttribute: "Empathy", Modifier: 1},
			{ID: "BONUS_S09_C", ChoiceKey: "C", Name: "Renaissance Student", TargetAttribute: "Lore", Modifier: 1},
			{ID: "BONUS_S09_D", ChoiceKey: "D", Name: "Project Leader", TargetAttribute: "Presence", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S09", Name: "Project Partner", CostFP: 1, Description: "Establish creative collaboration"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S09_01_MASTER_ARTISAN", Name: "Master Artisan", CostFP: 5, Description: "Create masterworks, innovation bonuses, resource advantages"},
			{ID: "TRAIT_S09_02_TECHNICAL_SAVANT", Name: "Technical Savant", CostFP: 4, Description: "Understand any device, repair mastery, invention abilities"},
		},
	},
	{
		ID: "STAGE_10", StageNumber: 10, Name: "Young Adult", PrimaryAttribute: "Resolve",
		Purpose:        "Life direction, independence, persistence",
		CanonicalIntro: "That's it, you're an adult now and expected to \"make it\" \"somehow.\" You know what doesn't cost fate points? Being kind to one another. Good luck!",
		Enhancement:    Chapter2Enhancement{ID: "ENH_S10_EXTRA_DIE", CostFP: 3, RollExpression: "1d6; keep higher", FinalResultMin: 1, FinalResultMax: 6},
		BonusChoices: []Chapter2BonusChoice{
			{ID: "BONUS_S10_A", ChoiceKey: "A", Name: "Iron Will", TargetAttribute: "Resolve", Modifier: 1},
			{ID: "BONUS_S10_B", ChoiceKey: "B", Name: "Shared Experience", TargetAttribute: "Empathy", Modifier: 1},
			{ID: "BONUS_S10_C", ChoiceKey: "C", Name: "Life Experience", TargetAttribute: "Lore", Modifier: 1},
			{ID: "BONUS_S10_D", ChoiceKey: "D", Name: "Inner Peace", TargetAttribute: "Spirit", Modifier: 1},
		},
		Companion: &Chapter2Companion{ID: "COMP_S10", Name: "Life Path Ally", CostFP: 1, Description: "Establish shared adult goals"},
		Traits: []Chapter2Trait{
			{ID: "TRAIT_S10_01_UNBREAKABLE_WILL", Name: "Unbreakable Will", CostFP: 5, Description: "Immunity to mental effects, inspire determination in others"},
			{ID: "TRAIT_S10_02_PHOENIX_SPIRIT", Name: "Phoenix Spirit", CostFP: 4, Description: "Recover from setbacks faster, turn failure into strength"},
		},
	},
}

// AllChapter2Attributes lists the ten attributes in canonical order.
var AllChapter2Attributes = []string{
	"Spirit", "Might", "Empathy", "Grace", "Awareness",
	"Intellect", "Lore", "Presence", "Craft", "Resolve",
}

// AttributeScaleRatings maps a final score (0-10) to its canonical rating.
var AttributeScaleRatings = map[int]string{
	0: "Defeated", 1: "Difficult", 2: "Far Below Average", 3: "Below Average",
	4: "Slightly Below Average", 5: "Average", 6: "Above Average",
	7: "Excellent", 8: "Blessed Beginning", 9: "Sacred Soul", 10: "Greatest",
}

// Chapter2StageForNumber returns the stage data for a given stage number (1-10).
func Chapter2StageForNumber(n int) (Chapter2Stage, bool) {
	for _, s := range Chapter2Rules {
		if s.StageNumber == n {
			return s, true
		}
	}
	return Chapter2Stage{}, false
}

// Chapter2BonusChoiceByID finds a bonus choice by stable ID across all stages.
func Chapter2BonusChoiceByID(id string) (Chapter2BonusChoice, bool) {
	for _, stage := range Chapter2Rules {
		for _, bc := range stage.BonusChoices {
			if bc.ID == id {
				return bc, true
			}
		}
	}
	return Chapter2BonusChoice{}, false
}

// Chapter2TraitByID finds a trait by stable ID across all stages.
func Chapter2TraitByID(id string) (Chapter2Trait, bool) {
	for _, stage := range Chapter2Rules {
		for _, t := range stage.Traits {
			if t.ID == id {
				return t, true
			}
		}
	}
	return Chapter2Trait{}, false
}

// ValidateChapter2Rules panics at startup if the rules data is malformed.
// This is a development guard; production builds ignore it via init().
func ValidateChapter2Rules() error {
	if len(Chapter2Rules) != 10 {
		return fmt.Errorf("chapter2: expected 10 stages, got %d", len(Chapter2Rules))
	}
	attrCovered := map[string]bool{}
	for _, s := range Chapter2Rules {
		if s.StageNumber < 1 || s.StageNumber > 10 {
			return fmt.Errorf("chapter2: stage %q has invalid number %d", s.ID, s.StageNumber)
		}
		if len(s.BonusChoices) != 4 {
			return fmt.Errorf("chapter2: stage %s has %d bonus choices, want 4", s.ID, len(s.BonusChoices))
		}
		if len(s.Traits) != 2 {
			return fmt.Errorf("chapter2: stage %s has %d traits, want 2", s.ID, len(s.Traits))
		}
		if s.Companion == nil {
			return fmt.Errorf("chapter2: stage %s missing companion", s.ID)
		}
		attrCovered[s.PrimaryAttribute] = true
	}
	for _, attr := range AllChapter2Attributes {
		if !attrCovered[attr] {
			return fmt.Errorf("chapter2: attribute %q not covered as a primary attribute", attr)
		}
	}
	return nil
}
