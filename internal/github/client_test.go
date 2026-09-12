package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
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

func TestFetchSource(t *testing.T) {
	const source = "---\nlayout: home\n---\n# Home\n"

	client, requests := stub(t, func(w http.ResponseWriter, r *http.Request) {
		// GitHub wraps base64 at 60 columns; the client has to cope.
		encoded := base64.StdEncoding.EncodeToString([]byte(source))
		json.NewEncoder(w).Encode(map[string]string{
			"content":  encoded[:8] + "\n" + encoded[8:],
			"encoding": "base64",
		})
	})

	got, err := client.FetchSource(context.Background(), "en/glossary/fork.md")
	if err != nil {
		t.Fatalf("FetchSource() = %v", err)
	}
	if got != source {
		t.Errorf("FetchSource() = %q, want %q", got, source)
	}

	req := (*requests)[0]
	wantPath := "/repos/dragunovartem99/chessdocs/contents/content/docs/en/glossary/fork.md"
	if req.URL.Path != wantPath {
		t.Errorf("path = %q, want %q", req.URL.Path, wantPath)
	}
	if ref := req.URL.Query().Get("ref"); ref != "main" {
		t.Errorf("ref = %q, want the base branch", ref)
	}
	if auth := req.Header.Get("Authorization"); auth != "Bearer test-token" {
		t.Errorf("Authorization = %q", auth)
	}
}

