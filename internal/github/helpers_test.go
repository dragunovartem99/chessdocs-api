package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stub runs a client against a local stand-in for GitHub, and records the
// requests it receives.
func stub(t *testing.T, handler http.HandlerFunc) (*Client, *[]*http.Request) {
	t.Helper()

	var seen []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		clone := r.Clone(r.Context())
		clone.Body = io.NopCloser(strings.NewReader(string(body)))
		seen = append(seen, clone)

		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client := New(
		Config{Token: "test-token", Owner: "dragunovartem99", Repo: "chessdocs", BaseBranch: "main"},
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	return client, &seen
}

func graphqlVariables(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	var payload struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode GraphQL request: %v", err)
	}
	return payload.Variables
}
