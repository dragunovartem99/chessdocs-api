package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/api"
	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

const validBody = `{
	"title": "Fix the blockade article",
	"content": "# Blockade\n\nUpdated text.",
	"sourcePath": "en/glossary/blockade.md"
}`

// fakeGitHub stands in for the repository. Its zero value serves a plain page
// for any path and opens pull requests successfully.
type fakeGitHub struct {
	mu      sync.Mutex
	source  func(path string) (string, error)
	pull    func(sub docs.Submission) (string, error)
	fetched []string
	opened  []docs.Submission
}

func (f *fakeGitHub) FetchSource(_ context.Context, path string) (string, error) {
	f.mu.Lock()
	f.fetched = append(f.fetched, path)
	f.mu.Unlock()

	if f.source != nil {
		return f.source(path)
	}
	return "# Source of " + path + "\n", nil
}

func (f *fakeGitHub) OpenPullRequest(_ context.Context, sub docs.Submission) (string, error) {
	f.mu.Lock()
	f.opened = append(f.opened, sub)
	f.mu.Unlock()

	if f.pull != nil {
		return f.pull(sub)
	}
	return "https://github.com/example/repo/pull/1", nil
}

func (f *fakeGitHub) lastSubmission(t *testing.T) docs.Submission {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.opened) == 0 {
		t.Fatal("no pull request was opened")
	}
	return f.opened[len(f.opened)-1]
}

// serving starts the API in front of gh and returns a client-side call helper.
func serving(t *testing.T, gh *fakeGitHub) func(*http.Request) *http.Response {
	t.Helper()

	handler := api.NewServer(api.Options{
		GitHub:        gh,
		AllowedOrigin: "https://chessdocs.org",
		Logger:        slog.New(slog.DiscardHandler),
	})

	return func(req *http.Request) *http.Response {
		t.Helper()
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder.Result()
	}
}

func get(t *testing.T, path string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodGet, "/?path="+path, nil)
}

func post(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	read, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(read)
}

func errorMessage(t *testing.T, resp *http.Response) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	raw := body(t, resp)
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("response is not an error object: %s", raw)
	}
	return payload.Error
}
