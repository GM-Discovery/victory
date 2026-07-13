package playerprofile

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ReasonFaceReady               = "face_ready"
	ReasonMissingFaceCommit       = "missing_face_commit"
	ReasonMissingVisibleFaceField = "missing_visible_face_field"
	ReasonNotAuthenticated        = "not_authenticated"
)

// ReadyResult is the Trailer Face readiness verdict for one user (Kernel 68
// §3.1). Third Place visibility/access is gated on Ready being true.
type ReadyResult struct {
	Ready             bool   `json:"ready"`
	ReasonCode        string `json:"reason_code"`
	VisibleFieldCount int    `json:"visible_field_count"`
	HasStageName      bool   `json:"has_stage_name"`
}

// TrailerFaceReady reports whether userID has intentionally set up a usable
// Trailer Face: a stage name plus at least one visible Face field beyond
// bare defaults (Kernel 68 §1.1, §3.1). It is computed live from the
// existing Kernel 61 projection -- ProjectTrailerFace's Regions are already
// filtered to visible-only fields (BuildTrailerFace -> VisibleSortedFields),
// so a stage name is field-presence driven with no separate durable marker
// needed.
func TrailerFaceReady(ctx context.Context, pool *pgxpool.Pool, userID string) (ReadyResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ReadyResult{ReasonCode: ReasonNotAuthenticated}, errors.New("not_authenticated")
	}

	face, err := ProjectTrailerFace(ctx, pool, userID)
	if err != nil {
		return ReadyResult{}, err
	}

	hasStageName := strings.TrimSpace(face.StageName) != ""
	visibleCount := 0
	for _, fields := range face.Regions {
		visibleCount += len(fields)
	}

	result := ReadyResult{
		HasStageName:      hasStageName,
		VisibleFieldCount: visibleCount,
	}

	switch {
	case !hasStageName:
		result.Ready = false
		result.ReasonCode = ReasonMissingFaceCommit
	case visibleCount == 0:
		result.Ready = false
		result.ReasonCode = ReasonMissingVisibleFaceField
	default:
		result.Ready = true
		result.ReasonCode = ReasonFaceReady
	}

	return result, nil
}
