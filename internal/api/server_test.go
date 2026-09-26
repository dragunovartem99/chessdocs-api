package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/github"
)

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
