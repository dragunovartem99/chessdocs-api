package docs

import "testing"

func TestEditable(t *testing.T) {
	cases := map[string]struct {
		source string
		want   bool
	}{
		"no frontmatter":            {"# Blunder\n\nA bad move.", true},
		"frontmatter without flags": {"---\nlayout: home\n---\n# Home", true},
		"editLink true":             {"---\neditLink: true\n---\n# Opening", true},
		"editLink false":            {"---\nlayout: home\neditLink: false\n---\n# Home", false},
		"dev true":                  {"---\ndev: true\n---\n# PGN Editor", false},
		"carriage returns":          {"---\r\neditLink: false\r\n---\r\n# Home", false},
		"flag only in the prose":    {"# Title\n\nExample: `editLink: false`", true},
		"flag after the block":      {"---\nlayout: home\n---\ndev: true\n", true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := Editable(tc.source); got != tc.want {
				t.Errorf("Editable() = %v, want %v", got, tc.want)
			}
		})
	}
}
