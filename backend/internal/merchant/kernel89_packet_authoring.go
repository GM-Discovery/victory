// Kernel 89 §9: merchant authoring.
//
// Kernel 73 shipped the merchant MODEL (merchant_packets +
// merchant_packet_equipment_items, migration 061) but authored Kessa
// through a seed migration, leaving "a Director cannot currently edit
// Kessa's dialogue without a new migration" as a recorded gap. This file
// closes exactly that gap and nothing more: it writes the SAME two tables
// through the same shapes LoadPacketBySlug already reads.
//
// Two boundaries it deliberately holds:
//
//  1. There is no second equipment catalog (§9.1, and a FAIL criterion in
//     §41). Stock is chosen by id from `equipment_items` at the packet's own
//     location -- the canonical corpus seeded by migration 063 -- and an id
//     from anywhere else is rejected, not created.
//
//  2. It is not a dialogue-tree editor (§2, §38). The authored fields are
//     exactly Kernel 73's five fixed stance slots plus one Haggle slot; there
//     is no way to add a sixth stance, nest a response, or make one response
//     lead to another, because the storage has no column for it.
package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxPacketTextLength  = 2000
	maxPacketLabelLength = 120
	maxStanceResponses   = 5
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// slugify derives a stable slug from a display name so a Director never has
// to think about one. Slug is the packet's identity in
// participant_interactions.configuration_json, so it is generated once at
// create time and never rewritten by a later rename -- renaming a merchant
// must not orphan every interaction pointing at them.
func slugify(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := true
	for _, r := range lowered {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func boundedText(value string, max int, field string) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > max {
		return "", errors.New(field + "_too_long")
	}
	return value, nil
}

// normalizeStances accepts a partial map and returns exactly the five fixed
// stance slots, filling missing ones with a neutral default. Unknown stance
// keys are rejected rather than dropped: a client sending "intimidate" has
// a wrong idea about the model, and silently discarding it would let a
// Director believe they authored something that will never appear.
func normalizeStances(in map[string]StanceDisposition) (map[string]StanceDisposition, error) {
	known := map[string]bool{}
	for _, k := range FixedStanceKeys {
		known[k] = true
	}
	for key := range in {
		if !known[strings.ToLower(strings.TrimSpace(key))] {
			return nil, errors.New("unknown_stance_key")
		}
	}

	out := map[string]StanceDisposition{}
	for _, key := range FixedStanceKeys {
		entry := in[key]
		disposition := strings.ToLower(strings.TrimSpace(entry.Disposition))
		switch disposition {
		case "reject", "positive", "neutral":
		case "":
			disposition = "neutral"
		default:
			return nil, errors.New("unknown_stance_disposition")
		}
		responses := []string{}
		for _, raw := range entry.Responses {
			text, err := boundedText(raw, maxPacketTextLength, "stance_response")
			if err != nil {
				return nil, err
			}
			if text != "" {
				responses = append(responses, text)
			}
			if len(responses) >= maxStanceResponses {
				break
			}
		}
		out[key] = StanceDisposition{Disposition: disposition, Responses: responses}
	}
	return out, nil
}

// PacketInput is the authored packet. Haggle mechanics are intentionally
// absent: they stay at Kernel 73's seeded defaults (d6 skilled / d4
// unskilled / target 5) because inventing per-merchant dice would be a
// mechanics change, not an authoring feature.
type PacketInput struct {
	DisplayName        string
	IntroText          string
	StanceDispositions map[string]StanceDisposition
	HaggleSuccessText  string
	HaggleFailureText  string
	ReturnLabel        string
	CloseLabel         string
	Active             *bool
}

func (in PacketInput) normalized() (PacketInput, error) {
	var out PacketInput
	var err error
	if out.DisplayName, err = boundedText(in.DisplayName, maxPacketLabelLength, "display_name"); err != nil {
		return PacketInput{}, err
	}
	if out.DisplayName == "" {
		return PacketInput{}, errors.New("display_name_required")
	}
	if out.IntroText, err = boundedText(in.IntroText, maxPacketTextLength, "intro_text"); err != nil {
		return PacketInput{}, err
	}
	if out.HaggleSuccessText, err = boundedText(in.HaggleSuccessText, maxPacketTextLength, "haggle_success_text"); err != nil {
		return PacketInput{}, err
	}
	if out.HaggleFailureText, err = boundedText(in.HaggleFailureText, maxPacketTextLength, "haggle_failure_text"); err != nil {
		return PacketInput{}, err
	}
	if out.ReturnLabel, err = boundedText(in.ReturnLabel, maxPacketLabelLength, "return_label"); err != nil {
		return PacketInput{}, err
	}
	if out.CloseLabel, err = boundedText(in.CloseLabel, maxPacketLabelLength, "close_label"); err != nil {
		return PacketInput{}, err
	}
	if out.ReturnLabel == "" {
		out.ReturnLabel = "Return to Conversation"
	}
	if out.CloseLabel == "" {
		out.CloseLabel = "Leave"
	}
	if out.StanceDispositions, err = normalizeStances(in.StanceDispositions); err != nil {
		return PacketInput{}, err
	}
	out.Active = in.Active
	return out, nil
}

// CreatePacket authors a new merchant at locationID.
func CreatePacket(ctx context.Context, pool *pgxpool.Pool, locationID, actorUserID string, in PacketInput) (MerchantPacket, error) {
	clean, err := in.normalized()
	if err != nil {
		return MerchantPacket{}, err
	}
	slug := slugify(clean.DisplayName)
	if !slugPattern.MatchString(slug) {
		return MerchantPacket{}, errors.New("display_name_unusable_as_slug")
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM merchant_packets WHERE location_id = $1 AND slug = $2)
	`, locationID, slug).Scan(&exists); err != nil {
		return MerchantPacket{}, err
	}
	if exists {
		return MerchantPacket{}, errors.New("merchant_slug_taken")
	}

	dispositions, err := json.Marshal(clean.StanceDispositions)
	if err != nil {
		return MerchantPacket{}, err
	}
	active := true
	if clean.Active != nil {
		active = *clean.Active
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO merchant_packets (
			location_id, slug, display_name, intro_text, stance_dispositions_json,
			haggle_success_text, haggle_failure_text, return_label, close_label,
			active, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, NULLIF($11, '')::uuid)
		RETURNING `+merchantPacketColumns,
		locationID, slug, clean.DisplayName, clean.IntroText, dispositions,
		clean.HaggleSuccessText, clean.HaggleFailureText, clean.ReturnLabel, clean.CloseLabel,
		active, strings.TrimSpace(actorUserID))
	return scanMerchantPacket(row)
}

