package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

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
