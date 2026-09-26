package github

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

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