// UpdatePacket rewrites a packet's authored text. The slug is never touched
// -- see slugify's note.
func UpdatePacket(ctx context.Context, pool *pgxpool.Pool, packetID string, in PacketInput) (MerchantPacket, error) {
	clean, err := in.normalized()
	if err != nil {
		return MerchantPacket{}, err
	}
	dispositions, err := json.Marshal(clean.StanceDispositions)
	if err != nil {
		return MerchantPacket{}, err
	}
	active := true
	if clean.Active != nil {
		active = *clean.Active
	}
	row := pool.QueryRow(ctx, `
		UPDATE merchant_packets
		SET display_name = $2, intro_text = $3, stance_dispositions_json = $4::jsonb,
		    haggle_success_text = $5, haggle_failure_text = $6,
		    return_label = $7, close_label = $8, active = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING `+merchantPacketColumns,
		packetID, clean.DisplayName, clean.IntroText, dispositions,
		clean.HaggleSuccessText, clean.HaggleFailureText,
		clean.ReturnLabel, clean.CloseLabel, active)
	return scanMerchantPacket(row)
}

// SetPacketStock replaces a packet's inventory with exactly the given
// equipment items, in the given order.
//
// Replace-not-merge is deliberate: a Director editing a stock list is
// looking at the whole list, and "remove" has to mean something. The
// same-location check is the §9.1 guarantee -- an equipment id belonging to
// another Location is refused rather than copied in, so there is exactly
// one corpus.
func SetPacketStock(ctx context.Context, pool *pgxpool.Pool, packetID, locationID string, equipmentItemIDs []string) error {
	seen := map[string]bool{}
	ordered := []string{}
	for _, id := range equipmentItemIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ordered = append(ordered, id)
	}

	if len(ordered) > 0 {
		var validCount int
		if err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM equipment_items
			WHERE location_id = $1 AND id = ANY($2::uuid[])
		`, locationID, ordered).Scan(&validCount); err != nil {
			return err
		}
		if validCount != len(ordered) {
			return errors.New("equipment_item_not_found")
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM merchant_packet_equipment_items WHERE packet_id = $1`, packetID); err != nil {
		return err
	}
	for i, id := range ordered {
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_packet_equipment_items (packet_id, equipment_item_id, sort_order)
			VALUES ($1, $2, $3)
		`, packetID, id, i+1); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListPacketsForLocation lists every authored merchant at a Location with
// its current stock. Director-facing only.
func ListPacketsForLocation(ctx context.Context, pool *pgxpool.Pool, locationID string) ([]MerchantPacket, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+merchantPacketColumns+`
		FROM merchant_packets WHERE location_id = $1 ORDER BY display_name
	`, locationID)
	if err != nil {
		return nil, err
	}
	packets := []MerchantPacket{}
	for rows.Next() {
		p, err := scanMerchantPacket(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		packets = append(packets, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range packets {
		full, err := LoadPacketBySlug(ctx, pool, locationID, packets[i].Slug)
		if err != nil {
			return nil, err
		}
		packets[i].Stock = full.Stock
	}
	return packets, nil
}

func loadPacketLocation(ctx context.Context, pool *pgxpool.Pool, packetID string) (string, error) {
	packetID = strings.TrimSpace(packetID)
	if packetID == "" {
		return "", errors.New("merchant_packet_not_found")
	}
	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT location_id::text FROM merchant_packets WHERE id = $1
	`, packetID).Scan(&locationID); err != nil {
		return "", errors.New("merchant_packet_not_found")
	}
	return locationID, nil
}

