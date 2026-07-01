package characters

import "fmt"

// Chapter3ArchetypeDatasetVersion identifies the canonical archetype/quiz
// dataset this catalog was ported from (script.js, validated against the
// kernel spec's 14 archetypes / 15 questions / 89 answers counts).
const Chapter3ArchetypeDatasetVersion = "1.0.0"

const Chapter3RulesetVersion = "1.1"

// Chapter3ArchetypePrimary holds the longer-form narrative text shown on the
// quiz result page for the player's primary archetype.
type Chapter3ArchetypePrimary struct {
	Intro      string `json:"intro"`
	Strengths  string `json:"strengths"`
	Challenges string `json:"challenges"`
	Socio      string `json:"socio"`
}

// Chapter3Archetype is one entry in the canonical 14-archetype catalog.
type Chapter3Archetype struct {
	ID                 string                   `json:"id"`
	Key                string                   `json:"key"`
	Title              string                   `json:"title"`
	Code               string                   `json:"code"`
	DisplayOrder       int                      `json:"display_order"`
	ShortDescription   string                   `json:"short_description"`
	Echo               string                   `json:"echo"`
	Primary            Chapter3ArchetypePrimary `json:"primary"`
	HEXACO             string                   `json:"hexaco"`
	HEXACOPlain        string                   `json:"hexaco_plain"`
	CoreDrives         []string                 `json:"core_drives"`
	PrimaryAttribute   string                   `json:"primary_attribute"`
	SecondaryAttribute string                   `json:"secondary_attribute"`
	KeySkill           string                   `json:"key_skill"`
	Motto              string                   `json:"motto"`
	Notices            string                   `json:"notices"`
	StrengthsList      []string                 `json:"strengths_list"`
	GrowthEdges        []string                 `json:"growth_edges"`
	GroupRole          string                   `json:"group_role"`
	UnderStress        string                   `json:"under_stress"`
	PlaySuggestions    []string                 `json:"play_suggestions"`
	QuestionsToAsk     []string                 `json:"questions_to_ask"`
	HowToHelpYourself  []string                 `json:"how_to_help_yourself"`
}

