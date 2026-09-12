package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/api"
	"github.com/dragunovartem99/chessdocs-api/internal/docs"
	"github.com/dragunovartem99/chessdocs-api/internal/github"
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

func TestGetSourceReturnsMarkdown(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	resp := call(get(t, "en/glossary/blockade.md"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
	if got, want := body(t, resp), "# Source of en/glossary/blockade.md\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestNonEditablePagesAreRefused(t *testing.T) {
	frontmatters := map[string]string{
		"editLink: false": "---\neditLink: false\n---\n# Home\n",
		"dev: true":       "---\ndev: true\n---\n# PGN Editor\n",
	}

	for name, source := range frontmatters {
		t.Run(name, func(t *testing.T) {
			gh := &fakeGitHub{source: func(string) (string, error) { return source, nil }}
			call := serving(t, gh)

			for _, req := range []*http.Request{get(t, "en/about.md"), post(t, validBody)} {
				resp := call(req)
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("%s status = %d, want 403", req.Method, resp.StatusCode)
					continue
				}
				if got := errorMessage(t, resp); got != "This page is not editable" {
					t.Errorf("%s error = %q", req.Method, got)
				}
			}
			if len(gh.opened) != 0 {
				t.Errorf("opened %d pull requests, want 0", len(gh.opened))
			}
		})
	}
}

func TestPathsOutsideTheDocsTreeAreRejected(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	paths := []string{"", "../secrets.md", "en/../../etc/passwd.md", "en/notes.txt", "README.md"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			resp := call(get(t, path))
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestCreateSubmissionOpensPullRequest(t *testing.T) {
	gh := &fakeGitHub{}
	call := serving(t, gh)

	resp := call(post(t, `{
		"title": "  Fix the blockade article  ",
		"content": "  # Blockade\n\nUpdated text.\n  ",
		"sourcePath": "en/glossary/blockade.md",
		"author": { "name": " Artem ", "contact": " a@b.cd " }
	}`))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.StatusCode, body(t, resp))
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body(t, resp)), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.URL != "https://github.com/example/repo/pull/1" {
		t.Errorf("url = %q", payload.URL)
	}

	sub := gh.lastSubmission(t)
	if sub.Title != "Fix the blockade article" {
		t.Errorf("title = %q, want it trimmed", sub.Title)
	}
	if sub.Content != "# Blockade\n\nUpdated text." {
		t.Errorf("content = %q, want it trimmed", sub.Content)
	}
	if sub.Lang != docs.DefaultLang {
		t.Errorf("lang = %q, want the default %q", sub.Lang, docs.DefaultLang)
	}
	if sub.Author == nil || sub.Author.Name != "Artem" || sub.Author.Contact != "a@b.cd" {
		t.Errorf("author = %+v, want it trimmed", sub.Author)
	}
	// The page is read before it is written: editability lives in the source.
	if len(gh.fetched) != 1 || gh.fetched[0] != "en/glossary/blockade.md" {
		t.Errorf("fetched = %v, want the submitted path once", gh.fetched)
	}
}

func TestInvalidSubmissionsAreRejected(t *testing.T) {
	cases := map[string]struct {
		body   string
		status int
	}{
		"blank title":       {`{"title":"   ","content":"Body","sourcePath":"en/a.md"}`, http.StatusBadRequest},
		"blank content":     {`{"title":"Title","content":"\n\n","sourcePath":"en/a.md"}`, http.StatusBadRequest},
		"missing path":      {`{"title":"Title","content":"Body"}`, http.StatusBadRequest},
		"escaping path":     {`{"title":"Title","content":"Body","sourcePath":"../../etc/passwd.md"}`, http.StatusBadRequest},
		"bad contact":       {`{"title":"T","content":"B","sourcePath":"en/a.md","author":{"contact":"not-an-email"}}`, http.StatusBadRequest},
		"bad lang":          {`{"title":"T","content":"B","sourcePath":"en/a.md","lang":"english"}`, http.StatusBadRequest},
		"unknown field":     {`{"title":"T","content":"B","sourcePath":"en/a.md","admin":true}`, http.StatusBadRequest},
		"not an object":     {`["title"]`, http.StatusBadRequest},
		"empty body":        {``, http.StatusBadRequest},
		"oversized content": {`{"title":"T","content":"` + strings.Repeat("x", 17<<10) + `","sourcePath":"en/a.md"}`, http.StatusRequestEntityTooLarge},
	}

	gh := &fakeGitHub{}
	call := serving(t, gh)

	client := 0
	for name, tc := range cases {
		client++
		// Each case gets its own address: rejected requests still spend a
		// token, and there are more cases here than the write budget allows.
		ip := fmt.Sprintf("198.51.100.%d", client)

		t.Run(name, func(t *testing.T) {
			req := post(t, tc.body)
			req.Header.Set("X-Forwarded-For", ip)

			resp := call(req)
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
			}
			if msg := errorMessage(t, resp); msg == "" {
				t.Error("error message is empty")
			}
		})
	}
	if len(gh.opened) != 0 {
		t.Errorf("opened %d pull requests, want 0", len(gh.opened))
	}
}

