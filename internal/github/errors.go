package github

import (
	"io"
	"net/http"
	"strings"
)

// UpstreamError reports a GitHub call that did not succeed. Summary is written
// for the submitter and says nothing about our credentials or the repository;
// Detail carries the upstream response and belongs in the log only.
type UpstreamError struct {
	Summary string
	Detail  string
}

func (e *UpstreamError) Error() string {
	if e.Detail == "" {
		return e.Summary
	}
	return e.Summary + ": " + e.Detail
}

// HTTPStatus and PublicMessage let the HTTP layer answer without having to
// know that GitHub exists.
func (e *UpstreamError) HTTPStatus() int { return http.StatusBadGateway }

func (e *UpstreamError) PublicMessage() string { return e.Summary }

func snippet(r io.Reader) string {
	body, err := io.ReadAll(io.LimitReader(r, detailLimit))
	if err != nil {
		return err.Error()
	}
	return strings.TrimSpace(string(body))
}
