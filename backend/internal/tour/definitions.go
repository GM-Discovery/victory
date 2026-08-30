package tour

// Step is one beat of a tour: one practical question answered ("What is
// this? Why do I care? What do I click?" -- kernel-91 S49), never
// executable script. Target is the semantic reference the frontend engine
// resolves to a live DOM element (kernel-91 S32) -- nothing DOM-specific
// lives here.
type Step struct {
	Key    string `json:"key"`
	Target string `json:"target"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	// ActionRequired marks a click-gated step (kernel-91 S7): the frontend
	// engine only advances past it on a real click of the resolved target,
	// never automatically.
	ActionRequired bool `json:"action_required"`
}

// Definition is one authored tour. RequiredRoles is empty for tours gated
// only by venue access (e.g. the campus tours); when non-empty it lists
// location_role values and/or the "operator" sentinel (Operator sits
// outside the location_role enum -- see eligibility.go).
type Definition struct {
	Key           string   `json:"key"`
	Mandatory     bool     `json:"mandatory"`
	VenueSlug     string   `json:"venue_slug,omitempty"`
	RequiredRoles []string `json:"required_roles,omitempty"`
	Steps         []Step   `json:"steps"`
}

// Definitions is the full authored tour set for this pass (kernel-91's
// confirmed scope: mandatory campus orientation, the skippable campus
// continuation, the Catharsis Cast venue tour, and the Director's Chair
// role-overlay tour -- the S42-43 acceptance proof). Producer/Crew/Operator/
// Audience get eligibility plumbing (RequiredRoles values are already valid
// location_role/operator values) but no authored entry yet; adding one is a
// migration (tour_key CHECK) plus a new map entry here, matching
// tutorial.allMilestones' friction-by-design.
var Definitions = map[string]Definition{
	KeyCampusMandatory: {
		Key:       KeyCampusMandatory,
		Mandatory: true,
		Steps: []Step{
			{
				Key:            "audition-hall-pin",
				Target:         "venue:audition-hall",
				Title:          "The Audition Hall",
				Body:           "The Audition Hall is your gateway to the rest of campus. Start there whenever you want to see where else Victory can take you.",
				ActionRequired: true,
			},
			{
				Key:            "trailer-pin",
				Target:         "venue:trailers",
				Title:          "Your Trailer",
				Body:           "Your Trailer is your own space. Profile and personal presentation live there.",
				ActionRequired: true,
			},
		},
	},
	KeyCampusContinuation: {
		Key:       KeyCampusContinuation,
		Mandatory: false,
		Steps: []Step{
			{
				Key:    "catharsis-pin",
				Target: "venue:catharsis",
				Title:  "Catharsis",
				Body:   "Catharsis is where live play happens -- character build and the stage itself. This is where a Show is actually run.",
			},
		},
	},
	// Kernel 93 Pass C: split out of campus_continuation (see migration 112).
	// The Greenroom is only actually relevant once a Character workbook
	// exists to go look at -- eligibility.go gates this on an in-progress
	// character_cards row at Catharsis, not merely on having seen the
	// Catharsis pin.
	KeyGreenroomIntro: {
		Key:       KeyGreenroomIntro,
		Mandatory: false,
		Steps: []Step{
			{
				Key:    "greenroom-pin",
				Target: "venue:greenroom",
				Title:  "The Greenroom",
				Body:   "The Greenroom is preparation, not performance -- your Character workbooks live here, ready before you ever take the stage.",
			},
		},
	},
	KeyCatharsisCast: {
		Key:           KeyCatharsisCast,
		Mandatory:     false,
		VenueSlug:     "catharsis",
		RequiredRoles: []string{"cast"},
		Steps: []Step{
			{
				Key:    "character-tray",
				Target: "control:catharsis-character-tray",
				Title:  "Your Character",
				Body:   "This is where your Character's workbook lives -- open it any time to check who you are here.",
			},
			{
				Key:    "socio-hud",
				Target: "control:catharsis-socio-hud",
				Title:  "Game Status",
				Body:   "Fate, Stance, and your current standing live here. Keep an eye on this panel during a scene.",
			},
			{
				Key:    "dice-tray",
				Target: "control:catharsis-dice-tray",
				Title:  "Rolling",
				Body:   "When a roll is called for, this is where you make it.",
			},
		},
	},
	KeyDirectorsChairDirectorToolbox: {
		Key:           KeyDirectorsChairDirectorToolbox,
		Mandatory:     false,
		VenueSlug:     "directors-chair",
		RequiredRoles: []string{"director", "producer", "operator"},
		Steps: []Step{
			{
				Key:    "director-console",
				Target: "control:director-console",
				Title:  "The Director Console",
				Body:   "You're directing now. These controls belong to you -- start and close a Showing, and manage what the stage reveals.",
			},
			{
				Key:    "director-queue",
				Target: "control:director-queue",
				Title:  "The Director Queue",
				Body:   "Incoming access requests land here. Approve or deny who joins below you.",
			},
			{
				Key:    "invite-below-you",
				Target: "control:invite-below-you",
				Title:  "Invite Below You",
				Body:   "Grant Cast, Crew, or Audience access directly from here -- no separate invite tool needed.",
			},
		},
	},
}
