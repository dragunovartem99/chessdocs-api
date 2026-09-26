package github

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

func pullRequestBody(sub docs.Submission) string {
	author := "Anonymous"
	contact := ""
	if sub.Author != nil {
		if sub.Author.Name != "" {
			author = sub.Author.Name
		}
		contact = sub.Author.Contact
	}

	lines := []string{
		"Suggested edit to `" + docsRoot + "/" + sub.SourcePath + "`",
		"",
		"**Submitted by:** " + author,
	}
	if contact != "" {
		lines = append(lines, "**Contact:** "+contact)
	}
	return strings.Join(lines, "\n")
}

// branchName derives a readable, collision-resistant branch from the page and
// the submission title, e.g. edit/en-glossary-fork/clarify-the-example-m9k2p1.
func branchName(sub docs.Submission) string {
	page := strings.TrimSuffix(sub.SourcePath, ".md")
	page = strings.ReplaceAll(page, "/", "-")
	suffix := strconv.FormatInt(time.Now().UnixMilli(), 36)
	return "edit/" + slugify(page) + "/" + slugify(sub.Title) + "-" + suffix
}

// slugify reduces text to the lowercase ASCII alphanumerics git refs and human
// readers both tolerate. A title written entirely in another script leaves
// nothing behind, so it falls back to a fixed word.
func slugify(text string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		default:
			dash = true
		}
	}
	if b.Len() == 0 {
		return "submission"
	}
	return b.String()
}

// encodeContent prepares page content for the commit API, restoring the
// trailing newline every text file in the repository is expected to end with.
func encodeContent(content string) string {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return base64.StdEncoding.EncodeToString([]byte(content))
}
