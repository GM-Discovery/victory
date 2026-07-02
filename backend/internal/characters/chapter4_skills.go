package characters

import "fmt"

// Chapter4SkillCatalogueVersion identifies the versioned skill catalogue
// this data was ported from (chapter-4-skill-catalogue-v0.9-draft.json),
// reconciled per socio-skill-catalogue-audit-v0.1.md.
const Chapter4SkillCatalogueVersion = "0.9-draft"

type Chapter4SkillHelper struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Chapter4Skill struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	AttributeID     string                `json:"attribute_id"`
	AttributeName   string                `json:"attribute_name"`
	SourceOrder     int                   `json:"source_order"`
	CardDescription string                `json:"card_description"`
	Helpers         []Chapter4SkillHelper `json:"helpers"`
}

type Chapter4Attribute struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourceOrder int    `json:"source_order"`
}

// Chapter4Attributes is the 10 Socio- attributes in canonical source order.
var Chapter4Attributes = []Chapter4Attribute{
	{ID: "ATTR_AWARENESS", Name: "Awareness", SourceOrder: 1},
	{ID: "ATTR_CRAFT", Name: "Craft", SourceOrder: 2},
	{ID: "ATTR_EMPATHY", Name: "Empathy", SourceOrder: 3},
	{ID: "ATTR_GRACE", Name: "Grace", SourceOrder: 4},
	{ID: "ATTR_INTELLECT", Name: "Intellect", SourceOrder: 5},
	{ID: "ATTR_LORE", Name: "Lore", SourceOrder: 6},
	{ID: "ATTR_MIGHT", Name: "Might", SourceOrder: 7},
	{ID: "ATTR_PRESENCE", Name: "Presence", SourceOrder: 8},
	{ID: "ATTR_RESOLVE", Name: "Resolve", SourceOrder: 9},
	{ID: "ATTR_SPIRIT", Name: "Spirit", SourceOrder: 10},
}

