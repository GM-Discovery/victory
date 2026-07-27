package merchant

// Director+ HTTP surface for dialogue authoring (kernel-75, operator
// decision 3).
//
// Lives in the merchant package rather than in dialogue for the same reason
// every other participant HTTP handler does: dialogue is a leaf that holds
// no authority logic, and the writeOK/writeError envelope lives here.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dialogue"
	"victory/backend/internal/showruns"
)

// packetLocation resolves the Location a packet belongs to, which is what
// the authority helpers are scoped by.
func packetLocation(ctx context.Context, pool *pgxpool.Pool, packetID string) (string, error) {
	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT location_id::text FROM dialogue_packets WHERE id = $1
	`, packetID).Scan(&locationID); err != nil {
		return "", errors.New("dialogue_packet_not_found")
	}
	return locationID, nil
}

func topicPacketLocation(ctx context.Context, pool *pgxpool.Pool, topicID string) (string, string, error) {
	var packetID, locationID string
	if err := pool.QueryRow(ctx, `
		SELECT p.id::text, p.location_id::text
		FROM dialogue_topics t JOIN dialogue_packets p ON p.id = t.packet_id
		WHERE t.id = $1
	`, topicID).Scan(&packetID, &locationID); err != nil {
		return "", "", errors.New("dialogue_topic_not_found")
	}
	return packetID, locationID, nil
}

// HandleLocationDialoguePackets handles GET
// /api/locations/{location_id}/dialogue-packets. Gate: CanViewBackstage.
func HandleLocationDialoguePackets(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		locationID := strings.TrimSpace(r.PathValue("location_id"))
		allowed, err := showruns.CanViewBackstage(ctx, pool, userID, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}
		packets, err := dialogue.ListPacketsForLocation(ctx, pool, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"packets": packets})
	}
}

// HandleDialoguePacketByID handles GET and PATCH
// /api/dialogue-packets/{packet_id}.
//
// Reads take CanViewBackstage; WRITES take CanManageShowRun, which is
// stricter than the CanCrewPerformNonDestructiveEdit that
// merchant.UpdateInteraction uses. The distinction is deliberate: an
// interaction's button label is stage furniture, whereas an NPC's authored
// prose is canon that every Player reads verbatim. The operator asked for
// Director+ on prose, and this is where that lands.
func HandleDialoguePacketByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		packetID := strings.TrimSpace(r.PathValue("packet_id"))
		locationID, err := packetLocation(ctx, pool, packetID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			allowed, err := showruns.CanViewBackstage(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !allowed {
				writeError(w, errors.New("not_authorized"))
				return
			}
			packet, topics, err := dialogue.LoadPacketForEditing(ctx, pool, packetID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"packet": packet, "topics": topics})

		case http.MethodPatch:
			allowed, err := showruns.CanManageShowRun(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !allowed {
				writeError(w, errors.New("not_authorized"))
				return
			}
			var patch dialogue.PacketPatch
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			packet, err := dialogue.UpdatePacket(ctx, pool, userID, packetID, patch)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"packet": packet})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleDialogueTopicByID handles PATCH /api/dialogue-topics/{topic_id}.
// Gate: CanManageShowRun at the packet's Location.
func HandleDialogueTopicByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		topicID := strings.TrimSpace(r.PathValue("topic_id"))
		_, locationID, err := topicPacketLocation(ctx, pool, topicID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := showruns.CanManageShowRun(ctx, pool, userID, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}
		var patch dialogue.TopicPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		topic, err := dialogue.UpdateTopic(ctx, pool, userID, topicID, patch)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"topic": topic})
	}
}

// HandleDialoguePacketRevisions handles GET
// /api/dialogue-packets/{packet_id}/revisions. Gate: CanViewBackstage.
func HandleDialoguePacketRevisions(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		packetID := strings.TrimSpace(r.PathValue("packet_id"))
		locationID, err := packetLocation(ctx, pool, packetID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := showruns.CanViewBackstage(ctx, pool, userID, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}
		revisions, err := dialogue.ListRevisions(ctx, pool, packetID, 50)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"revisions": revisions})
	}
}
