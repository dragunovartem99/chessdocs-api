package docs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// openapi.yaml is the published contract for this API, and the validation in
// this package is a hand-written implementation of it. These tests re-read the
// document and fail if the two ever describe different rules — the drift that
// a generated validator would have made impossible.

type schema struct {
	Type       string            `yaml:"type"`
	Pattern    string            `yaml:"pattern"`
	MaxLength  int               `yaml:"maxLength"`
	Default    string            `yaml:"default"`
	Const      *string           `yaml:"const"`
	Ref        string            `yaml:"$ref"`
	Required   []string          `yaml:"required"`
	Properties map[string]schema `yaml:"properties"`
	AnyOf      []schema          `yaml:"anyOf"`
}

func contract(t *testing.T) map[string]schema {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read the contract: %v", err)
	}

	var document struct {
		Components struct {
			Schemas map[string]schema `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatalf("parse the contract: %v", err)
	}
	return document.Components.Schemas
}

func TestContractSourcePath(t *testing.T) {
	if got, want := contract(t)["SourcePath"].Pattern, sourcePathPattern.String(); got != want {
		t.Errorf("openapi.yaml SourcePath pattern = %q, implementation = %q", got, want)
	}
}

func TestContractSubmissionFields(t *testing.T) {
	submission := contract(t)["Submission"]

	if want := []string{"title", "content", "sourcePath"}; !slices.Equal(submission.Required, want) {
		t.Errorf("openapi.yaml requires %v, implementation requires %v", submission.Required, want)
	}

	// The contract spells "must not be blank" as a pattern requiring one
	// non-whitespace character; Validate enforces it after trimming.
	for _, field := range []string{"title", "content"} {
		if got := submission.Properties[field].Pattern; got != `\S` {
			t.Errorf("openapi.yaml %s pattern = %q, want the blank check %q", field, got, `\S`)
		}
	}

	lengths := map[string]int{"title": MaxTitleLen, "content": MaxContentLen}
	for field, want := range lengths {
		if got := submission.Properties[field].MaxLength; got != want {
			t.Errorf("openapi.yaml %s maxLength = %d, implementation = %d", field, got, want)
		}
	}

	lang := submission.Properties["lang"]
	if lang.Pattern != langPattern.String() {
		t.Errorf("openapi.yaml lang pattern = %q, implementation = %q", lang.Pattern, langPattern.String())
	}
	if lang.Default != DefaultLang {
		t.Errorf("openapi.yaml lang default = %q, implementation = %q", lang.Default, DefaultLang)
	}

	if ref := submission.Properties["sourcePath"].Ref; ref != "#/components/schemas/SourcePath" {
		t.Errorf("openapi.yaml sourcePath = %q, want the shared SourcePath schema", ref)
	}
}

func TestContractAuthorFields(t *testing.T) {
	author := contract(t)["Submission"].Properties["author"]

	if got := author.Properties["name"].MaxLength; got != MaxNameLen {
		t.Errorf("openapi.yaml author.name maxLength = %d, implementation = %d", got, MaxNameLen)
	}

	// contact is either the empty string or an email address.
	var blankAllowed bool
	var address schema
	for _, alternative := range author.Properties["contact"].AnyOf {
		if alternative.Const != nil && *alternative.Const == "" {
			blankAllowed = true
			continue
		}
		address = alternative
	}

	if !blankAllowed {
		t.Error("openapi.yaml no longer allows a blank contact, but Validate does")
	}
	if address.Pattern != contactPattern.String() {
		t.Errorf("openapi.yaml contact pattern = %q, implementation = %q", address.Pattern, contactPattern.String())
	}
	if address.MaxLength != MaxContactLen {
		t.Errorf("openapi.yaml contact maxLength = %d, implementation = %d", address.MaxLength, MaxContactLen)
	}
}