// Chapter4Skills is the complete versioned 100-skill catalogue (10 attributes
// x 10 skills), ported verbatim from chapter-4-skill-catalogue-v0.9-draft.json.
var Chapter4Skills = []Chapter4Skill{
	{
		ID: "SKILL_AWARENESS_ALERTNESS", Name: "Alertness", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 1, CardDescription: "Fast-twitch attention to sudden change",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_ALERTNESS_1", Name: "Chain Reaction", Description: "Alert others in proximity"}, {ID: "HELPER_AWARENESS_ALERTNESS_2", Name: "Draw and Point", Description: "Grant ally advantage vs fast attacker"}},
	},
	{
		ID: "SKILL_AWARENESS_ANIMAL_HANDLING", Name: "Animal Handling", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 2, CardDescription: "Reading, calming, or directing animals",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_ANIMAL_HANDLING_1", Name: "Soothing Touch", Description: "Suppress status effect on animal"}, {ID: "HELPER_AWARENESS_ANIMAL_HANDLING_2", Name: "Tactical Mount", Description: "Grant movement-based repositioning"}},
	},
	{
		ID: "SKILL_AWARENESS_INSIGHT", Name: "Insight", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 3, CardDescription: "Reading intentions, emotions, hidden motives",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_INSIGHT_1", Name: "Emotion Echo", Description: "Clarify stance choice for ally"}, {ID: "HELPER_AWARENESS_INSIGHT_2", Name: "Tell Spotter", Description: "Catch inconsistencies or subtle lies"}},
	},
	{
		ID: "SKILL_AWARENESS_INTUITION", Name: "Intuition", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 4, CardDescription: "Non-analytical sense of something's wrong",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_INTUITION_1", Name: "Gut Signal", Description: "Warn ally of narrative danger"}, {ID: "HELPER_AWARENESS_INTUITION_2", Name: "Path Shift", Description: "Grant reroll on direction-based choice"}},
	},
	{
		ID: "SKILL_AWARENESS_PERCEPTION", Name: "Perception", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 5, CardDescription: "General environmental awareness, sensory acuity",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_PERCEPTION_1", Name: "Snap Point", Description: "Signal ally to hidden threat"}, {ID: "HELPER_AWARENESS_PERCEPTION_2", Name: "Sensory Relay", Description: "Pass awareness result to party"}},
	},
	{
		ID: "SKILL_AWARENESS_SEARCH", Name: "Search", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 6, CardDescription: "Focused examination of areas, objects, patterns",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_SEARCH_1", Name: "Tool Sweep", Description: "Ally rerolls trap detection"}, {ID: "HELPER_AWARENESS_SEARCH_2", Name: "Sorted Sweep", Description: "Filter noise from useful finds"}},
	},
	{
		ID: "SKILL_AWARENESS_SURVIVAL", Name: "Survival", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 7, CardDescription: "Sensing and adapting to natural environments",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_SURVIVAL_1", Name: "Camp Scout", Description: "Set up temporary recovery zone"}, {ID: "HELPER_AWARENESS_SURVIVAL_2", Name: "Signal Trail", Description: "Lead others to safe zones"}},
	},
	{
		ID: "SKILL_AWARENESS_TRACKING", Name: "Tracking", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 8, CardDescription: "Following physical trails, behavioral patterns",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_TRACKING_1", Name: "Pace Reader", Description: "Predict when target will rest"}, {ID: "HELPER_AWARENESS_TRACKING_2", Name: "Track Exchange", Description: "Swap trail signs between trackers"}},
	},
	{
		ID: "SKILL_AWARENESS_TRAP_SENSE", Name: "Trap Sense", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 9, CardDescription: "Detecting danger through instinct and detail",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_TRAP_SENSE_1", Name: "Silent Stop", Description: "Cancel ally movement"}, {ID: "HELPER_AWARENESS_TRAP_SENSE_2", Name: "Design Mimic", Description: "Use trap signs for intimidation"}},
	},
	{
		ID: "SKILL_AWARENESS_VIGILANCE", Name: "Vigilance", AttributeID: "ATTR_AWARENESS", AttributeName: "Awareness",
		SourceOrder: 10, CardDescription: "Sustained attentiveness over time",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_AWARENESS_VIGILANCE_1", Name: "Relay Whisper", Description: "Share incoming threat silently"}, {ID: "HELPER_AWARENESS_VIGILANCE_2", Name: "Shift Change", Description: "Reduce exhaustion penalties"}},
	},
	{
		ID: "SKILL_CRAFT_ARTIFICE", Name: "Artifice", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 1, CardDescription: "Enchanting, binding magic/tech into objects",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_ARTIFICE_1", Name: "Resonant Anchor", Description: "Retain 1 component on failed roll"}, {ID: "HELPER_CRAFT_ARTIFICE_2", Name: "Harmonic Fusion", Description: "Combine two magical effects"}},
	},
	{
		ID: "SKILL_CRAFT_ARTISTRY", Name: "Artistry", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 2, CardDescription: "Expressive creation for beauty and meaning",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_ARTISTRY_1", Name: "Shared Canvas", Description: "Ally uses status to grant extra die"}, {ID: "HELPER_CRAFT_ARTISTRY_2", Name: "Emotional Resonance", Description: "Influence emotional state"}},
	},
	{
		ID: "SKILL_CRAFT_CONSTRUCTION", Name: "Construction", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 3, CardDescription: "Building physical structures",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_CONSTRUCTION_1", Name: "Safety Anchor Point", Description: "Negate first failed coordination"}, {ID: "HELPER_CRAFT_CONSTRUCTION_2", Name: "Load Distribution", Description: "Enhance structural integrity"}},
	},
	{
		ID: "SKILL_CRAFT_COOKING", Name: "Cooking", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 4, CardDescription: "Preparing meals, performance-enhancing food",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_COOKING_1", Name: "Savoring Steam", Description: "Allies clear 1 minor status"}, {ID: "HELPER_CRAFT_COOKING_2", Name: "Culinary Chemistry", Description: "Create food with temp bonuses"}},
	},
	{
		ID: "SKILL_CRAFT_DESIGN", Name: "Design", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 5, CardDescription: "Planning through form and function",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_DESIGN_1", Name: "Form Speaks", Description: "+1d4 to first roll with design"}, {ID: "HELPER_CRAFT_DESIGN_2", Name: "Ergonomic Flow", Description: "Reduce coordination complexity by 1"}},
	},
	{
		ID: "SKILL_CRAFT_MASTERWORK_CREATION", Name: "Masterwork Creation", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 6, CardDescription: "Creating unique legacy items",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_MASTERWORK_CREATION_1", Name: "Legacy Link", Description: "Gain bond, reroll later"}, {ID: "HELPER_CRAFT_MASTERWORK_CREATION_2", Name: "Evolutionary Design", Description: "Item grows with user"}},
	},
	{
		ID: "SKILL_CRAFT_REPAIR", Name: "Repair", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 7, CardDescription: "Diagnosing and restoring broken items",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_REPAIR_1", Name: "Field Jury Rig", Description: "Restore 1 function for scene"}, {ID: "HELPER_CRAFT_REPAIR_2", Name: "System Restoration", Description: "Improve item condition"}},
	},
	{
		ID: "SKILL_CRAFT_SMITHING", Name: "Smithing", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 8, CardDescription: "Metalworking, forging weapons/armor",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_SMITHING_1", Name: "Heat Synchrony", Description: "Helpers reroll 1 die"}, {ID: "HELPER_CRAFT_SMITHING_2", Name: "Alloy Mastery", Description: "+1d4 to final quality"}},
	},
	{
		ID: "SKILL_CRAFT_TAILORING", Name: "Tailoring", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 9, CardDescription: "Designing and crafting garments",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_TAILORING_1", Name: "Fit Check", Description: "Reduce disguise detection TV by 2"}, {ID: "HELPER_CRAFT_TAILORING_2", Name: "Style Synthesis", Description: "Clothing grants social bonuses"}},
	},
	{
		ID: "SKILL_CRAFT_TOOLCRAFT", Name: "Toolcraft", AttributeID: "ATTR_CRAFT", AttributeName: "Craft",
		SourceOrder: 10, CardDescription: "Designing and creating functional tools",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_CRAFT_TOOLCRAFT_1", Name: "Function First", Description: "Treat tool as matching skill"}, {ID: "HELPER_CRAFT_TOOLCRAFT_2", Name: "Modular Design", Description: "Create multi-function tool"}},
	},
	{
		ID: "SKILL_EMPATHY_CAREGIVING", Name: "Caregiving", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 1, CardDescription: "Attending to physical wounds",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_CAREGIVING_1", Name: "Attend Patient", Description: "Reduce recovery time"}, {ID: "HELPER_EMPATHY_CAREGIVING_2", Name: "Assist Physician", Description: "Reduce coordination complexity"}},
	},
	{
		ID: "SKILL_EMPATHY_COMFORT", Name: "Comfort", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 2, CardDescription: "Immediate emotional first aid",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_COMFORT_1", Name: "Shared Burden", Description: "Transfer 1 status to self"}, {ID: "HELPER_EMPATHY_COMFORT_2", Name: "Presence of Peace", Description: "+1 Resolve nearby"}},
	},
	{
		ID: "SKILL_EMPATHY_COUNSELING", Name: "Counseling", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 3, CardDescription: "Talk-based healing",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_COUNSELING_1", Name: "Safe Space", Description: "Reduce trauma recovery TV"}, {ID: "HELPER_EMPATHY_COUNSELING_2", Name: "Guided Reflection", Description: "Remove persistent negative status"}},
	},
	{
		ID: "SKILL_EMPATHY_EMPATHIZING", Name: "Empathizing", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 4, CardDescription: "Insight into others' states",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_EMPATHIZING_1", Name: "Active Listening", Description: "Gain empathic insight"}, {ID: "HELPER_EMPATHY_EMPATHIZING_2", Name: "Background Knowledge", Description: "Reduce Insight TV"}},
	},
	{
		ID: "SKILL_EMPATHY_HEALING", Name: "Healing", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 5, CardDescription: "First aid and recovery",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_HEALING_1", Name: "Field Medic", Description: "Heal downed character"}, {ID: "HELPER_EMPATHY_HEALING_2", Name: "Triage Coordinator", Description: "Optimize group healing"}},
	},
	{
		ID: "SKILL_EMPATHY_MEDIATION", Name: "Mediation", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 6, CardDescription: "De-escalating conflict",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_MEDIATION_1", Name: "Shared Ground", Description: "Reduce social TVs"}, {ID: "HELPER_EMPATHY_MEDIATION_2", Name: "Neutral Territory", Description: "Create temporary truce"}},
	},
	{
		ID: "SKILL_EMPATHY_PSYCHOLOGY", Name: "Psychology", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 7, CardDescription: "Understanding mental states",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_PSYCHOLOGY_1", Name: "Mind Map", Description: "Name belief/fear for +1d6"}, {ID: "HELPER_EMPATHY_PSYCHOLOGY_2", Name: "Behavioral Prediction", Description: "Predict responses"}},
	},
	{
		ID: "SKILL_EMPATHY_READING_EMOTIONS", Name: "Reading Emotions", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 8, CardDescription: "Sensing emotional shifts",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_READING_EMOTIONS_1", Name: "Emotional Mirror", Description: "Copy/counter emotional status"}, {ID: "HELPER_EMPATHY_READING_EMOTIONS_2", Name: "Mood Cartographer", Description: "Read group dynamics"}},
	},
	{
		ID: "SKILL_EMPATHY_SOCIAL_WORK", Name: "Social Work", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 9, CardDescription: "Coordinating services, aid",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_SOCIAL_WORK_1", Name: "Community Link", Description: "Call in help"}, {ID: "HELPER_EMPATHY_SOCIAL_WORK_2", Name: "Resource Coordinator", Description: "Maximize aid benefit"}},
	},
	{
		ID: "SKILL_EMPATHY_THERAPY", Name: "Therapy", AttributeID: "ATTR_EMPATHY", AttributeName: "Empathy",
		SourceOrder: 10, CardDescription: "Long-term trauma care",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_EMPATHY_THERAPY_1", Name: "Breakthrough Session", Description: "Restore Heart HP"}, {ID: "HELPER_EMPATHY_THERAPY_2", Name: "Trauma Integration", Description: "Transform trauma into strength"}},
	},
	{
		ID: "SKILL_GRACE_ACROBATICS", Name: "Acrobatics", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 1, CardDescription: "Tumbling, aerial maneuvers",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_ACROBATICS_1", Name: "Tumble Assist", Description: "Reduce fall damage"}, {ID: "HELPER_GRACE_ACROBATICS_2", Name: "Momentum Redirect", Description: "Convert failed jump"}},
	},
	{
		ID: "SKILL_GRACE_AESTHETIC_DESIGN", Name: "Aesthetic Design", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 2, CardDescription: "Creation and assessment of aesthetic appeal, visual harmony, and artistic impact",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_AESTHETIC_DESIGN_1", Name: "Pitch Ideas", Description: "Brainstorm and communicate new creative concepts"}, {ID: "HELPER_GRACE_AESTHETIC_DESIGN_2", Name: "Model Ideas", Description: "Create representations for flat bonuses to design implementation"}},
	},
	{
		ID: "SKILL_GRACE_ARCHERY", Name: "Archery", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 3, CardDescription: "Precision ranged attacks",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_ARCHERY_1", Name: "Spotter Assist", Description: "Reduce range penalty"}, {ID: "HELPER_GRACE_ARCHERY_2", Name: "Volley Fire", Description: "Splash effect"}},
	},
	{
		ID: "SKILL_GRACE_BALANCE", Name: "Balance", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 4, CardDescription: "Stability on unstable terrain",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_BALANCE_1", Name: "Stabilizing Grip", Description: "Prevent ally fall"}, {ID: "HELPER_GRACE_BALANCE_2", Name: "Dynamic Counterbalance", Description: "Reroll reposition"}},
	},
	{
		ID: "SKILL_GRACE_DODGE", Name: "Dodge", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 5, CardDescription: "Evading attacks or hazards",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_DODGE_1", Name: "Evade Signal", Description: "Advantage on dodge"}, {ID: "HELPER_GRACE_DODGE_2", Name: "Cover Leap", Description: "Move ally out of danger"}},
	},
	{
		ID: "SKILL_GRACE_ETIQUETTE", Name: "Etiquette", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 6, CardDescription: "Navigating customs and decorum",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_ETIQUETTE_1", Name: "Whispered Reminder", Description: "Prevent faux pas"}, {ID: "HELPER_GRACE_ETIQUETTE_2", Name: "Cultural Translation", Description: "Reduce culture TV"}},
	},
	{
		ID: "SKILL_GRACE_PERFORMANCE", Name: "Performance", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 7, CardDescription: "Physical expression to entertain",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_PERFORMANCE_1", Name: "Support Harmony", Description: "+1d6 Performance"}, {ID: "HELPER_GRACE_PERFORMANCE_2", Name: "Crowd Sync", Description: "Direct audience reaction"}},
	},
	{
		ID: "SKILL_GRACE_SLEIGHT_OF_HAND", Name: "Sleight of Hand", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 8, CardDescription: "Concealment, manipulation",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_SLEIGHT_OF_HAND_1", Name: "Misdirection", Description: "Force reroll perception"}, {ID: "HELPER_GRACE_SLEIGHT_OF_HAND_2", Name: "Quick Exchange", Description: "Swap objects"}},
	},
	{
		ID: "SKILL_GRACE_SOCIAL_GRACE", Name: "Social Grace", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 9, CardDescription: "Charm, likability",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_SOCIAL_GRACE_1", Name: "Wing Ally", Description: "+1d6 social"}, {ID: "HELPER_GRACE_SOCIAL_GRACE_2", Name: "Compliment Redirect", Description: "Hostile to neutral"}},
	},
	{
		ID: "SKILL_GRACE_STEALTH", Name: "Stealth", AttributeID: "ATTR_GRACE", AttributeName: "Grace",
		SourceOrder: 10, CardDescription: "Avoiding detection",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_GRACE_STEALTH_1", Name: "Distraction Setup", Description: "Reduce stealth TV"}, {ID: "HELPER_GRACE_STEALTH_2", Name: "Shadow Lead", Description: "Ally follows result"}},
	},
	{
		ID: "SKILL_INTELLECT_DEBATE", Name: "Debate", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 1, CardDescription: "Structured verbal sparring",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_DEBATE_1", Name: "Opening Gambit", Description: "-2 TV defense"}, {ID: "HELPER_INTELLECT_DEBATE_2", Name: "Closing Point", Description: "+1d6 Convince"}},
	},
	{
		ID: "SKILL_INTELLECT_INNOVATION", Name: "Innovation", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 2, CardDescription: "Creating something new",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_INNOVATION_1", Name: "Brainstorm Ideas", Description: "Reroll planning dice"}, {ID: "HELPER_INTELLECT_INNOVATION_2", Name: "Needs Analysis", Description: "Lower difficulty tier"}},
	},
	{
		ID: "SKILL_INTELLECT_INVESTIGATION", Name: "Investigation", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 3, CardDescription: "Piecing together clues",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_INVESTIGATION_1", Name: "Lead Tracker", Description: "Reroll Search/Logic"}, {ID: "HELPER_INTELLECT_INVESTIGATION_2", Name: "Evidence Chain", Description: "Convert clues to dice"}},
	},
	{
		ID: "SKILL_INTELLECT_LANGUAGE", Name: "Language", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 4, CardDescription: "Understanding new tongues",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_LANGUAGE_1", Name: "Phrase Coach", Description: "Temporary fluency"}, {ID: "HELPER_INTELLECT_LANGUAGE_2", Name: "Cultural Anchor", Description: "Reduce unfamiliarity TV"}},
	},
	{
		ID: "SKILL_INTELLECT_LOGIC", Name: "Logic", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 5, CardDescription: "Identifying reasoning flaws",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_LOGIC_1", Name: "Rebuttal Anchor", Description: "Reroll Convince/Insight"}, {ID: "HELPER_INTELLECT_LOGIC_2", Name: "Contradiction Catcher", Description: "Nullify stance bonus"}},
	},
	{
		ID: "SKILL_INTELLECT_MATHEMATICS", Name: "Mathematics", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 6, CardDescription: "Abstract modeling, probability",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_MATHEMATICS_1", Name: "Countdown Sync", Description: "Act simultaneously"}, {ID: "HELPER_INTELLECT_MATHEMATICS_2", Name: "Max-Efficiency Plan", Description: "Halve resource cost"}},
	},
	{
		ID: "SKILL_INTELLECT_RESEARCH", Name: "Research", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 7, CardDescription: "Extracting archived knowledge",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_RESEARCH_1", Name: "Source Verifier", Description: "Negate misinformation"}, {ID: "HELPER_INTELLECT_RESEARCH_2", Name: "Note Network", Description: "Share findings"}},
	},
	{
		ID: "SKILL_INTELLECT_SYSTEMS_TINKERING", Name: "Systems & Tinkering", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 8, CardDescription: "Understanding complex systems",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_SYSTEMS_TINKERING_1", Name: "Power Redirect", Description: "Empower ally action"}, {ID: "HELPER_INTELLECT_SYSTEMS_TINKERING_2", Name: "Backdoor Patch", Description: "Reduce interface TV"}},
	},
	{
		ID: "SKILL_INTELLECT_TACTICS", Name: "Tactics", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 9, CardDescription: "Predicting battlefield flow",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_TACTICS_1", Name: "Command Relay", Description: "Reposition ally"}, {ID: "HELPER_INTELLECT_TACTICS_2", Name: "Focus Fire Plan", Description: "Reroll lowest damage"}},
	},
	{
		ID: "SKILL_INTELLECT_TEACHING", Name: "Teaching", AttributeID: "ATTR_INTELLECT", AttributeName: "Intellect",
		SourceOrder: 10, CardDescription: "Imparting knowledge",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_INTELLECT_TEACHING_1", Name: "Skill Share", Description: "Ally uses your dice tier"}, {ID: "HELPER_INTELLECT_TEACHING_2", Name: "Lesson in Motion", Description: "Temporary stance action"}},
	},
	{
		ID: "SKILL_LORE_ANTHROPOLOGY", Name: "Anthropology", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 1, CardDescription: "Study of cultures and rites",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_ANTHROPOLOGY_1", Name: "Rite of Welcome", Description: "Ignore unfamiliarity TV"}, {ID: "HELPER_LORE_ANTHROPOLOGY_2", Name: "Social Architecture", Description: "Reduce TV by 2"}},
	},
	{
		ID: "SKILL_LORE_DOCUMENTATION", Name: "Documentation", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 2, CardDescription: "Recording, legal writing",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_DOCUMENTATION_1", Name: "Legal Precision", Description: "Reduce confusion TV"}, {ID: "HELPER_LORE_DOCUMENTATION_2", Name: "Institutional Memory", Description: "Access archives"}},
	},
	{
		ID: "SKILL_LORE_GEOGRAPHY", Name: "Geography", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 3, CardDescription: "Regional and terrain knowledge",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_GEOGRAPHY_1", Name: "Trail Advantage", Description: "Extra travel zone"}, {ID: "HELPER_LORE_GEOGRAPHY_2", Name: "Terrain Reading", Description: "+1d4 terrain actions"}},
	},
	{
		ID: "SKILL_LORE_HISTORY", Name: "History", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 4, CardDescription: "Understanding past events",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_HISTORY_1", Name: "Echo of the Past", Description: "+1d6 precedent"}, {ID: "HELPER_LORE_HISTORY_2", Name: "Pattern Convergence", Description: "Predict patterns"}},
	},
	{
		ID: "SKILL_LORE_MEDICINE", Name: "Medicine", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 5, CardDescription: "Biological systems and treatment",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_MEDICINE_1", Name: "Steady Hands", Description: "Ignore stress penalty"}, {ID: "HELPER_LORE_MEDICINE_2", Name: "Diagnostic Insight", Description: "Double healing"}},
	},
	{
		ID: "SKILL_LORE_MYTHOLOGY", Name: "Mythology", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 6, CardDescription: "Deific narratives",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_MYTHOLOGY_1", Name: "Living Legend", Description: "+1d4 invoking myth"}, {ID: "HELPER_LORE_MYTHOLOGY_2", Name: "Archetypal Invocation", Description: "Channel archetype"}},
	},
	{
		ID: "SKILL_LORE_NATURAL_LORE", Name: "Natural Lore", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 7, CardDescription: "Knowledge of natural world",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_NATURAL_LORE_1", Name: "Botany", Description: "+1d6 plants"}, {ID: "HELPER_LORE_NATURAL_LORE_2", Name: "Zoology", Description: "+1d6 animals"}},
	},
	{
		ID: "SKILL_LORE_RHETORIC", Name: "Rhetoric", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 8, CardDescription: "Structured argument",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_RHETORIC_1", Name: "Quoting You", Description: "Reroll using ally statement"}, {ID: "HELPER_LORE_RHETORIC_2", Name: "Discourse Control", Description: "Control framing"}},
	},
	{
		ID: "SKILL_LORE_SOCIOLOGY", Name: "Sociology", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 9, CardDescription: "Large-scale people systems",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_SOCIOLOGY_1", Name: "Economics", Description: "Resource distribution"}, {ID: "HELPER_LORE_SOCIOLOGY_2", Name: "Power Systems", Description: "Power dynamics"}},
	},
	{
		ID: "SKILL_LORE_SUPERNATURAL_LORE", Name: "Supernatural Lore", AttributeID: "ATTR_LORE", AttributeName: "Lore",
		SourceOrder: 10, CardDescription: "Forbidden knowledge",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_LORE_SUPERNATURAL_LORE_1", Name: "Protective Ink", Description: "Mental resistance"}, {ID: "HELPER_LORE_SUPERNATURAL_LORE_2", Name: "Forbidden Synthesis", Description: "Combine sources"}},
	},
	{
		ID: "SKILL_MIGHT_ATHLETICS", Name: "Athletics", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 1, CardDescription: "Running, jumping, climbing",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_ATHLETICS_1", Name: "Spotter", Description: "Reroll or reduce fall severity"}, {ID: "HELPER_MIGHT_ATHLETICS_2", Name: "Assist Leap", Description: "Throw/vault ally"}},
	},
	{
		ID: "SKILL_MIGHT_BREAKING_FORCING", Name: "Breaking / Forcing", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 2, CardDescription: "Overpowering barriers",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_BREAKING_FORCING_1", Name: "Tool Leverage", Description: "Reduce TV with tools"}, {ID: "HELPER_MIGHT_BREAKING_FORCING_2", Name: "Weak Point Callout", Description: "Lower object TV"}},
	},
	{
		ID: "SKILL_MIGHT_CLIMBING", Name: "Climbing", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 3, CardDescription: "Vertical movement",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_CLIMBING_1", Name: "Belayer", Description: "Prevent fall damage"}, {ID: "HELPER_MIGHT_CLIMBING_2", Name: "Anchor Setup", Description: "Reduce climbing TV"}},
	},
	{
		ID: "SKILL_MIGHT_ENDURANCE", Name: "Endurance", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 4, CardDescription: "Sustained effort",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_ENDURANCE_1", Name: "Pace Partner", Description: "Share endurance result"}, {ID: "HELPER_MIGHT_ENDURANCE_2", Name: "Ration Tracker", Description: "Supplies last longer"}},
	},
	{
		ID: "SKILL_MIGHT_GRAPPLING", Name: "Grappling", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 5, CardDescription: "Wrestling, pinning",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_GRAPPLING_1", Name: "Assist Lock", Description: "Advantage on pin"}, {ID: "HELPER_MIGHT_GRAPPLING_2", Name: "Leverage Shift", Description: "Positional advantage"}},
	},
	{
		ID: "SKILL_MIGHT_HEAVY_LIFTING", Name: "Heavy Lifting", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 6, CardDescription: "Lifting, bracing heavy objects",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_HEAVY_LIFTING_1", Name: "Weight Distribution", Description: "Reduce exhaustion"}, {ID: "HELPER_MIGHT_HEAVY_LIFTING_2", Name: "Structural Bracer", Description: "Prevent collapse"}},
	},
	{
		ID: "SKILL_MIGHT_INTIMIDATION_PHYSICAL", Name: "Intimidation (Physical)", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 7, CardDescription: "Using force or presence",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_INTIMIDATION_PHYSICAL_1", Name: "Demonstration", Description: "Prove strength"}, {ID: "HELPER_MIGHT_INTIMIDATION_PHYSICAL_2", Name: "Pack Pressure", Description: "Group intimidation"}},
	},
	{
		ID: "SKILL_MIGHT_MELEE_COMBAT", Name: "Melee Combat", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 8, CardDescription: "Close-quarters fighting",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_MELEE_COMBAT_1", Name: "Flanking", Description: "Advantage on attack"}, {ID: "HELPER_MIGHT_MELEE_COMBAT_2", Name: "Weapon Specialist", Description: "Reroll damage 1s"}},
	},
	{
		ID: "SKILL_MIGHT_PAIN_TOLERANCE", Name: "Pain Tolerance", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 9, CardDescription: "Functioning while injured",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_PAIN_TOLERANCE_1", Name: "Stoic Presence", Description: "Resist fear/pain"}, {ID: "HELPER_MIGHT_PAIN_TOLERANCE_2", Name: "Ignore Injury", Description: "Avoid wound once/scene"}},
	},
	{
		ID: "SKILL_MIGHT_SWIMMING", Name: "Swimming", AttributeID: "ATTR_MIGHT", AttributeName: "Might",
		SourceOrder: 10, CardDescription: "Movement through water",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_MIGHT_SWIMMING_1", Name: "Lifeline Throw", Description: "Assist ally"}, {ID: "HELPER_MIGHT_SWIMMING_2", Name: "Form Sync", Description: "Swim in formation"}},
	},
	{
		ID: "SKILL_PRESENCE_AUTHORITY", Name: "Authority", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 1, CardDescription: "Institutional legitimacy",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_AUTHORITY_1", Name: "By the Book", Description: "Legal bonus"}, {ID: "HELPER_PRESENCE_AUTHORITY_2", Name: "Invoke Order", Description: "Grant cover"}},
	},
	{
		ID: "SKILL_PRESENCE_COMMAND", Name: "Command", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 2, CardDescription: "Directing others",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_COMMAND_1", Name: "Form Up", Description: "Rearrange allies"}, {ID: "HELPER_PRESENCE_COMMAND_2", Name: "Priority Call", Description: "Interrupt initiative"}},
	},
	{
		ID: "SKILL_PRESENCE_DIPLOMACY", Name: "Diplomacy", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 3, CardDescription: "Negotiation and peace",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_DIPLOMACY_1", Name: "Terms Setter", Description: "Reduce negotiation TV"}, {ID: "HELPER_PRESENCE_DIPLOMACY_2", Name: "Buffer Line", Description: "Buy time"}},
	},
	{
		ID: "SKILL_PRESENCE_INSPIRE", Name: "Inspire", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 4, CardDescription: "Instilling morale",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_INSPIRE_1", Name: "Shared Spark", Description: "Extend bonus"}, {ID: "HELPER_PRESENCE_INSPIRE_2", Name: "Echo Call", Description: "Reinforce moment"}},
	},
	{
		ID: "SKILL_PRESENCE_INTIMIDATION", Name: "Intimidation", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 5, CardDescription: "Applying fear",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_INTIMIDATION_1", Name: "Glare Down", Description: "Cancel stance"}, {ID: "HELPER_PRESENCE_INTIMIDATION_2", Name: "Power Display", Description: "Temporary advantage"}},
	},
	{
		ID: "SKILL_PRESENCE_LEADERSHIP", Name: "Leadership", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 6, CardDescription: "Long-term influence",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_LEADERSHIP_1", Name: "Doctrine Builder", Description: "Team behaviors"}, {ID: "HELPER_PRESENCE_LEADERSHIP_2", Name: "Fallback Coordinator", Description: "Defensive stance"}},
	},
	{
		ID: "SKILL_PRESENCE_PRESENCE", Name: "Presence", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 7, CardDescription: "Overwhelming aura",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_PRESENCE_1", Name: "Step Inward", Description: "Silence aggression"}, {ID: "HELPER_PRESENCE_PRESENCE_2", Name: "Emotional Anchor", Description: "Resist fear"}},
	},
	{
		ID: "SKILL_PRESENCE_REPUTATION", Name: "Reputation", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 8, CardDescription: "Using fame or infamy",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_REPUTATION_1", Name: "Name Leverage", Description: "Force reroll defense"}, {ID: "HELPER_PRESENCE_REPUTATION_2", Name: "Persona Mask", Description: "Substitute identity"}},
	},
	{
		ID: "SKILL_PRESENCE_RESOLUTE", Name: "Resolute", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 9, CardDescription: "Holding steady",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_RESOLUTE_1", Name: "Hold Line", Description: "Prevent loss"}, {ID: "HELPER_PRESENCE_RESOLUTE_2", Name: "Echo Core", Description: "Restore stance tier"}},
	},
	{
		ID: "SKILL_PRESENCE_SPEECHCRAFT", Name: "Speechcraft", AttributeID: "ATTR_PRESENCE", AttributeName: "Presence",
		SourceOrder: 10, CardDescription: "Persuasive oratory",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_PRESENCE_SPEECHCRAFT_1", Name: "Flourish", Description: "Partial success"}, {ID: "HELPER_PRESENCE_SPEECHCRAFT_2", Name: "Echo Rhetoric", Description: "Reroll Convince"}},
	},
	{
		ID: "SKILL_RESOLVE_DISCIPLINE", Name: "Discipline", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 1, CardDescription: "Maintaining routines",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_DISCIPLINE_1", Name: "Daily Ritual", Description: "Bonus die"}, {ID: "HELPER_RESOLVE_DISCIPLINE_2", Name: "Discipline Network", Description: "Group bonus"}},
	},
	{
		ID: "SKILL_RESOLVE_FORTIFICATION", Name: "Fortification", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 2, CardDescription: "Mental preparation",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_FORTIFICATION_1", Name: "Mental Armor", Description: "Ignore status"}, {ID: "HELPER_RESOLVE_FORTIFICATION_2", Name: "Preparation Protocol", Description: "Initiative bonus"}},
	},
	{
		ID: "SKILL_RESOLVE_GRIEF_PROCESSING", Name: "Grief Processing", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 3, CardDescription: "Coping with loss",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_GRIEF_PROCESSING_1", Name: "Remember Together", Description: "Heal allies"}, {ID: "HELPER_RESOLVE_GRIEF_PROCESSING_2", Name: "Memorial Strength", Description: "Transform grief"}},
	},
	{
		ID: "SKILL_RESOLVE_HABIT_FORMATION", Name: "Habit Formation", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 4, CardDescription: "Creating routines",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_HABIT_FORMATION_1", Name: "Routine Anchor", Description: "+1d4 habit roll"}, {ID: "HELPER_RESOLVE_HABIT_FORMATION_2", Name: "Behavioral Architecture", Description: "Design system"}},
	},
	{
		ID: "SKILL_RESOLVE_LONG_TERM_PLANNING", Name: "Long-Term Planning", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 5, CardDescription: "Strategic foresight",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_LONG_TERM_PLANNING_1", Name: "Vision Anchor", Description: "+1 coordination"}, {ID: "HELPER_RESOLVE_LONG_TERM_PLANNING_2", Name: "Strategic Reserve", Description: "Auto-succeed plan"}},
	},
	{
		ID: "SKILL_RESOLVE_MENTAL_FORTITUDE", Name: "Mental Fortitude", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 6, CardDescription: "Resisting mental effects",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_MENTAL_FORTITUDE_1", Name: "Iron Core", Description: "Ignore status"}, {ID: "HELPER_RESOLVE_MENTAL_FORTITUDE_2", Name: "Unbreakable Will", Description: "+1d6 resist"}},
	},
	{
		ID: "SKILL_RESOLVE_PERSISTENCE", Name: "Persistence", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 7, CardDescription: "Continuing effort",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_PERSISTENCE_1", Name: "Stay the Course", Description: "Reroll project step"}, {ID: "HELPER_RESOLVE_PERSISTENCE_2", Name: "Momentum Builder", Description: "+1d4 project rolls"}},
	},
	{
		ID: "SKILL_RESOLVE_PROJECT_COMPLETION", Name: "Project Completion", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 8, CardDescription: "Ensuring completion",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_PROJECT_COMPLETION_1", Name: "Closing Push", Description: "Partial to full success"}, {ID: "HELPER_RESOLVE_PROJECT_COMPLETION_2", Name: "Completion Catalyst", Description: "Halve final time"}},
	},
	{
		ID: "SKILL_RESOLVE_RECOVERY", Name: "Recovery", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 9, CardDescription: "Healing from trauma",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_RECOVERY_1", Name: "Second Wind", Description: "Regain HP"}, {ID: "HELPER_RESOLVE_RECOVERY_2", Name: "Resilience Training", Description: "Improve recovery"}},
	},
	{
		ID: "SKILL_RESOLVE_STRESS_MANAGEMENT", Name: "Stress Management", AttributeID: "ATTR_RESOLVE", AttributeName: "Resolve",
		SourceOrder: 10, CardDescription: "Coping with pressure",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_RESOLVE_STRESS_MANAGEMENT_1", Name: "Regulation Break", Description: "Cancel status"}, {ID: "HELPER_RESOLVE_STRESS_MANAGEMENT_2", Name: "Pressure Valve", Description: "Reduce complexity"}},
	},
	{
		ID: "SKILL_SPIRIT_CHANNELING", Name: "Channeling", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 1, CardDescription: "Drawing unseen energy",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_CHANNELING_1", Name: "Spirit Anchor", Description: "Resist mind effects"}, {ID: "HELPER_SPIRIT_CHANNELING_2", Name: "Energy Transfer", Description: "Shift cost"}},
	},
	{
		ID: "SKILL_SPIRIT_FAITH", Name: "Faith", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 2, CardDescription: "Holding belief",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_FAITH_1", Name: "Affirm Belief", Description: "Remove Haunted/Fragmented"}, {ID: "HELPER_SPIRIT_FAITH_2", Name: "Sacred Insight", Description: "Moral framework"}},
	},
	{
		ID: "SKILL_SPIRIT_INVOCATION", Name: "Invocation", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 3, CardDescription: "Calling greater forces",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_INVOCATION_1", Name: "Group Petition", Description: "Reroll Invocation"}, {ID: "HELPER_SPIRIT_INVOCATION_2", Name: "Sanctify Ground", Description: "Location boon"}},
	},
	{
		ID: "SKILL_SPIRIT_MEDITATION", Name: "Meditation", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 4, CardDescription: "Quieting the mind",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_MEDITATION_1", Name: "Breathe Together", Description: "Share bonus"}, {ID: "HELPER_SPIRIT_MEDITATION_2", Name: "Focus Chant", Description: "Delay collapse"}},
	},
	{
		ID: "SKILL_SPIRIT_PURIFICATION", Name: "Purification", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 5, CardDescription: "Cleansing corruption",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_PURIFICATION_1", Name: "Joint Cleanse", Description: "Heal multiple"}, {ID: "HELPER_SPIRIT_PURIFICATION_2", Name: "Echo Purge", Description: "Reroll in sacred space"}},
	},
	{
		ID: "SKILL_SPIRIT_RITUAL", Name: "Ritual", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 6, CardDescription: "Coordinated magical acts",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_RITUAL_1", Name: "Circle Formation", Description: "+1 die per participant"}, {ID: "HELPER_SPIRIT_RITUAL_2", Name: "Power Reservoir", Description: "Store magic"}},
	},
	{
		ID: "SKILL_SPIRIT_SOUL_READING", Name: "Soul Reading", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 7, CardDescription: "Perceiving core nature",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_SOUL_READING_1", Name: "Mirror Soul", Description: "Reroll vs despair"}, {ID: "HELPER_SPIRIT_SOUL_READING_2", Name: "Truth Manifest", Description: "Bypass deception"}},
	},
	{
		ID: "SKILL_SPIRIT_SPIRIT_SENSE", Name: "Spirit Sense", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 8, CardDescription: "Detecting unseen presences",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_SPIRIT_SENSE_1", Name: "Pulse Alert", Description: "Warn allies"}, {ID: "HELPER_SPIRIT_SPIRIT_SENSE_2", Name: "Tag Aura", Description: "Mark entity"}},
	},
	{
		ID: "SKILL_SPIRIT_SPIRITUAL_INSIGHT", Name: "Spiritual Insight", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 9, CardDescription: "Interpreting omens",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_SPIRITUAL_INSIGHT_1", Name: "Dream Echo", Description: "+1d6 prediction"}, {ID: "HELPER_SPIRIT_SPIRITUAL_INSIGHT_2", Name: "Fate Twist", Description: "Symbolic substitution"}},
	},
	{
		ID: "SKILL_SPIRIT_WARDING", Name: "Warding", AttributeID: "ATTR_SPIRIT", AttributeName: "Spirit",
		SourceOrder: 10, CardDescription: "Protective barriers",
		Helpers: []Chapter4SkillHelper{{ID: "HELPER_SPIRIT_WARDING_1", Name: "Ritual Anchor", Description: "Stabilize magic"}, {ID: "HELPER_SPIRIT_WARDING_2", Name: "Reactive Barrier", Description: "Interrupt damage"}},
	},
}

