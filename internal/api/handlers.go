package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

// getSource returns the current markdown of a docs page so the browser can
// open it in the editor.
func (s *server) getSource(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if err := docs.ValidateSourcePath("path", path); err != nil {
		s.fail(w, r, err)
		return
	}

	source, err := s.github.FetchSource(r.Context(), path)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !docs.Editable(source) {
		s.fail(w, r, docs.ErrNotEditable)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, source)
}

// createSubmission turns an edited page into a pull request. The page's
// current source is fetched first: editability lives in the frontmatter, and
// only the base branch is authoritative about it.
func (s *server) createSubmission(w http.ResponseWriter, r *http.Request) {
	sub, err := decodeSubmission(w, r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	sub.Normalize()
	if err := sub.Validate(); err != nil {
		s.fail(w, r, err)
		return
	}

	source, err := s.github.FetchSource(r.Context(), sub.SourcePath)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !docs.Editable(source) {
		s.fail(w, r, docs.ErrNotEditable)
		return
	}

	url, err := s.github.OpenPullRequest(r.Context(), sub)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}

// notFound backs every request the two routes above do not claim, so that even
// a stray URL answers in the documented error shape.
func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	writeError(w, http.StatusNotFound, "Not found")
}

// decodeSubmission reads the request body into a Submission, translating the
// ways that can go wrong into messages a submitter can act on.
func decodeSubmission(w http.ResponseWriter, r *http.Request) (docs.Submission, error) {
	var sub docs.Submission

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sub); err != nil {
		return sub, decodeError(err)
	}
	// A second JSON value in the body means the client sent something other
	// than the single object the contract describes.
	if decoder.More() {
		return sub, badRequest("Request body must be a single JSON object")
	}
	return sub, nil
}

func decodeError(err error) error {
	var maxBytes *http.MaxBytesError
	var syntax *json.SyntaxError
	var unmarshal *json.UnmarshalTypeError

	switch {
	case errors.As(err, &maxBytes):
		return &docs.Error{
			Status:  http.StatusRequestEntityTooLarge,
			Message: "Request body is too large",
		}
	case errors.Is(err, io.EOF):
		return badRequest("Request body is empty")
	case errors.As(err, &syntax):
		return badRequest("Request body is not valid JSON")
	case errors.As(err, &unmarshal):
		return badRequest("Field %q has the wrong type", unmarshal.Field)
	default:
		// The decoder reports unknown fields only as a formatted string.
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return badRequest("Unknown field %s", field)
		}
		return badRequest("Request body is not valid JSON")
	}
}

func badRequest(format string, args ...any) error {
	return &docs.Error{Status: http.StatusBadRequest, Message: fmt.Sprintf(format, args...)}
}
