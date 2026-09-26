package api

import (
	"io"
	"net/http"

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
