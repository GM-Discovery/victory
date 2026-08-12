package drawing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/rollaudience"
	"victory/backend/internal/venuecoordination"
)

// StageVenueSessionID is the venuecoordination session identity for the
// live main stage: the Session ID itself, mirroring
// storyboards.StoryboardVenueSessionID's board_id keying (Kernel 83's
// generic Registry is shared process-wide across venue types; different
// UUID spaces never collide). Turn/Leader drawing authority (kernel §9)
// reuses this exact Kernel 83 infrastructure rather than building new
// initiative/turn state.
func StageVenueSessionID(sessionID string) string { return strings.TrimSpace(sessionID) }

// LoadSettings returns showID's drawing-mode/measurement settings, or the
// documented defaults if the Show has never had settings written (kernel
// §7 "at minimum document the initial policy" -- director_only /
// alternating_1_2 / 1 square = 5 ft).
func LoadSettings(ctx context.Context, pool *pgxpool.Pool, showID string) (Settings, error) {
	s := Settings{
		ShowID:         showID,
		DrawingMode:    ModeDirectorOnly,
		ScaleGridUnits: 1,
		ScaleRealUnits: 5,
		ScaleUnitLabel: "ft",
		DiagonalPolicy: DiagonalAlternating,
	}
	var calib []byte
	err := pool.QueryRow(ctx, `
		SELECT drawing_mode, scale_grid_units, scale_real_units, scale_unit_label,
		       diagonal_policy, gridless_calibration
		FROM stage_drawing_settings WHERE show_id = $1
	`, showID).Scan(&s.DrawingMode, &s.ScaleGridUnits, &s.ScaleRealUnits, &s.ScaleUnitLabel,
		&s.DiagonalPolicy, &calib)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, nil
		}
		return s, err
	}
	if len(calib) > 0 {
		s.GridlessCalibration = map[string]any{}
		_ = json.Unmarshal(calib, &s.GridlessCalibration)
	}
	return s, nil
}

// SaveSettings upserts showID's settings. Only Director+ may call this
// (checked by the HTTP handler before calling in, matching cohorts/http.go
// and storyboards/http.go's convention of authority-in-the-domain-call).
func SaveSettings(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, showID string, s Settings) (Settings, error) {
	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, actorUserID)
	if err != nil {
		return Settings{}, err
	}
	if !isDirector {
		return Settings{}, errors.New("not_authorized")
	}
	if !ValidDrawingModes[s.DrawingMode] {
		return Settings{}, errors.New("invalid_drawing_mode")
	}
	if !ValidDiagonalPolicies[s.DiagonalPolicy] {
		return Settings{}, errors.New("invalid_diagonal_policy")
	}
	if s.ScaleGridUnits <= 0 || s.ScaleRealUnits <= 0 {
		return Settings{}, errors.New("invalid_scale")
	}
	label := strings.TrimSpace(s.ScaleUnitLabel)
	if label == "" || len(label) > 24 {
		return Settings{}, errors.New("invalid_scale_unit_label")
	}
	var calibJSON []byte
	if s.GridlessCalibration != nil {
		calibJSON, _ = json.Marshal(s.GridlessCalibration)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO stage_drawing_settings (
			show_id, drawing_mode, scale_grid_units, scale_real_units, scale_unit_label,
			diagonal_policy, gridless_calibration, updated_by_user_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (show_id) DO UPDATE SET
			drawing_mode = EXCLUDED.drawing_mode,
			scale_grid_units = EXCLUDED.scale_grid_units,
			scale_real_units = EXCLUDED.scale_real_units,
			scale_unit_label = EXCLUDED.scale_unit_label,
			diagonal_policy = EXCLUDED.diagonal_policy,
			gridless_calibration = EXCLUDED.gridless_calibration,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`, showID, s.DrawingMode, s.ScaleGridUnits, s.ScaleRealUnits, label,
		s.DiagonalPolicy, calibJSON, actorUserID)
	if err != nil {
		return Settings{}, err
	}
	s.ShowID = showID
	s.ScaleUnitLabel = label
	return s, nil
}

// isEligibleParticipant reports whether userID is a non-audience
// participant of sessionID -- the Freeform-mode eligibility bar (kernel
// §9 "all eligible participants in current drawing scope").
func isEligibleParticipant(ctx context.Context, pool *pgxpool.Pool, sessionID, userID string) (bool, error) {
	var role string
	err := pool.QueryRow(ctx, `
		SELECT role::text FROM session_participants WHERE session_id = $1 AND user_id = $2
	`, sessionID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return strings.ToLower(strings.TrimSpace(role)) != "audience", nil
}

// CanDraw enforces kernel §9's three drawing-authority modes for creating
// a new object. Director+ can always draw regardless of mode (kernel §1
// "Director+ can always draw").
func CanDraw(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, sessionID, showID, userID string) (bool, error) {
	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, userID)
	if err != nil {
		return false, err
	}
	if isDirector {
		return true, nil
	}

	settings, err := LoadSettings(ctx, pool, showID)
	if err != nil {
		return false, err
	}

	switch settings.DrawingMode {
	case ModeDirectorOnly:
		return false, nil
	case ModeTurnLeader:
		if reg == nil {
			return false, nil
		}
		reg.EnsureSession(StageVenueSessionID(sessionID), "stage", showID, "")
		state, ok := reg.Get(StageVenueSessionID(sessionID))
		if !ok {
			return false, nil
		}
		return (state.GroupLeaderUserID != "" && state.GroupLeaderUserID == userID) ||
			(state.CurrentTurnUserID != "" && state.CurrentTurnUserID == userID), nil
	case ModeFreeform:
		return isEligibleParticipant(ctx, pool, sessionID, userID)
	default:
		return false, nil
	}
}

// CanEditObject enforces kernel §1 editing-ownership rules: the creator
// may edit/delete their own unlocked object; Director+ may edit/delete/
// unlock/reassign anything, locked or not.
func CanEditObject(ctx context.Context, pool *pgxpool.Pool, sessionID, userID string, obj Object) (bool, error) {
	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, userID)
	if err != nil {
		return false, err
	}
	if isDirector {
		return true, nil
	}
	if obj.Locked {
		return false, nil
	}
	return obj.CreatorUserID == userID, nil
}