// Chapter4SkillByID looks up a skill by its stable ID.
func Chapter4SkillByID(id string) (Chapter4Skill, bool) {
	for _, s := range Chapter4Skills {
		if s.ID == id {
			return s, true
		}
	}
	return Chapter4Skill{}, false
}

// Chapter4SkillByName looks up a skill by its exact canonical display name.
func Chapter4SkillByName(name string) (Chapter4Skill, bool) {
	for _, s := range Chapter4Skills {
		if s.Name == name {
			return s, true
		}
	}
	return Chapter4Skill{}, false
}

// Chapter4SkillsForAttribute returns the (up to) 10 skills belonging to the
// given attribute ID, sorted by source_order.
func Chapter4SkillsForAttribute(attributeID string) []Chapter4Skill {
	out := make([]Chapter4Skill, 0, 10)
	for _, s := range Chapter4Skills {
		if s.AttributeID == attributeID {
			out = append(out, s)
		}
	}
	return out
}

// Chapter4AttributeByName looks up an attribute by its canonical display name
// (e.g. "Craft").
func Chapter4AttributeByName(name string) (Chapter4Attribute, bool) {
	for _, a := range Chapter4Attributes {
		if a.Name == name {
			return a, true
		}
	}
	return Chapter4Attribute{}, false
}

