package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

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
