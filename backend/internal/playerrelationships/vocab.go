package playerrelationships

// VocabTerm is one named dropdown value. Users always see the label; the
// backend stores the key. There are no visible numeric scores
// (Kernel 62 §4.3, §6).
type VocabTerm struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// The qualitative vocabularies are fixed by the kernel spec (Kernel 62 §6).
// Order here is the display/sort order.
var (
	TrustVocab = []VocabTerm{
		{Key: "unknown", Label: "Unknown"},
		{Key: "cautious", Label: "Cautious"},
		{Key: "developing", Label: "Developing"},
		{Key: "trusted", Label: "Trusted"},
		{Key: "deeply_trusted", Label: "Deeply Trusted"},
	}

	ClosenessVocab = []VocabTerm{
		{Key: "unknown", Label: "Unknown"},
		{Key: "distant", Label: "Distant"},
		{Key: "familiar", Label: "Familiar"},
		{Key: "friendly", Label: "Friendly"},
		{Key: "close", Label: "Close"},
		{Key: "core_relationship", Label: "Core Relationship"},
	}

	ReliabilityVocab = []VocabTerm{
		{Key: "unknown", Label: "Unknown"},
		{Key: "inconsistent", Label: "Inconsistent"},
		{Key: "usually_reliable", Label: "Usually Reliable"},
		{Key: "reliable", Label: "Reliable"},
		{Key: "highly_reliable", Label: "Highly Reliable"},
	}

	CommunicationEaseVocab = []VocabTerm{
		{Key: "unknown", Label: "Unknown"},
		{Key: "difficult", Label: "Difficult"},
		{Key: "uneven", Label: "Uneven"},
		{Key: "workable", Label: "Workable"},
		{Key: "easy", Label: "Easy"},
		{Key: "very_easy", Label: "Very Easy"},
	}

	// RelationshipStateVocab includes "archived" for display, but the
	// archived state is only ever entered/left through the archive and
	// unarchive operations -- a direct state update cannot set it
	// (Kernel 62 §4.6, §14).
	RelationshipStateVocab = []VocabTerm{
		{Key: "active", Label: "Active"},
		{Key: "quiet", Label: "Quiet"},
		{Key: "strained", Label: "Strained"},
		{Key: "rebuilding", Label: "Rebuilding"},
		{Key: "archived", Label: "Archived"},
	}
)

const StateArchived = "archived"

// CategoryKeys are the initial checkbox categories (Kernel 62 §5.2).
// "custom" requires a non-empty custom label.
var CategoryKeys = []string{
	"acquaintance",
	"friend",
	"close_friend",
	"family",
	"collaborator",
	"coworker",
	"player",
	"gm_facilitator",
	"mentor",
	"mentee",
	"client",
	"community_contact",
	"creative_partner",
	"professional_contact",
	"custom",
}

const CategoryCustom = "custom"

// CategoryLabels maps category keys to display labels for the frontend.
var CategoryLabels = map[string]string{
	"acquaintance":         "Acquaintance",
	"friend":               "Friend",
	"close_friend":         "Close Friend",
	"family":               "Family",
	"collaborator":         "Collaborator",
	"coworker":             "Coworker",
	"player":               "Player",
	"gm_facilitator":       "GM / Facilitator",
	"mentor":               "Mentor",
	"mentee":               "Mentee",
	"client":               "Client",
	"community_contact":    "Community Contact",
	"creative_partner":     "Creative Partner",
	"professional_contact": "Professional Contact",
	"custom":               "Custom",
}

// FollowUp statuses (Kernel 62 §5.6).
const (
	FollowUpOpen      = "open"
	FollowUpDone      = "done"
	FollowUpDismissed = "dismissed"
)

var followUpStatuses = map[string]bool{
	FollowUpOpen:      true,
	FollowUpDone:      true,
	FollowUpDismissed: true,
}

func vocabHasKey(vocab []VocabTerm, key string) bool {
	for _, term := range vocab {
		if term.Key == key {
			return true
		}
	}
	return false
}

// ValidateTrustKey and friends reject any key outside the fixed vocabulary
// (Kernel 62 §6, §15.1 "qualitative values reject invalid keys").
func ValidateTrustKey(key string) bool             { return vocabHasKey(TrustVocab, key) }
func ValidateClosenessKey(key string) bool         { return vocabHasKey(ClosenessVocab, key) }
func ValidateReliabilityKey(key string) bool       { return vocabHasKey(ReliabilityVocab, key) }
func ValidateCommunicationEaseKey(key string) bool { return vocabHasKey(CommunicationEaseVocab, key) }

// ValidateSettableStateKey accepts only states an observer may set directly.
// "archived" is excluded -- it is owned by archive/unarchive (Kernel 62 §14).
func ValidateSettableStateKey(key string) bool {
	return key != StateArchived && vocabHasKey(RelationshipStateVocab, key)
}

// ValidateCategoryKey reports whether key is a known checkbox category.
func ValidateCategoryKey(key string) bool {
	for _, k := range CategoryKeys {
		if k == key {
			return true
		}
	}
	return false
}

func ValidateFollowUpStatus(status string) bool {
	return followUpStatuses[status]
}
