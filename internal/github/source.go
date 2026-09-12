package github

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
)

// FetchSource returns the markdown of one docs page as it stands on the base
// branch. sourcePath is docs-relative and must already be validated.
func (c *Client) FetchSource(ctx context.Context, sourcePath string) (string, error) {
	endpoint := c.endpoint(
		url.Values{"ref": {c.cfg.BaseBranch}},
		"repos", c.cfg.Owner, c.cfg.Repo, "contents", docsRoot, sourcePath,
	)

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := c.send(ctx, "GET", endpoint, nil, &file); err != nil {
		return "", err
	}
	// Files above a megabyte come back with their content elided; no docs page
	// is anywhere near that, so treat it as an upstream fault rather than as
	// an empty page the submitter would then overwrite.
	if file.Encoding != "base64" {
		return "", &UpstreamError{
			Summary: "GitHub sent an unreadable response",
			Detail:  "unexpected content encoding " + file.Encoding,
		}
	}

	// GitHub wraps its base64 at 60 columns.
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return "", &UpstreamError{Summary: "GitHub sent an unreadable response", Detail: err.Error()}
	}
	return string(decoded), nil
}
