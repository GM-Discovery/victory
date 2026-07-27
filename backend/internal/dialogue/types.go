// Package dialogue is Kernel 74's bounded guided-dialogue Program: an
// authored NPC packet (Ra) delivered through topic buttons with
// prerequisites and per-participant seen-state, ending in a Player-
// controlled Leave.
//
// What this is NOT, deliberately (kernel-74 S1.6, S15): a chatbot, AI
// dialogue, a general dialogue-graph editor, arbitrary scripting, or a
// Director-dependent live-roleplay dependency for required tutorial
// exposition. Topics are a flat authored list with prerequisite edges. The
// graph is validated acyclic on load, so a bad seed fails loudly rather
// than stranding a Player in an unreachable topic.
//
// The Director may still roleplay Ra out loud; the Player never waits for
// them to (S9.6).
package dialogue

import "time"

// Packet is one NPC's authored conversation.
type Packet struct {
	ID                   string    `json:"id"`
	LocationID           string    `json:"location_id"`
	Slug                 string    `json:"slug"`
	NPCName              string    `json:"npc_name"`
	PortraitAssetID      string    `json:"portrait_asset_id,omitempty"`
	PortraitURL          string    `json:"portrait_url,omitempty"`
	OpeningNarration     string    `json:"opening_narration"`
	OpeningLine          string    `json:"opening_line"`
	ClosingNarration     string    `json:"closing_narration"`
	LeaveLabel           string    `json:"leave_label"`
	DestinationSceneSlug string    `json:"destination_scene_slug,omitempty"`
	Active               bool      `json:"active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// ClosingBeats is the staged lock-reveal ending (kernel-75 S3.1), added
	// by migration 072. Never read this field directly -- use
	// ClosingBeatsOrNarration, which supplies the closing_narration fallback
	// an unedited or pre-072 packet needs.
	ClosingBeats []string `json:"closing_beats"`
	// ContentOrigin is 'seed' until a Director edits this packet through the
	// authoring API, then 'authored'. Content migrations must only touch
	// rows still marked 'seed'.
	ContentOrigin string `json:"content_origin"`
}

// Topic is one authored question and its authored answer.
//
// ResponseText is intentionally present on every topic in the Player-facing
// payload only AFTER the topic is unlocked -- see TopicView, which is what
// actually crosses the wire.
type Topic struct {
	ID                    string   `json:"id"`
	PacketID              string   `json:"packet_id"`
	TopicKey              string   `json:"topic_key"`
	Label                 string   `json:"label"`
	ResponseText          string   `json:"response_text"`
	RequiredForCompletion bool     `json:"required_for_completion"`
	SortOrder             int      `json:"sort_order"`
	Active                bool     `json:"active"`
	RequiresTopicKeys     []string `json:"requires_topic_keys,omitempty"`
}

// TopicView is the curated per-Player shape of a topic, mirroring how
// cues.PlayerVisibleCue curates a Cue: a locked topic's authored response
// never leaves the server, so a Player cannot read ahead by inspecting the
// network payload and then claim to have "seen" it.
type TopicView struct {
	TopicKey string `json:"topic_key"`
	Label    string `json:"label"`
	Seen     bool   `json:"seen"`
	Unlocked bool   `json:"unlocked"`
	Required bool   `json:"required"`
	SortOrder int   `json:"sort_order"`
}

// State is the whole Program Panel payload for one Player at one moment.
// Every field is server-computed; the client renders it and never derives
// unlock or completion itself (S9.1, S13).
type State struct {
	PacketSlug       string      `json:"packet_slug"`
	NPCName          string      `json:"npc_name"`
	PortraitAssetID  string      `json:"portrait_asset_id,omitempty"`
	PortraitURL      string      `json:"portrait_url,omitempty"`
	OpeningNarration string      `json:"opening_narration"`
	// CurrentResponse is the line currently displayed: the packet's opening
	// line on open, or the response of the topic just asked.
	CurrentResponse string      `json:"current_response"`
	Topics          []TopicView `json:"topics"`
	LeaveLabel      string      `json:"leave_label"`
	// CanLeave gates the Leave control (S9.3): true once every
	// required_for_completion topic has been viewed. Optional topics are
	// never required.
	CanLeave bool `json:"can_leave"`
	// ClosingNarration is populated only by Leave -- the lock-reveal beat
	// (S10.1). Empty on open and on topic reads.
	ClosingNarration string `json:"closing_narration,omitempty"`

	// ClosingBeats is the same ending broken into ordered beats so the
	// Program can reveal it a step at a time (kernel-75 S3.1). Populated
	// only by Leave. Always non-empty when ClosingNarration is non-empty,
	// because it comes from Packet.ClosingBeatsOrNarration -- a client may
	// render either, and an older client that only knows ClosingNarration
	// keeps working unchanged.
	ClosingBeats []string `json:"closing_beats,omitempty"`
}