// --- HTTP -----------------------------------------------------------------

func decodePacketBody(r *http.Request) (PacketInput, []string, bool, error) {
	var body struct {
		DisplayName        string                       `json:"display_name"`
		IntroText          string                       `json:"intro_text"`
		StanceDispositions map[string]StanceDisposition `json:"stance_dispositions"`
		HaggleSuccessText  string                       `json:"haggle_success_text"`
		HaggleFailureText  string                       `json:"haggle_failure_text"`
		ReturnLabel        string                       `json:"return_label"`
		CloseLabel         string                       `json:"close_label"`
		Active             *bool                        `json:"active"`
		EquipmentItemIDs   *[]string                    `json:"equipment_item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return PacketInput{}, nil, false, errors.New("invalid_body")
	}
	in := PacketInput{
		DisplayName:        body.DisplayName,
		IntroText:          body.IntroText,
		StanceDispositions: body.StanceDispositions,
		HaggleSuccessText:  body.HaggleSuccessText,
		HaggleFailureText:  body.HaggleFailureText,
		ReturnLabel:        body.ReturnLabel,
		CloseLabel:         body.CloseLabel,
		Active:             body.Active,
	}
	if body.EquipmentItemIDs == nil {
		return in, nil, false, nil
	}
	return in, *body.EquipmentItemIDs, true, nil
}

// HandleShowMerchantPackets handles GET and POST
// /api/shows/{show_id}/merchant-packets.
//
// Show-scoped rather than location-scoped in the URL because that is what
// a Director in a live venue has in hand; the Show resolves to its Show
// Run's Location, which is the packets' actual scope.
func HandleShowMerchantPackets(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		locationID, err := requireShowDirector(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			packets, err := ListPacketsForLocation(ctx, pool, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			// The canonical corpus, read through the same helper the
			// existing equipment editor uses -- §9.1's "reuse it, do not
			// create a replacement catalog" is satisfied by there being no
			// second query, not just no second table.
			catalog, err := ListEquipmentItems(ctx, pool, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"packets": packets, "catalog": catalog})

		case http.MethodPost:
			in, stock, hasStock, err := decodePacketBody(r)
			if err != nil {
				writeError(w, err)
				return
			}
			packet, err := CreatePacket(ctx, pool, locationID, userID, in)
			if err != nil {
				writeError(w, err)
				return
			}
			if hasStock {
				if err := SetPacketStock(ctx, pool, packet.ID, locationID, stock); err != nil {
					writeError(w, err)
					return
				}
			}
			full, err := LoadPacketBySlug(ctx, pool, locationID, packet.Slug)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"packet": full})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleMerchantPacketByID handles PATCH /api/merchant-packets/{packet_id}.
//
// Authority is resolved from the STORED packet's own Location, never from a
// client-supplied Show -- a Director at one Location cannot edit another
// Location's merchant by naming their own Show in the URL.
func HandleMerchantPacketByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		packetID := strings.TrimSpace(r.PathValue("packet_id"))
		locationID, err := loadPacketLocation(ctx, pool, packetID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := canManageEquipment(ctx, pool, userID, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}

		in, stock, hasStock, err := decodePacketBody(r)
		if err != nil {
			writeError(w, err)
			return
		}
		packet, err := UpdatePacket(ctx, pool, packetID, in)
		if err != nil {
			writeError(w, err)
			return
		}
		if hasStock {
			if err := SetPacketStock(ctx, pool, packetID, locationID, stock); err != nil {
				writeError(w, err)
				return
			}
		}
		full, err := LoadPacketBySlug(ctx, pool, locationID, packet.Slug)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"packet": full})
	}
}
