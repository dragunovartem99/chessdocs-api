package github

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Clarify the example":   "clarify-the-example",
		"  Spaces  everywhere ": "spaces-everywhere",
		"Rook + Pawn = win!":    "rook-pawn-win",
		"Опечатка":              "submission",
		"":                      "submission",
		"---":                   "submission",
	}

	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}