// ValidateChapter4SkillCatalogue checks catalogue integrity: exactly 10
// attributes, exactly 10 skills per attribute, exactly 100 total, no
// duplicate stable IDs, no duplicate canonical names, gap-free source_order
// 1-10 within each attribute.
func ValidateChapter4SkillCatalogue() error {
	if len(Chapter4Attributes) != 10 {
		return fmt.Errorf("chapter4: expected 10 attributes, got %d", len(Chapter4Attributes))
	}
	if len(Chapter4Skills) != 100 {
		return fmt.Errorf("chapter4: expected 100 skills, got %d", len(Chapter4Skills))
	}

	seenIDs := map[string]bool{}
	seenNames := map[string]bool{}
	countByAttr := map[string]int{}
	orderSeenByAttr := map[string]map[int]bool{}
	for _, s := range Chapter4Skills {
		if seenIDs[s.ID] {
			return fmt.Errorf("chapter4: duplicate skill stable ID %q", s.ID)
		}
		seenIDs[s.ID] = true
		if seenNames[s.Name] {
			return fmt.Errorf("chapter4: duplicate canonical skill name %q", s.Name)
		}
		seenNames[s.Name] = true
		countByAttr[s.AttributeID]++
		if orderSeenByAttr[s.AttributeID] == nil {
			orderSeenByAttr[s.AttributeID] = map[int]bool{}
		}
		if orderSeenByAttr[s.AttributeID][s.SourceOrder] {
			return fmt.Errorf("chapter4: duplicate source_order %d within attribute %q", s.SourceOrder, s.AttributeID)
		}
		orderSeenByAttr[s.AttributeID][s.SourceOrder] = true
	}

	for _, a := range Chapter4Attributes {
		if countByAttr[a.ID] != 10 {
			return fmt.Errorf("chapter4: attribute %q has %d skills, want 10", a.ID, countByAttr[a.ID])
		}
		for order := 1; order <= 10; order++ {
			if !orderSeenByAttr[a.ID][order] {
				return fmt.Errorf("chapter4: attribute %q missing source_order %d", a.ID, order)
			}
		}
	}

	// Every sample-archetype key skill must resolve to exactly one skill, and
	// (per the routing decision) that skill's governing attribute is what
	// Chapter 4 uses to resolve the ten-card group -- so this is always true
	// by construction, but we still verify the key skill name resolves.
	for _, archetype := range Chapter3Archetypes {
		if _, ok := Chapter4SkillByName(archetype.KeySkill); !ok {
			return fmt.Errorf("chapter4: archetype %q key skill %q does not resolve to any catalogue skill", archetype.Key, archetype.KeySkill)
		}
	}

	return nil
}
