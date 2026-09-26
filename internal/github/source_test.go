package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

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