func TestFetchSourceReportsUpstreamFailures(t *testing.T) {
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"message":"Not Found"}`)
	})

	_, err := client.FetchSource(context.Background(), "en/missing.md")

	var upstream *UpstreamError
	if !errors.As(err, &upstream) {
		t.Fatalf("FetchSource() = %v, want an UpstreamError", err)
	}
	if upstream.HTTPStatus() != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", upstream.HTTPStatus())
	}
	if upstream.PublicMessage() != "GitHub API error (404)" {
		t.Errorf("public message = %q", upstream.PublicMessage())
	}
	// The upstream body is kept for the log but not for the submitter.
	if !strings.Contains(upstream.Error(), "Not Found") {
		t.Errorf("logged error = %q, want the upstream body", upstream.Error())
	}
	if strings.Contains(upstream.PublicMessage(), "Not Found") {
		t.Error("the upstream body leaked into the public message")
	}
}

func TestFetchSourceRejectsElidedContent(t *testing.T) {
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"content": "", "encoding": "none"})
	})

	if _, err := client.FetchSource(context.Background(), "en/huge.md"); err == nil {
		t.Fatal("FetchSource() = nil, want an error rather than an empty page")
	}
}

func TestOpenPullRequest(t *testing.T) {
	var mutation map[string]any

	// The head query comes first, then the mutation that does the work.
	calls := 0
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		if calls++; calls == 1 {
			io.WriteString(w, `{"data":{"repository":{"id":"R_1","ref":{"target":{"oid":"deadbeef"}}}}}`)
			return
		}
		mutation = graphqlVariables(t, r)
		io.WriteString(w, `{"data":{"pr":{"pullRequest":{"url":"https://github.com/o/r/pull/7"}}}}`)
	})

	url, err := client.OpenPullRequest(context.Background(), docs.Submission{
		Title:      "Clarify the example",
		Content:    "# Fork\n\nA double attack.",
		Lang:       "en",
		SourcePath: "en/glossary/fork.md",
		Author:     &docs.Author{Name: "Artem", Contact: "a@b.cd"},
	})
	if err != nil {
		t.Fatalf("OpenPullRequest() = %v", err)
	}
	if url != "https://github.com/o/r/pull/7" {
		t.Errorf("url = %q", url)
	}

	if got := mutation["path"]; got != "content/docs/en/glossary/fork.md" {
		t.Errorf("committed path = %v, want it under the docs root", got)
	}
	if got := mutation["baseOid"]; got != "deadbeef" {
		t.Errorf("baseOid = %v, want the head the branch was cut from", got)
	}
	if got := mutation["prTitle"]; got != "Docs edit: Clarify the example" {
		t.Errorf("pull request title = %v", got)
	}

	branch, _ := mutation["branchName"].(string)
	if !strings.HasPrefix(branch, "edit/en-glossary-fork/clarify-the-example-") {
		t.Errorf("branch = %q, want it named after the page and the title", branch)
	}
	if mutation["refName"] != "refs/heads/"+branch {
		t.Errorf("refName = %v, want it to match the branch", mutation["refName"])
	}

	content, _ := mutation["content"].(string)
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		t.Fatalf("commit content is not base64: %v", err)
	}
	if want := "# Fork\n\nA double attack.\n"; string(decoded) != want {
		t.Errorf("commit content = %q, want %q with the trailing newline restored", decoded, want)
	}

	body, _ := mutation["prBody"].(string)
	for _, want := range []string{"content/docs/en/glossary/fork.md", "**Submitted by:** Artem", "**Contact:** a@b.cd"} {
		if !strings.Contains(body, want) {
			t.Errorf("pull request body is missing %q:\n%s", want, body)
		}
	}
}

func TestOpenPullRequestCreditsAnonymousSubmitters(t *testing.T) {
	var mutation map[string]any

	// The head query comes first, then the mutation that does the work.
	calls := 0
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		if calls++; calls == 1 {
			io.WriteString(w, `{"data":{"repository":{"id":"R_1","ref":{"target":{"oid":"deadbeef"}}}}}`)
			return
		}
		mutation = graphqlVariables(t, r)
		io.WriteString(w, `{"data":{"pr":{"pullRequest":{"url":"https://github.com/o/r/pull/8"}}}}`)
	})

	_, err := client.OpenPullRequest(context.Background(), docs.Submission{
		Title:      "Опечатка",
		Content:    "# Вилка",
		SourcePath: "ru/glossary/fork.md",
	})
	if err != nil {
		t.Fatalf("OpenPullRequest() = %v", err)
	}

	body, _ := mutation["prBody"].(string)
	if !strings.Contains(body, "**Submitted by:** Anonymous") {
		t.Errorf("body = %q, want an anonymous attribution", body)
	}
	if strings.Contains(body, "**Contact:**") {
		t.Error("body advertises a contact that was never given")
	}
	// A title with nothing ASCII left still has to yield a usable ref.
	if branch, _ := mutation["branchName"].(string); !strings.HasPrefix(branch, "edit/ru-glossary-fork/submission-") {
		t.Errorf("branch = %q, want the fallback slug", branch)
	}
}

func TestGraphQLErrorsAreUpstreamFailures(t *testing.T) {
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":null,"errors":[{"message":"Resource not accessible"}]}`)
	})

	_, err := client.OpenPullRequest(context.Background(), docs.Submission{
		Title: "Title", Content: "Body", SourcePath: "en/a.md",
	})

	var upstream *UpstreamError
	if !errors.As(err, &upstream) {
		t.Fatalf("OpenPullRequest() = %v, want an UpstreamError", err)
	}
	if upstream.PublicMessage() != "GitHub GraphQL error" {
		t.Errorf("public message = %q", upstream.PublicMessage())
	}
	if !strings.Contains(upstream.Error(), "Resource not accessible") {
		t.Errorf("logged error = %q, want the GraphQL message", upstream.Error())
	}
}

func TestOpenPullRequestFailsWhenTheBaseBranchIsMissing(t *testing.T) {
	client, _ := stub(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"repository":{"id":"R_1","ref":null}}}`)
	})

	_, err := client.OpenPullRequest(context.Background(), docs.Submission{
		Title: "Title", Content: "Body", SourcePath: "en/a.md",
	})
	if err == nil {
		t.Fatal("OpenPullRequest() = nil, want an error rather than a panic")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Clarify the example":   "clarify-the-example",
		"  Spaces  everywhere ": "spaces-everywhere",
		"Rook + Pawn = win!":    "rook-pawn-win",
		"Опечатка":              "submission",
		"":                      "submission",
		"---":                   "submission",
	}

	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}