// Chapter3Archetypes is the complete versioned archetype catalog, ported
// verbatim (profile text, attributes, key skills) from the canonical
// reference implementation.
var Chapter3Archetypes = []Chapter3Archetype{
	{
		ID: "ARCHETYPE_AESTHETICIAN", Key: "Aesthetician", Title: "The Aesthetician", Code: "AES", DisplayOrder: 1,
		ShortDescription: "Possesses a keen eye for beauty and detail. Their style and craftsmanship can be overlooked, while their pursuit of perfection can lead to anxiety and self-imposed burdens.",
		Echo:             "You appreciate beauty, craftsmanship, and the details that make something memorable. You're often drawn to things that feel meaningful, elegant, or deeply human.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You notice beauty where other people see ordinary things. Sometimes it's found in art, music, or nature. Sometimes it's found in a well-made tool, a meaningful tradition, or a person quietly doing the right thing.",
			Strengths:  "Aestheticians help people appreciate what matters. They often recognize quality, meaning, and craftsmanship long before anyone else does. Their attention to beauty can make experiences richer, environments more welcoming, and ideas easier to remember.",
			Challenges: "Beauty is difficult to measure and impossible to finish. Aestheticians can become frustrated when others overlook details that seem important to them, and their pursuit of excellence can sometimes become perfectionism. They may spend so much time refining something that they struggle to call it complete.",
			Socio:      "In Socio-, Aestheticians often become artists, designers, musicians, storytellers, artisans, and curators of meaningful experiences. They are at their best when helping people see the value, beauty, and humanity that already exists around them.",
		},
		HEXACO:           "High Openness, Moderate Conscientiousness, Moderate Emotionality",
		HEXACOPlain:      "You tend to notice resonance, beauty, craft, and the details that make something feel meaningful.",
		CoreDrives:       []string{"Beauty", "Harmony", "Resonance"},
		PrimaryAttribute: "Craft", SecondaryAttribute: "Grace", KeySkill: "Aesthetic Design",
		Motto:             "You often find meaning in places other people dismiss as ordinary.",
		Notices:           "You notice beauty, symbolism, craftsmanship, atmosphere, sacrifice, and the small details that give something significance.",
		StrengthsList:     []string{"Recognizing meaning", "Attention to detail", "Creative expression", "Helping others appreciate what matters"},
		GrowthEdges:       []string{"Accepting imperfection", "Finishing before refining", "Balancing ideals with reality", "Allowing others to value different things"},
		GroupRole:         "In a group, you are often the person reminding everyone why something matters, not just what it accomplishes.",
		UnderStress:       "When stressed, you may become overly critical, discouraged by compromises, or feel disconnected from things that seem hollow or purely functional.",
		PlaySuggestions:   []string{"Artist", "Designer", "Storyteller", "Cultural Keeper"},
		QuestionsToAsk:    []string{"What gives this meaning?", "What beauty is being overlooked?", "What deserves greater care?"},
		HowToHelpYourself: []string{"Not everything meaningful must be perfect.", "Share your work before it feels finished.", "Beauty often survives imperfections."},
	},
	{
		ID: "ARCHETYPE_CATALYST", Key: "Catalyst", Title: "The Catalyst", Code: "CAT", DisplayOrder: 2,
		ShortDescription: "Creates momentum, energy, and action. They bring people together and make things happen, but can experience significant emotional crashes beneath their enthusiasm.",
		Echo:             "You bring energy and momentum wherever you go. When a group becomes stuck, you're often one of the first people ready to get things moving again.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You don't like watching good ideas sit on a shelf. When something needs to happen, your instinct is often to gather people, create momentum, and get things moving.",
			Strengths:  "Catalysts generate action. They help people overcome hesitation, uncertainty, and inertia. Groups often benefit from their energy because they remind everyone that progress usually starts with simply taking the first step.",
			Challenges: "Living as a source of momentum can be exhausting. Catalysts sometimes push themselves harder than they realize, especially when they feel responsible for keeping everyone else motivated. They may also become frustrated when others move more slowly than they do.",
			Socio:      "In Socio-, Catalysts often become organizers, community builders, movement leaders, event coordinators, explorers, and pioneers. They excel at helping people move from intention to action.",
		},
		HEXACO:           "High Extraversion, High Openness, Low Conscientiousness",
		HEXACOPlain:      "You are drawn toward movement, novelty, and energy more than fixed routines or slow approval processes.",
		CoreDrives:       []string{"Change", "Energy", "Movement"},
		PrimaryAttribute: "Presence", SecondaryAttribute: "Might", KeySkill: "Inspire",
		Motto:             "You get restless when everyone agrees something should happen and nobody is doing it.",
		Notices:           "You notice momentum. You notice when people are excited, discouraged, stuck, or waiting for someone else to go first.",
		StrengthsList:     []string{"Getting things started", "Bringing energy into a room", "Helping people commit", "Moving from idea to action"},
		GrowthEdges:       []string{"Finishing what you start", "Slowing down long enough to think", "Letting other people set the pace", "Avoiding burnout"},
		GroupRole:         "In a group, you are often the one who gets everyone moving when the conversation starts going in circles.",
		UnderStress:       "When stressed, you can become impatient, impulsive, or frustrated by people who need more time than you do.",
		PlaySuggestions:   []string{"Organizer", "Expedition Leader", "Movement Starter", "Entrepreneur"},
		QuestionsToAsk:    []string{"What am I waiting for someone else to start?", "Where am I creating movement?", "Am I moving toward something or merely away from something?"},
		HowToHelpYourself: []string{"Finish one thing before starting two new ones.", "Build recovery time into your plans.", "Not every problem requires immediate action."},
	},
	{
		ID: "ARCHETYPE_ARCHITECT", Key: "Architect", Title: "The Architect", Code: "ARC", DisplayOrder: 3,
		ShortDescription: "Builds systems, writes laws, and maps meaning. They can get lost in perfection or detached from reality, but when they share their vision, entire cultures shift.",
		Echo:             "You think in systems, structures, and long-term design. You're interested not only in what works, but in why it works and how it fits into a larger whole.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You rarely stop at asking whether something works. You want to understand how all the pieces fit together and whether the design itself makes sense.",
			Strengths:  "Architects see connections that other people overlook. They naturally think about systems, incentives, rules, and patterns. When they encounter a problem, they often look deeper than the immediate issue and try to redesign the structure underneath it.",
			Challenges: "The same ability that helps Architects see the bigger picture can sometimes pull them away from reality. It's easy to become fascinated by a perfect design that never gets built, or to spend so much time refining a vision that action never begins.",
			Socio:      "In Socio-, Architects often become lawmakers, philosophers, planners, reformers, institution builders, and visionaries. They are most powerful when they help others understand the structure beneath the surface.",
		},
		HEXACO:           "High Intellect, Moderate Openness, Moderate Conscientiousness",
		HEXACOPlain:      "You tend to look for the design beneath the situation: why it works, why it fails, and how the pieces fit.",
		CoreDrives:       []string{"Design", "Systems", "Intentionality"},
		PrimaryAttribute: "Lore", SecondaryAttribute: "Intellect", KeySkill: "Design",
		Motto:             "You rarely stop at asking whether something works. You want to know why it works.",
		Notices:           "You notice systems, assumptions, incentives, and the invisible structures shaping what people do.",
		StrengthsList:     []string{"System design", "Long-term thinking", "Seeing connections", "Creating structure"},
		GrowthEdges:       []string{"Accepting messy reality", "Explaining ideas simply", "Moving from planning to action", "Avoiding perfectionism"},
		GroupRole:         "In a group, you are often the person asking how the whole thing fits together.",
		UnderStress:       "When stressed, you can disappear into planning, redesigning, or trying to solve problems that do not actually need solving yet.",
		PlaySuggestions:   []string{"Lawgiver", "Scholar", "Founder", "Visionary"},
		QuestionsToAsk:    []string{"Why does this system exist?", "What assumptions am I making?", "What happens if I am wrong?"},
		HowToHelpYourself: []string{"Test ideas in reality early.", "Simplicity is often a strength.", "A working solution is usually better than a perfect design."},
	},
	{
		ID: "ARCHETYPE_PERFORMER", Key: "Performer", Title: "The Performer", Code: "PRF", DisplayOrder: 4,
		ShortDescription: "Uses attention, expression, and presence to move hearts and influence outcomes. They excel in the spotlight but benefit from remembering it is not always theirs to hold.",
		Echo:             "You understand the power of attention and expression. Whether through humor, storytelling, leadership, or presence, you know how to engage the people around you.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You understand that how something is communicated matters almost as much as what is communicated. Your instinct is often to think about the audience, the experience, and the emotional impact of a moment.",
			Strengths:  "Performers know how to hold attention. They can make ideas memorable, help groups feel connected, and create experiences that people carry with them long after the moment has passed. They often serve as translators between information and emotion.",
			Challenges: "Attention is a powerful tool, but it can become a tempting reward. Performers may struggle when they feel unseen or underappreciated, and they can sometimes focus too much on presentation at the expense of substance. Learning when to step forward and when to step back is an important part of their growth.",
			Socio:      "In Socio-, Performers often become storytellers, leaders, teachers, entertainers, speakers, diplomats, and public figures. They are at their best when using attention to help others understand, connect, and act.",
		},
		HEXACO:           "High Extraversion, Low Emotionality, Moderate Openness",
		HEXACOPlain:      "You tend to understand attention, mood, timing, and how people experience a moment together.",
		CoreDrives:       []string{"Expression", "Social Harmony"},
		PrimaryAttribute: "Presence", SecondaryAttribute: "Grace", KeySkill: "Performance",
		Motto:             "You understand that people rarely remember information, but they often remember how something felt.",
		Notices:           "You notice attention, emotion, timing, energy, and how experiences land with the people around you.",
		StrengthsList:     []string{"Communication", "Presence", "Adaptability", "Emotional engagement"},
		GrowthEdges:       []string{"Being comfortable outside the spotlight", "Separating attention from approval", "Showing vulnerability", "Letting moments belong to others"},
		GroupRole:         "In a group, you are often the person helping people connect with an idea, a story, or each other.",
		UnderStress:       "When stressed, you may become overly concerned with perception, seek validation, or hide your struggles behind performance.",
		PlaySuggestions:   []string{"Entertainer", "Speaker", "Bard", "Master of Ceremonies"},
		QuestionsToAsk:    []string{"How will this be experienced?", "What feeling am I creating?", "Am I seeking connection or approval?"},
		HowToHelpYourself: []string{"Not every moment needs an audience.", "Authenticity is stronger than polish.", "Let people see you, not just your performance."},
	},
	{
		ID: "ARCHETYPE_STRATEGIST", Key: "Strategist", Title: "The Strategist", Code: "STG", DisplayOrder: 5,
		ShortDescription: "Understands leverage, planning, and coordination. They see how pieces fit together and seek effective solutions without sacrificing principles.",
		Echo:             "You naturally think several steps ahead. You enjoy understanding how choices connect and often consider consequences before committing to a path.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "While other people focus on what's happening now, you're often thinking about what happens next. You like understanding how decisions, people, and resources interact over time.",
			Strengths:  "Strategists are skilled at finding leverage. They understand that not every problem needs more effort and that the right action at the right time can change everything. They often help groups avoid predictable mistakes because they notice consequences before they arrive.",
			Challenges: "Looking ahead has a cost. Strategists can become trapped in planning, overestimate their ability to predict events, or grow frustrated when others seem focused only on the present moment.",
			Socio:      "In Socio-, Strategists often become commanders, advisors, negotiators, coordinators, campaign planners, and problem-solvers. They thrive when they can align people and resources toward a shared goal.",
		},
		HEXACO:           "High Intellect, High Conscientiousness",
		HEXACOPlain:      "You tend to think ahead, track consequences, and look for the move that changes what happens next.",
		CoreDrives:       []string{"Planning", "Foresight", "Clarity"},
		PrimaryAttribute: "Intellect", SecondaryAttribute: "Lore", KeySkill: "Tactics",
		Motto:             "You are usually thinking one or two steps further ahead than the current conversation.",
		Notices:           "You notice leverage, timing, consequences, and opportunities that other people have not connected yet.",
		StrengthsList:     []string{"Planning", "Decision making", "Risk assessment", "Coordination"},
		GrowthEdges:       []string{"Acting without perfect information", "Trusting others with responsibility", "Remaining flexible", "Avoiding analysis paralysis"},
		GroupRole:         "In a group, you are often the person helping everyone understand where today's choices lead tomorrow.",
		UnderStress:       "When stressed, you may overthink, overplan, or become frustrated when people ignore obvious consequences.",
		PlaySuggestions:   []string{"Commander", "Advisor", "Negotiator", "Tactician"},
		QuestionsToAsk:    []string{"Where does this lead?", "What am I optimizing for?", "What consequences am I ignoring?"},
		HowToHelpYourself: []string{"Make decisions before certainty arrives.", "Some risks must be taken.", "People rarely follow plans as neatly as plans suggest."},
	},
	{
		ID: "ARCHETYPE_STEWARD", Key: "Steward", Title: "The Steward", Code: "STW", DisplayOrder: 6,
		ShortDescription: "Maintains systems of care, continuity, and well-being. They reliably support others and often forget that they, too, need support.",
		Echo:             "You value reliability, continuity, and care. You tend to think about how people, systems, and communities can be maintained over the long term.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You notice what has been entrusted to you and feel responsible for taking care of it. Whether it's a family, a team, a tradition, a project, or a community, your instinct is often to preserve and improve rather than replace.",
			Strengths:  "Stewards understand that many important things survive because someone quietly maintains them. They bring consistency, reliability, and follow-through to groups that might otherwise lose focus. People often trust Stewards because they do what they say they will do.",
			Challenges: "Stewards can become so focused on responsibility that they struggle to rest. They may continue carrying obligations long after they should have shared them or set them down. They can also become frustrated when others fail to care for things they consider important.",
			Socio:      "In Socio-, Stewards often become caretakers, administrators, organizers, community leaders, and guardians of important traditions. They help ensure that good things continue to exist tomorrow.",
		},
		HEXACO:           "High Conscientiousness, High Honesty-Humility, High Agreeableness",
		HEXACOPlain:      "You tend to care about continuity, responsibility, and whether people or systems can keep going over time.",
		CoreDrives:       []string{"Duty", "Nurture", "Protection"},
		PrimaryAttribute: "Craft", SecondaryAttribute: "Resolve", KeySkill: "Caregiving",
		Motto:             "You think about whether something can keep going long after everyone else has moved on.",
		Notices:           "You notice maintenance, responsibility, neglected needs, and the ongoing work required to keep people, places, and systems healthy.",
		StrengthsList:     []string{"Reliability", "Consistency", "Responsibility", "Long-term care"},
		GrowthEdges:       []string{"Asking for help", "Sharing responsibility", "Accepting change", "Making room for your own needs"},
		GroupRole:         "In a group, you are often the person making sure tomorrow still works after today's excitement is over.",
		UnderStress:       "When stressed, you may become overburdened, resentful, or feel like you are carrying responsibilities nobody else sees.",
		PlaySuggestions:   []string{"Caretaker", "Community Organizer", "Custodian", "Manager"},
		QuestionsToAsk:    []string{"What depends on me?", "What will still matter next year?", "What am I maintaining?"},
		HowToHelpYourself: []string{"You do not need to carry everything.", "Sustainability includes yourself.", "Responsibilities can be shared."},
	},
	{
		ID: "ARCHETYPE_CHALLENGER", Key: "Challenger", Title: "The Challenger", Code: "CHL", DisplayOrder: 7,
		ShortDescription: "Questions assumptions, systems, and hypocrisy. They make others uncomfortable in service of deeper truth and critical reflection.",
		Echo:             "You question assumptions and are willing to push back when something doesn't seem right. You would rather have an uncomfortable truth than a comfortable illusion.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You are not easily satisfied by 'because that's how it's always been done.' When something feels inconsistent, unfair, dishonest, or poorly reasoned, your instinct is often to examine it more closely.",
			Strengths:  "Challengers help people think more clearly. They test ideas, expose weak reasoning, and encourage others to examine beliefs that might otherwise go unquestioned. Their willingness to ask difficult questions often prevents groups from drifting into complacency.",
			Challenges: "Not every situation needs debate. Challengers can become frustrated when others avoid hard conversations, and others can become frustrated when Challengers continue asking questions after everyone else is ready to move on. Knowing when a question has served its purpose is an important skill.",
			Socio:      "In Socio-, Challengers often become investigators, advocates, critics, judges, philosophers, and truth-seekers. They are at their best when helping people separate what is true from what is merely familiar.",
		},
		HEXACO:           "Low Agreeableness, High Openness, High Honesty-Humility",
		HEXACOPlain:      "You tend to question assumptions, test claims, and prefer an uncomfortable truth over a comfortable mistake.",
		CoreDrives:       []string{"Truth", "Disruption", "Autonomy"},
		PrimaryAttribute: "Intellect", SecondaryAttribute: "Presence", KeySkill: "Debate",
		Motto:             "You would rather have an uncomfortable truth than a comfortable mistake.",
		Notices:           "You notice contradictions, weak arguments, assumptions, and places where the accepted story does not quite hold together.",
		StrengthsList:     []string{"Critical thinking", "Independent judgment", "Intellectual honesty", "Asking difficult questions"},
		GrowthEdges:       []string{"Knowing when a point has been made", "Recognizing emotional realities as well as logical ones", "Being patient with uncertainty", "Accepting that not every disagreement needs resolution"},
		GroupRole:         "In a group, you are often the person testing ideas before everyone commits to them.",
		UnderStress:       "When stressed, you may become argumentative, dismissive, or focus so heavily on flaws that you lose sight of what is working.",
		PlaySuggestions:   []string{"Investigator", "Debater", "Auditor", "Truth-Seeker"},
		QuestionsToAsk:    []string{"Is this true?", "What assumptions am I making?", "What evidence would change my mind?"},
		HowToHelpYourself: []string{"Being right and being effective are different things.", "Questions can build as well as dismantle.", "People are more than their arguments."},
	},
	{
		ID: "ARCHETYPE_SEEKER", Key: "Seeker", Title: "The Seeker", Code: "SEK", DisplayOrder: 8,
		ShortDescription: "Refuses to settle for surface answers. They pursue hidden truths, spiritual questions, and deeper meaning, though they can become consumed by a single mystery.",
		Echo:             "You are naturally curious and rarely satisfied with simple answers. When something captures your interest, you tend to keep digging until you understand it more fully.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You are drawn toward discovery. When you encounter a mystery, an unanswered question, or an unfamiliar path, your instinct is often to explore rather than move on.",
			Strengths:  "Seekers expand the boundaries of what people know and understand. They ask questions others never thought to ask and often discover ideas, places, and perspectives that would otherwise remain hidden. Their curiosity keeps them growing throughout their lives.",
			Challenges: "Curiosity can become a distraction when every answer leads to three more questions. Seekers may struggle to settle on a direction, especially when there is always another mystery waiting beyond the horizon. They can also become so focused on discovery that they lose sight of what they already know.",
			Socio:      "In Socio-, Seekers often become explorers, scholars, pilgrims, researchers, travelers, and students of the world. They are at their best when helping others discover something new.",
		},
		HEXACO:           "High Openness, High Emotionality, Moderate Extraversion",
		HEXACOPlain:      "You tend to follow questions, meanings, mysteries, and possibilities even when the path is not settled yet.",
		CoreDrives:       []string{"Meaning", "Discovery", "Growth"},
		PrimaryAttribute: "Spirit", SecondaryAttribute: "Lore", KeySkill: "Spiritual Insight",
		Motto:             "You rarely stop at the first answer when there is a deeper one waiting underneath.",
		Notices:           "You notice mysteries, unanswered questions, hidden connections, and possibilities that other people pass by without exploring.",
		StrengthsList:     []string{"Curiosity", "Open-mindedness", "Self-reflection", "Pursuit of meaning"},
		GrowthEdges:       []string{"Knowing when an answer is sufficient", "Committing to a path", "Avoiding endless searching", "Staying grounded in the present"},
		GroupRole:         "In a group, you are often the person asking questions nobody else thought to ask.",
		UnderStress:       "When stressed, you may become lost in possibilities, chase answers that never arrive, or struggle to settle on a direction.",
		PlaySuggestions:   []string{"Explorer", "Scholar", "Pilgrim", "Researcher"},
		QuestionsToAsk:    []string{"What am I really looking for?", "What question keeps returning?", "What would enough look like?"},
		HowToHelpYourself: []string{"Some answers only appear through action.", "Curiosity needs direction.", "Occasionally stop searching and practice living."},
	},
	{
		ID: "ARCHETYPE_RECONCILER", Key: "Reconciler", Title: "The Reconciler", Code: "REC", DisplayOrder: 9,
		ShortDescription: "Builds bridges between people and restores trust. Often caught between competing loyalties, they carry burdens others may not notice.",
		Echo:             "You value understanding and cooperation. When conflict arises, you're often looking for a way forward that preserves relationships rather than destroys them.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You rarely enjoy seeing people divided from one another. When conflict appears, your instinct is often to understand both sides and look for a path that allows people to move forward together.",
			Strengths:  "Reconcilers help people communicate when communication has broken down. They are often skilled listeners and can translate ideas, concerns, and emotions between people who struggle to understand one another. Their presence can prevent small disagreements from becoming lasting divisions.",
			Challenges: "Trying to understand everyone can be exhausting. Reconcilers may find themselves trapped between competing loyalties or blamed by multiple sides at once. They can also spend so much time seeking agreement that they delay necessary decisions.",
			Socio:      "In Socio-, Reconcilers often become diplomats, mediators, negotiators, interpreters, and community builders. They are at their best when helping people find common ground without ignoring real differences.",
		},
		HEXACO:           "High Agreeableness, High Honesty-Humility, Moderate Openness",
		HEXACOPlain:      "You tend to look for a path that preserves trust, restores understanding, and lets people remain human to each other.",
		CoreDrives:       []string{"Unity", "Diplomacy", "Healing"},
		PrimaryAttribute: "Empathy", SecondaryAttribute: "Presence", KeySkill: "Mediation",
		Motto:             "You usually find yourself trying to save relationships that other people are ready to give up on.",
		Notices:           "You notice distance between people. You notice misunderstandings, hurt feelings, broken trust, and opportunities to reconnect.",
		StrengthsList:     []string{"Building trust", "Finding common ground", "Conflict resolution", "Helping people feel heard"},
		GrowthEdges:       []string{"Setting boundaries", "Accepting that not every conflict can be resolved", "Avoiding people-pleasing", "Choosing a side when necessary"},
		GroupRole:         "In a group, you are often the person helping people work together after everyone else has stopped trying.",
		UnderStress:       "When stressed, you may take responsibility for conflicts that were never yours to solve or avoid necessary disagreements.",
		PlaySuggestions:   []string{"Diplomat", "Mediator", "Ambassador", "Community Leader"},
		QuestionsToAsk:    []string{"What relationship needs repair?", "What would trust require?", "Am I protecting peace or avoiding conflict?"},
		HowToHelpYourself: []string{"Not every disagreement is yours to solve.", "Boundaries are healthy.", "Sometimes honesty creates more peace than compromise."},
	},
	{
		ID: "ARCHETYPE_WATCHER", Key: "Watcher", Title: "The Watcher", Code: "WAT", DisplayOrder: 10,
		ShortDescription: "Observes before acting and listens before speaking. Their insight reveals patterns and truths, though they may hesitate to participate directly.",
		Echo:             "You prefer to understand a situation before rushing into it. Your attention to detail often helps you notice patterns, problems, and opportunities that others miss.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "While other people are already reacting, you're usually still paying attention. You want to understand what's happening before deciding what to do about it.",
			Strengths:  "You tend to notice details that others overlook. Whether it's a contradiction in someone's story, a strange pattern, or a small clue that doesn't fit, you're often gathering information while everyone else is spending it.",
			Challenges: "People may mistake your caution for indecision or distance. In reality, you're often trying to make sense of the situation before committing to a path. The challenge is knowing when you've learned enough and it's time to act.",
			Socio:      "In Socio-, Watchers often become investigators, scouts, scholars, advisors, analysts, and observers of human nature. They may not always lead from the front, but they frequently understand the situation before anyone else does.",
		},
		HEXACO:           "Low Extraversion, High Emotionality, High Conscientiousness",
		HEXACOPlain:      "You tend to observe carefully, notice consequences, and wait to understand the situation before acting.",
		CoreDrives:       []string{"Insight", "Safety", "Reflection"},
		PrimaryAttribute: "Awareness", SecondaryAttribute: "Spirit", KeySkill: "Insight",
		Motto:             "You notice the pattern before others notice there is a pattern.",
		Notices:           "You tend to notice details, contradictions, shifts in tone, and the quiet facts other people move past.",
		StrengthsList:     []string{"Pattern recognition", "Careful listening", "Situational awareness", "Patience"},
		GrowthEdges:       []string{"Acting before every detail is known", "Letting people know what you see", "Joining the moment instead of only observing it"},
		GroupRole:         "In a group, you often become the person who understands what is happening before anyone says it directly.",
		UnderStress:       "Under stress, you may withdraw, overthink, or keep gathering information long after you already know enough.",
		PlaySuggestions:   []string{"Investigator", "Scout", "Advisor", "Scholar", "Quiet Witness"},
		QuestionsToAsk:    []string{"What am I missing?", "What pattern am I seeing?", "What am I waiting for before acting?"},
		HowToHelpYourself: []string{"Observation is only the first step.", "Share what you notice.", "Sometimes the next piece of information arrives after you move."},
	},
	{
		ID: "ARCHETYPE_BUILDER", Key: "Builder", Title: "The Builder", Code: "BLD", DisplayOrder: 11,
		ShortDescription: "Sees what is missing and works to make it real. Their creations shape the world, though they sometimes become lost in what they build.",
		Echo:             "You enjoy turning ideas into something tangible and useful. You often see opportunities to improve, repair, or create where others only see problems.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "When you see something missing, your first instinct is usually to build it. Whether it's a tool, a project, a business, a tradition, or a solution, you're drawn toward making ideas real.",
			Strengths:  "You tend to focus on what can actually be done. While others debate possibilities, you're often looking for the next practical step. People appreciate Builders because they create things that continue helping long after the excitement of an idea has faded.",
			Challenges: "Builders sometimes become so focused on the work that they forget to ask whether the work is still serving its original purpose. It can also be difficult to stop building long enough to appreciate what has already been accomplished.",
			Socio:      "In Socio-, Builders often become craftspeople, entrepreneurs, engineers, organizers, community leaders, and founders. They leave their mark on the world through the things they create.",
		},
		HEXACO:           "High Conscientiousness, Moderate Honesty-Humility, Low Openness",
		HEXACOPlain:      "You tend to value practical follow-through, useful structure, and things that work in the real world.",
		CoreDrives:       []string{"Infrastructure", "Systems"},
		PrimaryAttribute: "Craft", SecondaryAttribute: "Resolve", KeySkill: "Construction",
		Motto:             "You see what is missing and immediately start thinking about how to make it real.",
		Notices:           "You notice gaps, inefficiencies, broken tools, unfinished work, and opportunities to improve something practical.",
		StrengthsList:     []string{"Follow-through", "Practical problem solving", "Creating useful things", "Reliability"},
		GrowthEdges:       []string{"Accepting imperfect solutions", "Delegating work", "Making time for people as well as projects", "Knowing when enough is enough"},
		GroupRole:         "In a group, you are often the person quietly turning ideas into something people can actually use.",
		UnderStress:       "When stressed, you may take on too much responsibility or become frustrated with people who do not carry their share.",
		PlaySuggestions:   []string{"Engineer", "Craftsperson", "Town Founder", "Quartermaster"},
		QuestionsToAsk:    []string{"What needs to exist that does not yet exist?", "What am I building?", "Who benefits from it?"},
		HowToHelpYourself: []string{"Perfection is often disguised procrastination.", "Let people help.", "Remember that relationships need maintenance too."},
	},
	{
		ID: "ARCHETYPE_FIREBRAND", Key: "Firebrand", Title: "The Firebrand", Code: "FIR", DisplayOrder: 12,
		ShortDescription: "Speaks boldly against injustice and inspires action. Their passion fuels change, though it can scorch allies and opponents alike.",
		Echo:             "You care deeply about change and are willing to challenge things that others simply accept. When you believe something needs to improve, you're rarely content to stay silent.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You do not like standing still when something needs to change. When you encounter a problem, your instinct is often to confront it directly rather than adapt to it or work around it.",
			Strengths:  "Firebrands help people act. They bring urgency to problems that others have learned to ignore and remind groups that frustration can be turned into meaningful change. Their conviction often inspires courage in people who might otherwise remain silent.",
			Challenges: "Passion is powerful, but it can also be exhausting. Firebrands may struggle with patience, especially when progress feels slow or resistance feels unreasonable. They can also find themselves carrying battles long after everyone else has gone home.",
			Socio:      "In Socio-, Firebrands often become reformers, activists, revolutionaries, whistleblowers, organizers, and outspoken leaders. They are at their best when helping people turn conviction into action.",
		},
		HEXACO:           "High Emotionality, Low Agreeableness, High Extraversion",
		HEXACOPlain:      "You tend to notice what people have learned to tolerate and feel pressure to challenge it when it seems wrong.",
		CoreDrives:       []string{"Conviction", "Upheaval", "Power"},
		PrimaryAttribute: "Presence", SecondaryAttribute: "Spirit", KeySkill: "Speechcraft",
		Motto:             "You have a hard time leaving a wrong thing alone simply because everyone else has gotten used to it.",
		Notices:           "You notice unfairness, hypocrisy, neglected problems, and the quiet cost of things people have learned to accept.",
		StrengthsList:     []string{"Moral courage", "Conviction", "Speaking difficult truths", "Motivating change"},
		GrowthEdges:       []string{"Choosing timing carefully", "Listening before acting", "Remembering change takes time", "Distinguishing resistance from opposition"},
		GroupRole:         "In a group, you are often the person asking why things are done that way in the first place.",
		UnderStress:       "When stressed, you may become impatient, cynical, or so focused on the problem that you lose sight of the people around it.",
		PlaySuggestions:   []string{"Reformer", "Advocate", "Whistleblower", "Revolutionary"},
		QuestionsToAsk:    []string{"What am I no longer willing to accept?", "What change am I actually seeking?", "Is my anger serving the outcome?"},
		HowToHelpYourself: []string{"Choose your battles.", "Give allies room to learn.", "Focus as much on solutions as problems."},
	},
	{
		ID: "ARCHETYPE_GUARDIAN", Key: "Guardian", Title: "The Guardian", Code: "GRD", DisplayOrder: 13,
		ShortDescription: "Stands between danger and what matters most. Fiercely protective, they may endure overwhelming hardship on behalf of others.",
		Echo:             "You keep an eye on what matters and feel responsible for protecting it. When others overlook a risk, you're often the one thinking about what could go wrong.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You pay attention to what needs protection. Whether it's a person, a community, a principle, or a place, your instinct is often to place yourself between it and potential harm.",
			Strengths:  "Guardians are dependable when things become difficult. They are willing to shoulder responsibility, confront threats, and remain steady during uncertainty. People often feel safer when a Guardian is nearby because they know someone is paying attention to the risks.",
			Challenges: "Guardians can become overly protective, carrying dangers that do not belong to them or struggling to trust others with responsibilities they consider important. Constant vigilance can also become exhausting over time.",
			Socio:      "In Socio-, Guardians often become defenders, protectors, officers, advocates, mentors, and champions of vulnerable people. They help ensure that what matters survives long enough to flourish.",
		},
		HEXACO:           "High Conscientiousness, Low Openness, High Emotionality",
		HEXACOPlain:      "You tend to notice risk, duty, and what must be protected before things become unsafe.",
		CoreDrives:       []string{"Order", "Structure", "Stability"},
		PrimaryAttribute: "Resolve", SecondaryAttribute: "Lore", KeySkill: "Discipline",
		Motto:             "You naturally keep track of what could go wrong before it does.",
		Notices:           "You notice risks, vulnerabilities, responsibilities, and the people or things that depend on someone watching out for them.",
		StrengthsList:     []string{"Protectiveness", "Preparedness", "Loyalty", "Dependability"},
		GrowthEdges:       []string{"Trusting others", "Letting go of control", "Accepting uncertainty", "Recognizing when danger has passed"},
		GroupRole:         "In a group, you are often the person making sure everyone gets home safely.",
		UnderStress:       "When stressed, you may become overly cautious, controlling, or carry burdens that do not actually belong to you.",
		PlaySuggestions:   []string{"Protector", "Bodyguard", "Parent Figure", "Watch Captain"},
		QuestionsToAsk:    []string{"What must be protected?", "What danger am I preparing for?", "Has the threat actually arrived?"},
		HowToHelpYourself: []string{"Not every risk requires intervention.", "Trust is a form of strength.", "You are allowed to put your burdens down."},
	},
	{
		ID: "ARCHETYPE_COMFORTER", Key: "Comforter", Title: "The Comforter", Code: "CFT", DisplayOrder: 14,
		ShortDescription: "A great friend who ensures others are comfortable and cared for. They often focus so much on others that they forget to seek comfort themselves.",
		Echo:             "You tend to notice how people are feeling and often find yourself looking for ways to help. Even when you cannot solve a problem, you rarely stop caring about the people affected by it.",
		Primary: Chapter3ArchetypePrimary{
			Intro:      "You naturally pay attention to people. While others may focus on goals, plans, or problems, you're often aware of how those things are affecting the individuals involved.",
			Strengths:  "Comforters help people feel seen, heard, and valued. They often notice discomfort before it is spoken aloud and can create a sense of safety simply by being present. Friends, family members, and communities often rely on them more than they realize.",
			Challenges: "Because Comforters spend so much time caring for others, they sometimes neglect their own needs. They may also carry burdens that were never theirs to carry in the first place. Caring deeply is a strength, but it works best when paired with healthy boundaries.",
			Socio:      "In Socio-, Comforters often become healers, mentors, caretakers, counselors, hosts, and trusted friends. They strengthen groups by making sure people are not forgotten.",
		},
		HEXACO:           "High Emotionality, High Agreeableness, High Honesty-Humility",
		HEXACOPlain:      "You tend to feel the human cost of a situation quickly and orient toward care, safety, and relief.",
		CoreDrives:       []string{"Healing", "Belonging", "Compassion"},
		PrimaryAttribute: "Empathy", SecondaryAttribute: "Resolve", KeySkill: "Comfort",
		Motto:             "You notice the person before you notice the problem.",
		Notices:           "You notice pain, exhaustion, loneliness, fear, and the small signs that someone is carrying more than they let on.",
		StrengthsList:     []string{"Compassion", "Emotional support", "Patience", "Helping people feel safe"},
		GrowthEdges:       []string{"Taking care of yourself", "Accepting help from others", "Letting people struggle when they need to grow", "Protecting your own energy"},
		GroupRole:         "In a group, you are often the person checking on how everyone is doing while others focus on what needs to be done.",
		UnderStress:       "When stressed, you may neglect your own needs, absorb other people's problems, or become emotionally overwhelmed.",
		PlaySuggestions:   []string{"Healer", "Caretaker", "Mentor", "Trusted Friend"},
		QuestionsToAsk:    []string{"Who needs care?", "What am I carrying that belongs to someone else?", "What do I need right now?"},
		HowToHelpYourself: []string{"You are allowed to receive help.", "Compassion includes yourself.", "Rest is not selfish."},
	},
}

