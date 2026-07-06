package network

import (
	"reflect"
	"testing"
)

func TestParseCharAddSkillArgsPlainName(t *testing.T) {
	name, custom, err := parseCharAddSkillArgs([]string{"Animal", "Handling"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if custom != nil {
		t.Fatalf("expected no custom detail for plain name")
	}
	if name != "Animal Handling" {
		t.Fatalf("name = %q, want %q", name, "Animal Handling")
	}
}

func TestParseCharAddSkillArgsCustom(t *testing.T) {
	name, custom, err := parseCharAddSkillArgs([]string{
		"--custom", "--name", "Knife", "Juggling", "--description", "Tossing", "sharp", "things", "--attribute", "Grace",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty positional name for custom path, got %q", name)
	}
	want := &customSkillDetailWant{Name: "Knife Juggling", Description: "Tossing sharp things", AttributeName: "Grace"}
	got := &customSkillDetailWant{Name: custom.Name, Description: custom.Description, AttributeName: custom.AttributeName}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("custom = %+v, want %+v", got, want)
	}
}

type customSkillDetailWant struct {
	Name          string
	Description   string
	AttributeName string
}

func TestParseCharAddSkillArgsCustomRequiresName(t *testing.T) {
	_, _, err := parseCharAddSkillArgs([]string{"--custom", "--description", "d", "--attribute", "Grace"})
	if err == nil || err.Error() != "custom_skill_name_required" {
		t.Fatalf("err = %v, want custom_skill_name_required", err)
	}
}

func TestParseCharAddSkillArgsCustomRequiresDescription(t *testing.T) {
	_, _, err := parseCharAddSkillArgs([]string{"--custom", "--name", "n", "--attribute", "Grace"})
	if err == nil || err.Error() != "custom_skill_description_required" {
		t.Fatalf("err = %v, want custom_skill_description_required", err)
	}
}

func TestParseCharAddSkillArgsCustomRequiresAttribute(t *testing.T) {
	_, _, err := parseCharAddSkillArgs([]string{"--custom", "--name", "n", "--description", "d"})
	if err == nil || err.Error() != "attribute_required" {
		t.Fatalf("err = %v, want attribute_required", err)
	}
}

func TestParseCharAddSkillArgsRequiresArgs(t *testing.T) {
	_, _, err := parseCharAddSkillArgs(nil)
	if err == nil || err.Error() != "invalid_command_arguments" {
		t.Fatalf("err = %v, want invalid_command_arguments", err)
	}
}
