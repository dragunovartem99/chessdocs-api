// Package github opens the chessdocs repository to the API: it reads docs
// sources from the base branch and turns submissions into pull requests.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// docsRoot mirrors chessdocs' .vitepress config `srcDir`: submitted source
// paths are relative to it, so every repo path the API touches is prefixed
// with it.
const docsRoot = "content/docs"

const (
	defaultBaseURL = "https://api.github.com"
	userAgent      = "chessdocs-api"

	// detailLimit caps how much of an upstream error body reaches the log.
	detailLimit = 2048
)

// Config names the repository the API contributes to and the token it acts as.
type Config struct {
	Token      string
	Owner      string
	Repo       string
	BaseBranch string
}

// Client talks to one GitHub repository. It is safe for concurrent use.
type Client struct {
	cfg  Config
	base *url.URL
	http *http.Client
}

// Option customises a Client; the defaults are what production uses.
type Option func(*Client)

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(gh *Client) { gh.http = c }
}

// WithBaseURL points the client at another GitHub endpoint, which is how the
// tests aim it at a local server.
func WithBaseURL(raw string) Option {
	return func(gh *Client) {
		if parsed, err := url.Parse(raw); err == nil {
			gh.base = parsed
		}
	}
}

// New builds a client for cfg.
func New(cfg Config, opts ...Option) *Client {
	base, _ := url.Parse(defaultBaseURL)
	gh := &Client{
		cfg:  cfg,
		base: base,
		http: &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(gh)
	}
	return gh
}

// UpstreamError reports a GitHub call that did not succeed. Summary is written
// for the submitter and says nothing about our credentials or the repository;
// Detail carries the upstream response and belongs in the log only.
type UpstreamError struct {
	Summary string
	Detail  string
}

func (e *UpstreamError) Error() string {
	if e.Detail == "" {
		return e.Summary
	}
	return e.Summary + ": " + e.Detail
}

// HTTPStatus and PublicMessage let the HTTP layer answer without having to
// know that GitHub exists.
func (e *UpstreamError) HTTPStatus() int       { return http.StatusBadGateway }
func (e *UpstreamError) PublicMessage() string { return e.Summary }

func (c *Client) endpoint(query url.Values, segments ...string) string {
	u := *c.base
	u.Path = "/" + strings.Join(segments, "/")
	u.RawQuery = query.Encode()
	return u.String()
}

func (c *Client) send(ctx context.Context, method, endpoint string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("build %s request: %w", method, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &UpstreamError{Summary: "GitHub is unreachable", Detail: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &UpstreamError{
			Summary: fmt.Sprintf("GitHub API error (%d)", resp.StatusCode),
			Detail:  snippet(resp.Body),
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &UpstreamError{Summary: "GitHub sent an unreadable response", Detail: err.Error()}
	}
	return nil
}

func snippet(r io.Reader) string {
	body, err := io.ReadAll(io.LimitReader(r, detailLimit))
	if err != nil {
		return err.Error()
	}
	return strings.TrimSpace(string(body))
}
