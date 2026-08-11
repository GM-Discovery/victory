// Package stageeffects implements Kernel 86's transient/static Stage
// Effect contract: a lightweight, in-memory, session-scoped presentation
// record derived from an already-canonical Action (the `roll/dice` Action
// remains the sole source of truth -- kernel §5, §6). It follows the same
// "ephemeral live-session state, no DB table, nothing to migrate" shape as
// backend/internal/venuecoordination's Registry (Kernel 83), and the same
// division of labor: this package knows nothing about Victory roles,
// authority, or audience resolution -- it only tracks "which effects exist
// right now for this session, and which of those are pinned." The caller
// (backend/internal/network/ws.go) decides who may create/pin/dismiss an
// effect, using backend/internal/rollaudience for the roll's own audience.
package stageeffects

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// Effect is one theatrical stage effect (Kernel 86 §6). Payload carries
// whatever the effect Type needs to render -- for type "dice_roll" this is
// the curated actor/label/dice/modifier/total/explosion-chain display
// shape, never anything the recipient wasn't already authorized to receive
// (the caller only ever creates an Effect for users already resolved as
// recipients by rollaudience).
type Effect struct {
	ID             string         `json:"effect_id"`
	Type           string         `json:"type"`
	SourceActionID string         `json:"source_action_id"`
	SessionID      string         `json:"session_id"`
	ShowID         string         `json:"show_id,omitempty"`
	CohortID       string         `json:"cohort_id,omitempty"`
	Audience       string         `json:"audience"`
	ActorID        string         `json:"actor_id"`
	Label          string         `json:"label,omitempty"`
	Payload        map[string]any `json:"payload"`
	CreatedAt      time.Time      `json:"created_at"`
	DurationMs     int            `json:"duration_ms"`
	Pinned         bool           `json:"pinned"`
}

// expiryGrace is added to DurationMs before a never-pinned transient effect
// is swept from the registry, so a slightly-late reconnect or pin request
// arriving right at the visual fade-out boundary still finds it.
const expiryGrace = 5 * time.Second

// Registry holds active Stage Effects per session, entirely in memory --
// deliberate per kernel §6.1 ("do not create a second durable dice-history
// universe"). Transient effects are swept lazily (on the next registry
// call that touches their session) once DurationMs+expiryGrace has
// elapsed; pinned effects are retained until explicitly dismissed.
type Registry struct {
	mu       sync.Mutex
	sessions map[string]map[string]*Effect
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]map[string]*Effect)}
}

func randomEffectID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return "fx_" + hex.EncodeToString(buf[:])
	}
	return "fx_fallback"
}

// sweep removes expired, never-pinned effects from a session's map. Caller
// must hold r.mu.
func sweep(effects map[string]*Effect, now time.Time) {
	for id, e := range effects {
		if e.Pinned {
			continue
		}
		if now.Sub(e.CreatedAt) > time.Duration(e.DurationMs)*time.Millisecond+expiryGrace {
			delete(effects, id)
		}
	}
}

// Create registers a new Stage Effect and returns it with ID/CreatedAt
// assigned. sessionID and e.SessionID are expected to match; e.ID is
// ignored and always regenerated.
func (r *Registry) Create(sessionID string, e Effect) Effect {
	sessionID = strings.TrimSpace(sessionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	effects, ok := r.sessions[sessionID]
	if !ok {
		effects = make(map[string]*Effect)
		r.sessions[sessionID] = effects
	}
	sweep(effects, time.Now())

	e.ID = randomEffectID()
	e.SessionID = sessionID
	e.CreatedAt = time.Now()
	e.Pinned = false
	stored := e
	effects[e.ID] = &stored
	return stored
}

// Get returns effectID's current state for sessionID, if it still exists
// (transient effects vanish once expired).
func (r *Registry) Get(sessionID, effectID string) (Effect, bool) {
	sessionID = strings.TrimSpace(sessionID)
	effectID = strings.TrimSpace(effectID)

	r.mu.Lock()
	defer r.mu.Unlock()

	effects, ok := r.sessions[sessionID]
	if !ok {
		return Effect{}, false
	}
	sweep(effects, time.Now())
	e, ok := effects[effectID]
	if !ok {
		return Effect{}, false
	}
	return *e, true
}

// Pin marks effectID as static/pinned so it survives the transient sweep
// (kernel §1.5, §10). Authority is entirely the caller's job.
func (r *Registry) Pin(sessionID, effectID string) (Effect, bool) {
	sessionID = strings.TrimSpace(sessionID)
	effectID = strings.TrimSpace(effectID)

	r.mu.Lock()
	defer r.mu.Unlock()

	effects, ok := r.sessions[sessionID]
	if !ok {
		return Effect{}, false
	}
	sweep(effects, time.Now())
	e, ok := effects[effectID]
	if !ok {
		return Effect{}, false
	}
	e.Pinned = true
	return *e, true
}

// Dismiss removes effectID entirely (unpin-and-clear in one step -- kernel
// §10 has no "unpin but keep transiently visible" state, only pinned or
// gone). Returns false if it did not exist.
func (r *Registry) Dismiss(sessionID, effectID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	effectID = strings.TrimSpace(effectID)

	r.mu.Lock()
	defer r.mu.Unlock()

	effects, ok := r.sessions[sessionID]
	if !ok {
		return false
	}
	if _, ok := effects[effectID]; !ok {
		return false
	}
	delete(effects, effectID)
	return true
}

// ListPinned returns every currently-pinned effect for sessionID, for a
// reconnecting client to restore static projections (kernel §10: "preserve
// it across ordinary reconnect if bounded"). The caller must still filter
// this by the viewer's own audience authority before sending -- this
// registry does no role/visibility filtering, per this package's standing
// invariant.
func (r *Registry) ListPinned(sessionID string) []Effect {
	sessionID = strings.TrimSpace(sessionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	effects, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	sweep(effects, time.Now())

	out := make([]Effect, 0, len(effects))
	for _, e := range effects {
		if e.Pinned {
			out = append(out, *e)
		}
	}
	return out
}

// EndSession clears all Stage Effects for sessionID (mirrors
// venuecoordination.Registry.EndSession's session-lifecycle contract).
func (r *Registry) EndSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
}
