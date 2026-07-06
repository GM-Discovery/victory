package network

import "testing"

func TestFieldSetTextFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		ok   bool
	}{
		{name: "explicit set", args: []string{"set", "Mercy is for me."}, want: "Mercy is for me.", ok: true},
		{name: "explicit set split tail", args: []string{"set", "Mercy", "is", "for", "me."}, want: "Mercy is for me.", ok: true},
		{name: "natural shorthand", args: []string{"Mercy is for me."}, want: "Mercy is for me.", ok: true},
		{name: "natural shorthand split", args: []string{"Mercy", "is", "for", "me."}, want: "Mercy is for me.", ok: true},
		{name: "empty", args: nil, ok: false},
		{name: "set without body", args: []string{"set", " "}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := fieldSetTextFromArgs(tt.args)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("text = %q, want %q", got, tt.want)
			}
		})
	}
}