// Chapter3ArchetypeByKey returns the catalog entry for a stable key
// ("Catalyst", "Builder", ...), case-sensitive to match the source dataset.
func Chapter3ArchetypeByKey(key string) (Chapter3Archetype, bool) {
	for _, a := range Chapter3Archetypes {
		if a.Key == key {
			return a, true
		}
	}
	return Chapter3Archetype{}, false
}

// ValidateChapter3Archetypes checks catalog completeness and integrity.
func ValidateChapter3Archetypes() error {
	if len(Chapter3Archetypes) != 14 {
		return fmt.Errorf("chapter3: expected 14 archetypes, got %d", len(Chapter3Archetypes))
	}
	seenKeys := map[string]bool{}
	seenOrder := map[int]bool{}
	for _, a := range Chapter3Archetypes {
		if a.Key == "" || a.Title == "" || a.ID == "" {
			return fmt.Errorf("chapter3: archetype %q missing required identity fields", a.Key)
		}
		if seenKeys[a.Key] {
			return fmt.Errorf("chapter3: duplicate archetype key %q", a.Key)
		}
		seenKeys[a.Key] = true
		if seenOrder[a.DisplayOrder] {
			return fmt.Errorf("chapter3: duplicate display_order %d", a.DisplayOrder)
		}
		seenOrder[a.DisplayOrder] = true
		if a.PrimaryAttribute == "" || a.SecondaryAttribute == "" || a.KeySkill == "" {
			return fmt.Errorf("chapter3: archetype %q missing Mechanics fields", a.Key)
		}
	}
	return nil
}