func TestAnEmptyContactIsAllowed(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	resp := call(post(t, `{"title":"T","content":"B","sourcePath":"en/a.md","author":{"name":"","contact":""}}`))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", resp.StatusCode, body(t, resp))
	}
}

func TestSubmissionsAreRateLimitedPerClient(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	submit := func(ip string) *http.Response {
		req := post(t, validBody)
		req.Header.Set("X-Forwarded-For", ip)
		return call(req)
	}

	for i := range 5 {
		if resp := submit("203.0.113.7"); resp.StatusCode != http.StatusCreated {
			t.Fatalf("submission %d: status = %d, want 201", i+1, resp.StatusCode)
		}
	}

	blocked := submit("203.0.113.7")
	if blocked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", blocked.StatusCode)
	}
	if blocked.Header.Get("Retry-After") == "" {
		t.Error("a 429 should tell the client when to come back")
	}
	if msg := errorMessage(t, blocked); !strings.Contains(strings.ToLower(msg), "rate limit") {
		t.Errorf("error = %q", msg)
	}

	// A different caller is unaffected.
	if other := submit("198.51.100.9"); other.StatusCode != http.StatusCreated {
		t.Errorf("other client status = %d, want 201", other.StatusCode)
	}
}

func TestGitHubFailuresBecomeBadGateway(t *testing.T) {
	failure := &github.UpstreamError{Summary: "GitHub API error (500)", Detail: "token revoked"}
	gh := &fakeGitHub{
		source: func(string) (string, error) { return "", failure },
	}
	call := serving(t, gh)

	resp := call(get(t, "en/glossary/blockade.md"))
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	// The upstream detail stays in the log.
	if msg := errorMessage(t, resp); msg != failure.Summary {
		t.Errorf("error = %q, want %q", msg, failure.Summary)
	}
}

func TestPanicsAreContained(t *testing.T) {
	gh := &fakeGitHub{source: func(string) (string, error) { panic("boom") }}
	call := serving(t, gh)

	resp := call(get(t, "en/glossary/blockade.md"))

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if msg := errorMessage(t, resp); msg != "Internal server error" {
		t.Errorf("error = %q, want a generic message", msg)
	}
}

func TestCORS(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	t.Run("preflight from the site", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://chessdocs.org")
		resp := call(req)

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://chessdocs.org" {
			t.Errorf("Allow-Origin = %q", got)
		}
	})

	t.Run("another origin is not allowed", func(t *testing.T) {
		req := get(t, "en/glossary/blockade.md")
		req.Header.Set("Origin", "https://evil.example")
		resp := call(req)

		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, want it withheld", got)
		}
	})
}

func TestUnroutedRequestsAnswerInTheErrorShape(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	t.Run("unknown path", func(t *testing.T) {
		resp := call(httptest.NewRequest(http.MethodGet, "/elsewhere", nil))
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
		errorMessage(t, resp)
	})

	t.Run("unsupported method", func(t *testing.T) {
		resp := call(httptest.NewRequest(http.MethodDelete, "/", nil))
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", resp.StatusCode)
		}
		if resp.Header.Get("Allow") == "" {
			t.Error("a 405 should advertise the allowed methods")
		}
		errorMessage(t, resp)
	})
}
