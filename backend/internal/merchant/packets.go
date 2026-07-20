package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const merchantPacketColumns = `
	id::text, location_id::text, slug, display_name,
	COALESCE(portrait_asset_id::text, ''), intro_text, stance_dispositions_json,
	haggle_skill_key, haggle_target_value, haggle_skilled_die, haggle_unskilled_die,
	haggle_success_text, haggle_failure_text, return_label, close_label,
	active, created_at, updated_at
`

func scanMerchantPacket(row pgx.Row) (MerchantPacket, error) {
	var p MerchantPacket
	var dispositionsRaw []byte
	if err := row.Scan(
		&p.ID, &p.LocationID, &p.Slug, &p.DisplayName,
		&p.PortraitAssetID, &p.IntroText, &dispositionsRaw,
		&p.HaggleSkillKey, &p.HaggleTargetValue, &p.HaggleSkilledDie, &p.HaggleUnskilledDie,
		&p.HaggleSuccessText, &p.HaggleFailureText, &p.ReturnLabel, &p.CloseLabel,
		&p.Active, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MerchantPacket{}, errors.New("merchant_packet_not_found")
		}
		return MerchantPacket{}, err
	}
	_ = json.Unmarshal(dispositionsRaw, &p.StanceDispositions)
	return p, nil
}

// LoadPacketBySlug loads a merchant packet plus its active stock (equipment
// available for purchase), ordered as authored. The Kessa packet itself is
// seeded via migration (061_kernel73_participant_interactions.sql) rather
// than built through a full editor -- kernel-73 spec S11 explicitly allows
// "a bounded form or seed configuration" and warns against a general
// node-graph dialogue editor.
func LoadPacketBySlug(ctx context.Context, pool *pgxpool.Pool, locationID, slug string) (MerchantPacket, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return MerchantPacket{}, errors.New("merchant_packet_not_found")
	}
	row := pool.QueryRow(ctx, `
		SELECT `+merchantPacketColumns+`
		FROM merchant_packets
		WHERE location_id = $1 AND slug = $2
	`, locationID, slug)
	p, err := scanMerchantPacket(row)
	if err != nil {
		return MerchantPacket{}, err
	}

	rows, err := pool.Query(ctx, `
		SELECT `+equipmentItemColumns+`
		FROM merchant_packet_equipment_items mpi
		JOIN equipment_items e ON e.id = mpi.equipment_item_id
		WHERE mpi.packet_id = $1 AND e.active = TRUE
		ORDER BY mpi.sort_order ASC
	`, p.ID)
	if err != nil {
		return MerchantPacket{}, err
	}
	defer rows.Close()
	for rows.Next() {
		e, err := scanEquipmentItem(rows)
		if err != nil {
			return MerchantPacket{}, err
		}
		p.Stock = append(p.Stock, e)
	}
	if err := rows.Err(); err != nil {
		return MerchantPacket{}, err
	}
	return p, nil
}
