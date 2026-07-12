package showruns

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/playerprofile"
)

// roleDisplayLabel is the single place that maps a stored role value to its
// display string. All HTTP responses and UI copy funnel through this so the
// "say Player, not Cast" rule (Kernel 66) can't be violated by a stray call
// site formatting the role differently.
func roleDisplayLabel(role, customRoleLabel string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer":
		return "Producer"
	case "director":
		return "Director"
	case "player":
		return "Player"
	case "crew":
		return "Crew"
	case "audience":
		return "Audience"
	case "guest":
		return "Guest"
	case "observer":
		return "Observer"
	case "custom":
		if strings.TrimSpace(customRoleLabel) != "" {
			return customRoleLabel
		}
		return "Custom"
	default:
		return role
	}
}

// ProjectRosterMember builds the live, viewer-scoped internal-roster shape
// of one roster row, mirroring thirdplace.ProjectHeadshot's pattern exactly:
// stage name, portrait, and headline facts are re-derived from the member's
// *current* Trailer Face on every call via playerprofile.ProjectTrailerFace
// -- never read off the roster row, because no Face content is ever stored
// there.
func ProjectRosterMember(ctx context.Context, pool *pgxpool.Pool, viewerUserID string, member RosterMember) (RosterMemberProjection, error) {
	wb, err := playerprofile.EnsureWorkbook(ctx, pool, member.UserID)
	if err != nil {
		return RosterMemberProjection{}, err
	}
	face, err := playerprofile.ProjectTrailerFace(ctx, pool, member.UserID)
	if err != nil {
		return RosterMemberProjection{}, err
	}

	proj := RosterMemberProjection{
		MemberID:       member.ID,
		ProfileID:      wb.ID,
		Role:           member.Role,
		RoleLabel:      roleDisplayLabel(member.Role, member.CustomRoleLabel),
		ProgramVisible: member.ProgramVisible,
		AddedAt:        member.AddedAt,
		StageName:      face.StageName,
		HeadlineFacts:  []HeadlineFact{},
		TrailerURL:     "/venues/trailers/view.html?id=" + wb.ID,
		IsYou:          strings.TrimSpace(viewerUserID) == member.UserID,
	}

	for _, f := range face.Regions["identity_header"] {
		if f.FieldKey == "portrait_url" {
			proj.PortraitURL = f.DisplayValue
			break
		}
	}

	glance := face.Regions["at_a_glance"]
	limit := 3
	if len(glance) < limit {
		limit = len(glance)
	}
	for _, f := range glance[:limit] {
		proj.HeadlineFacts = append(proj.HeadlineFacts, HeadlineFact{Label: f.Label, Value: f.DisplayValue})
	}

	return proj, nil
}

// ProjectAudienceProgramEntry wraps ProjectRosterMember and trims the result
// to the curated Audience-facing subset -- it must not duplicate the Face
// read, only narrow what's returned (Kernel 66 "Audience does not see the
// full internal roster by default").
func ProjectAudienceProgramEntry(ctx context.Context, pool *pgxpool.Pool, viewerUserID string, member RosterMember) (AudienceProgramEntry, error) {
	full, err := ProjectRosterMember(ctx, pool, viewerUserID, member)
	if err != nil {
		return AudienceProgramEntry{}, err
	}
	return AudienceProgramEntry{
		Role:          full.Role,
		RoleLabel:     full.RoleLabel,
		StageName:     full.StageName,
		PortraitURL:   full.PortraitURL,
		HeadlineFacts: full.HeadlineFacts,
		TrailerURL:    full.TrailerURL,
		IsYou:         full.IsYou,
	}, nil
}
