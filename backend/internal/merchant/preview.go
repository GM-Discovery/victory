package merchant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// PreviewInteractionResult is the whole surface a Director's "Preview as
// Player" is allowed to see: enough to prove a bound token would open the
// right packet, and nothing that could purchase, roll, or record
// participation. Kernel 73A item 7.
type PreviewInteractionResult struct {
	InteractionID    string   `json:"interaction_id"`
	StageButtonLabel string   `json:"stage_button_label"`
	Enabled          bool     `json:"enabled"`
	PacketSlug       string   `json:"packet_slug,omitempty"`
	PacketDisplay    string   `json:"packet_display_name,omitempty"`
	PacketIntroText  string   `json:"packet_intro_text,omitempty"`
	StockNames       []string `json:"stock_names,omitempty"`
}

// PreviewInteraction is a structurally separate, read-only code path from
// OpenEquipMode/AttemptStance/PreviewHaggle/AttemptHaggle/AttemptPurchase --
// it never calls ResolveEligibleContext (which resolves a real roster row,
// a real selected/owned Character, and a real active session), never
// touches character_cards, show_run_roster_members, or inventory tables,
// and has no code path that could reach a mutating function. Authority is
// backstage visibility (Director/Producer/Operator/Crew), the opposite
// gate from a Player's participant eligibility -- a backstage viewer using
// this to "preview as player" is never mistaken for, or able to act as, an
// actual eligible Player, and an actual Player has no reason to ever call
// this endpoint (their real path is OpenEquipMode). This is what makes
// "Preview as Player" structurally incapable of purchasing/rolling/
// recording participation even from a modified client or a direct API
// call in preview context: the mutating functions are simply never
// reachable from here, not merely "not called by this client today."
func PreviewInteraction(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (PreviewInteractionResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return PreviewInteractionResult{}, errors.New("not_authenticated")
	}
	it, err := LoadInteractionByID(ctx, pool, interactionID)
	if err != nil {
		return PreviewInteractionResult{}, err
	}
	_, _, locationID, err := placementShowShowRunLocation(ctx, pool, it.ShowScenePlacementID)
	if err != nil {
		return PreviewInteractionResult{}, err
	}
	canView, err := showruns.CanViewBackstage(ctx, pool, actorUserID, locationID)
	if err != nil {
		return PreviewInteractionResult{}, err
	}
	if !canView {
		return PreviewInteractionResult{}, errors.New("not_authorized")
	}

	result := PreviewInteractionResult{
		InteractionID:    it.ID,
		StageButtonLabel: it.StageButtonLabel,
		Enabled:          it.Enabled,
	}
	if it.InteractionType != InteractionTypeOpenEquipMode {
		return result, nil
	}
	packetSlug := packetSlugFromConfig(it.ConfigurationJSON)
	if packetSlug == "" {
		return result, nil
	}
	result.PacketSlug = packetSlug
	packet, err := LoadPacketBySlug(ctx, pool, locationID, packetSlug)
	if err != nil {
		// A misconfigured/missing packet is a legitimate preview finding
		// (the same gap KessaReachabilityDiagnostics would flag), not a
		// hard error -- report it as an empty display rather than failing
		// the whole preview request.
		return result, nil
	}
	result.PacketDisplay = packet.DisplayName
	result.PacketIntroText = packet.IntroText
	for _, item := range packet.Stock {
		result.StockNames = append(result.StockNames, item.Name)
	}
	return result, nil
}
