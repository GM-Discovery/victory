package profiles

import "testing"

func TestSanitizeProfileRequest(t *testing.T) {
	in := profileRequest{
		UserID:              "  user-1  ",
		StageName:           "  Stage Name  ",
		Pronouns:            "  they/them ",
		HeadshotURL:         "  data:image/png;base64,abc  ",
		PerformanceAgeRange: "  Mid 30s to Mid 50s ",
		Bio:                 "  Bio  ",
		Credits:             "  Credits  ",
		Skills:              "  Skills  ",
		Availability:        "  Evenings  ",
		PublicLinks:         "  https://example.com  ",
	}

	got := sanitizeProfileRequest(in)
	if got.UserID != "user-1" || got.StageName != "Stage Name" || got.Pronouns != "they/them" {
		t.Fatalf("sanitizeProfileRequest trimmed values incorrectly: %+v", got)
	}
	if got.HeadshotURL != "data:image/png;base64,abc" || got.PublicLinks != "https://example.com" {
		t.Fatalf("sanitizeProfileRequest did not trim url fields: %+v", got)
	}
}

func TestProfileStateAndProjection(t *testing.T) {
	profile := Profile{
		Draft: PublicFields{
			StageName: "Draft Name",
		},
		Published: PublicFields{
			StageName: "Published Name",
		},
	}

	if got := profileState(profile); got != "draft" {
		t.Fatalf("expected draft state, got %q", got)
	}

	if got := publishedProjection(profile); got.StageName != "" || got.DisplayName != "" || len(got.History) != 0 {
		t.Fatalf("expected empty public projection until publish, got %+v", got)
	}

	profile.IsPublished = true
	if got := profileState(profile); got != "published" {
		t.Fatalf("expected published state, got %q", got)
	}

	if got := publishedProjection(profile); got.StageName != "Published Name" {
		t.Fatalf("expected published projection to surface published data, got %+v", got)
	}
}
