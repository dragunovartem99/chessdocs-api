package docs

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	sub := Submission{
		Title:      "  Title  ",
		Content:    "  Body\n",
		Author:     &Author{Name: " Artem ", Contact: " a@b.cd "},
		SourcePath: "ru/glossary/fork.md",
	}

	sub.Normalize()

	want := Submission{
		Title:      "Title",
		Content:    "Body",
		Author:     &Author{Name: "Artem", Contact: "a@b.cd"},
		Lang:       DefaultLang,
		SourcePath: "ru/glossary/fork.md",
	}
	if sub.Title != want.Title || sub.Content != want.Content || sub.Lang != want.Lang {
		t.Errorf("Normalize() = %+v, want %+v", sub, want)
	}
	if *sub.Author != *want.Author {
		t.Errorf("author = %+v, want %+v", sub.Author, want.Author)
	}
}

func TestNormalizeKeepsAnExplicitLanguage(t *testing.T) {
	sub := Submission{Lang: "ru"}
	sub.Normalize()

	if sub.Lang != "ru" {
		t.Errorf("lang = %q, want it untouched", sub.Lang)
	}
}

func TestNormalizeLeavesAnAbsentAuthorAbsent(t *testing.T) {
	sub := Submission{Title: "Title", Content: "Body", SourcePath: "en/glossary/fork.md"}
	sub.Normalize()

	if sub.Author != nil {
		t.Errorf("author = %+v, want nil", sub.Author)
	}
}

func valid() Submission {
	return Submission{
		Title:      "Clarify the example",
		Content:    "# Fork\n\nA double attack.",
		Lang:       DefaultLang,
		SourcePath: "en/glossary/fork.md",
	}
}

func TestValidateAcceptsAWellFormedSubmission(t *testing.T) {
	sub := valid()
	sub.Author = &Author{Name: "Artem", Contact: "a@b.cd"}

	if err := sub.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Submission){
		"a blank title":       func(s *Submission) { s.Title = "" },
		"an overlong title":   func(s *Submission) { s.Title = strings.Repeat("x", MaxTitleLen+1) },
		"blank content":       func(s *Submission) { s.Content = "" },
		"overlong content":    func(s *Submission) { s.Content = strings.Repeat("x", MaxContentLen+1) },
		"a missing path":      func(s *Submission) { s.SourcePath = "" },
		"a traversing path":   func(s *Submission) { s.SourcePath = "en/../../etc/passwd.md" },
		"a non-markdown path": func(s *Submission) { s.SourcePath = "en/notes.txt" },
		"a rootless path":     func(s *Submission) { s.SourcePath = "glossary/fork.md" },
		"an unknown language": func(s *Submission) { s.Lang = "english" },
		"a malformed contact": func(s *Submission) { s.Author = &Author{Contact: "not-an-email"} },
		"an overlong contact": func(s *Submission) { s.Author = &Author{Contact: strings.Repeat("x", MaxContactLen) + "@b.cd"} },
		"an overlong name":    func(s *Submission) { s.Author = &Author{Name: strings.Repeat("x", MaxNameLen+1)} },
	}

	for name, break_ := range cases {
		t.Run(name, func(t *testing.T) {
			sub := valid()
			break_(&sub)

			err := sub.Validate()
			if err == nil {
				t.Fatal("Validate() = nil, want an error")
			}

			var invalid *Error
			if !errors.As(err, &invalid) || invalid.Status != 400 {
				t.Fatalf("Validate() = %v, want a 400", err)
			}
			if invalid.Message == "" {
				t.Error("the submitter was given no reason")
			}
		})
	}
}

func TestValidateAcceptsAnEmptyContact(t *testing.T) {
	sub := valid()
	sub.Author = &Author{Name: "Artem"}

	if err := sub.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil: contact is optional", err)
	}
}
