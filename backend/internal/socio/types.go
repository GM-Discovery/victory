// Package socio is Kernel 85's Socio Game Status mechanical layer: eight
// attribute-based HP pools plus a status-effect registry
// (character_socio_state, socio_statuses, character_socio_status_effects --
// migration 097). None of this existed before this kernel (repo audit
// found CharacterCard carries no attribute/HP fields at all); it is
// deliberately scoped narrow per operator decision (2026-08-09): paired
// current/maximum integers with fixed labels, no character-sheet UI
// wiring, no derived fields. This is the one canonical mechanical store --
// Game Status must never write anywhere else.
package socio

import "time"

// PoolKey identifies one of the eight fixed HP pools. The Attribute names
// (Might, Intellect, ...) are Socio character-build vocabulary that already
// exists elsewhere (backend/internal/characters' chapter2-4 files) and are
// carried here only as display labels -- PoolKey is always the HP-pool
// name (Health, Psyche, ...), never the attribute name, matching kernel-85
// S2's table.
type PoolKey string

const (
	PoolHealth     PoolKey = "health"
	PoolPsyche     PoolKey = "psyche"
	PoolMotion     PoolKey = "motion"
	PoolWill       PoolKey = "will"
	PoolEssence    PoolKey = "essence"
	PoolFocus      PoolKey = "focus"
	PoolPerception PoolKey = "perception"
	PoolHeart      PoolKey = "heart"
)

// PoolLabels is the fixed display order and Attribute -> HP Pool mapping
// from kernel-85 S2's table.
var PoolLabels = []struct {
	Key       PoolKey
	Label     string
	Attribute string
}{
	{PoolHealth, "Health", "Might"},
	{PoolPsyche, "Psyche", "Intellect"},
	{PoolMotion, "Motion", "Grace"},
	{PoolWill, "Will", "Presence"},
	{PoolEssence, "Essence", "Spirit"},
	{PoolFocus, "Focus", "Resolve"},
	{PoolPerception, "Perception", "Awareness"},
	{PoolHeart, "Heart", "Empathy"},
}

func IsValidPoolKey(key string) bool {
	for _, p := range PoolLabels {
		if string(p.Key) == key {
			return true
		}
	}
	return false
}

// Pool is one HP pool's current/maximum pair, curated for the Game Status
// wire shape.
type Pool struct {
	Key     PoolKey `json:"key"`
	Label   string  `json:"label"`
	Current int     `json:"current"`
	Max     int     `json:"max"`
}

// State is one Character's full set of eight HP pools.
type State struct {
	CharacterCardID string    `json:"character_card_id"`
	Pools           []Pool    `json:"pools"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StatusDefinition is one row of socio_statuses -- registry data, not a Go
// const list (kernel-85 S2.3: "do not assume this list is exhaustive or
// final").
type StatusDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ActiveStatus is one currently-applied (uncleared) status effect on a
// Character. Exactly one of StatusKey (a canonical socio_statuses tag) or
// CustomLabel (a Kernel 88 Director-authored blank state, e.g. "Waiting on
// Kessa") is set -- migration 102's check constraint enforces this at the
// database level; IsBlank mirrors that split for callers.
type ActiveStatus struct {
	ID              string    `json:"id"`
	StatusKey       string    `json:"status_key,omitempty"`
	Label           string    `json:"label"`
	CustomLabel     string    `json:"custom_label,omitempty"`
	IsBlank         bool      `json:"is_blank"`
	Intensity       *int      `json:"intensity,omitempty"`
	AppliedByUserID string    `json:"applied_by_user_id"`
	AppliedAt       time.Time `json:"applied_at"`
}

// CharacterBlock is one Character's full Game Status card: identity, HP
// pools, active statuses (kernel-85 S2.1's required per-Character block).
// Social Stance and Fate/EP fields are omitted -- neither is canonical
// state anywhere in this repo yet (repo audit S13.9), and S2.5 forbids
// inventing a Stance subsystem merely to decorate this panel.
type CharacterBlock struct {
	CharacterCardID string         `json:"character_card_id"`
	CharacterName   string         `json:"character_name"`
	UserID          string         `json:"user_id"`
	CohortID        string         `json:"cohort_id,omitempty"`
	Pools           []Pool         `json:"pools"`
	ActiveStatuses  []ActiveStatus `json:"active_statuses"`
}
