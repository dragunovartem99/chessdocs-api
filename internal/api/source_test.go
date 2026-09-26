package api_test

import (
	"net/http"
	"strings"
	"testing"
)

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
