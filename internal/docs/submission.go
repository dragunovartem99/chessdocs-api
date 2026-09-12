// Package docs models the chessdocs contribution domain: the edit submissions
// readers send in, and the rules deciding which pages accept them. It is pure
// data and validation — nothing here talks to the network.
package docs

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

// DefaultLang is applied to submissions that omit lang.
const DefaultLang = "en"

// Field limits, mirrored from openapi.yaml. contract_test.go fails if the two
// ever drift apart.
const (
	MaxTitleLen   = 120
	MaxContentLen = 4000
	MaxNameLen    = 120
	MaxContactLen = 120
)

// Submission is a reader's proposed replacement for one docs page.
type Submission struct {
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Author     *Author `json:"author,omitempty"`
	Lang       string  `json:"lang,omitempty"`
	SourcePath string  `json:"sourcePath"`
}

// Author is the optional attribution a submitter may attach to their edit.
type Author struct {
	Name    string `json:"name,omitempty"`
	Contact string `json:"contact,omitempty"`
}

// Normalize trims the submitted text and fills in defaults. Call it before
// Validate: the limits below apply to the stored values, not to the padding
// around them.
func (s *Submission) Normalize() {
	s.Title = strings.TrimSpace(s.Title)
	s.Content = strings.TrimSpace(s.Content)
	s.SourcePath = strings.TrimSpace(s.SourcePath)

	if s.Lang == "" {
		s.Lang = DefaultLang
	}
	if s.Author != nil {
		s.Author.Name = strings.TrimSpace(s.Author.Name)
		s.Author.Contact = strings.TrimSpace(s.Author.Contact)
	}
}

// Validate reports the first way the submission breaks the published contract.
// Every message it returns is safe to show the submitter.
func (s *Submission) Validate() error {
	if err := validateText("title", s.Title, MaxTitleLen); err != nil {
		return err
	}
	if err := validateText("content", s.Content, MaxContentLen); err != nil {
		return err
	}
	if err := ValidateSourcePath("sourcePath", s.SourcePath); err != nil {
		return err
	}
	if !langPattern.MatchString(s.Lang) {
		return invalid("lang must be a two-letter language code")
	}
	if s.Author == nil {
		return nil
	}
	if utf8.RuneCountInString(s.Author.Name) > MaxNameLen {
		return invalid("author.name must be at most %d characters", MaxNameLen)
	}
	if contact := s.Author.Contact; contact != "" {
		if utf8.RuneCountInString(contact) > MaxContactLen {
			return invalid("author.contact must be at most %d characters", MaxContactLen)
		}
		if !contactPattern.MatchString(contact) {
			return invalid("author.contact must be an email address")
		}
	}
	return nil
}

// ValidateSourcePath checks a docs-relative markdown path, naming the offending
// field so the same rule can back both the query parameter and the body field.
func ValidateSourcePath(field, path string) error {
	if path == "" {
		return invalid("%s is required", field)
	}
	if !sourcePathPattern.MatchString(path) {
		return invalid("%s must be a docs-relative markdown path, e.g. en/glossary/fork.md", field)
	}
	return nil
}

func validateText(field, value string, max int) error {
	if value == "" {
		return invalid("%s must not be blank", field)
	}
	if utf8.RuneCountInString(value) > max {
		return invalid("%s must be at most %d characters", field, max)
	}
	return nil
}

// Error is a failure that is safe to report verbatim: it carries both the
// status the API should answer with and the exact message to send.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string         { return e.Message }
func (e *Error) HTTPStatus() int       { return e.Status }
func (e *Error) PublicMessage() string { return e.Message }

// ErrNotEditable rejects pages whose frontmatter opts out of editing.
var ErrNotEditable = &Error{Status: http.StatusForbidden, Message: "This page is not editable"}

func invalid(format string, args ...any) *Error {
	return &Error{Status: http.StatusBadRequest, Message: fmt.Sprintf(format, args...)}
}
