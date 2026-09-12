package github

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

const repoHeadQuery = `
	query($owner: String!, $repo: String!, $baseRef: String!) {
		repository(owner: $owner, name: $repo) {
			id
			ref(qualifiedName: $baseRef) {
				target { oid }
			}
		}
	}
`

// openSubmissionMutation branches, commits and opens the pull request in one
// round trip. createCommitOnBranch is used rather than the git plumbing because
// GitHub signs what it writes; expectedHeadOid holds it to the commit the branch
// was just cut from, so a retry cannot stack a second edit onto the same branch.
const openSubmissionMutation = `
	mutation(
		$repoId: ID!
		$refName: String!
		$baseOid: GitObjectID!
		$nameWithOwner: String!
		$branchName: String!
		$path: String!
		$content: Base64String!
		$commitMessage: String!
		$baseRefName: String!
		$prTitle: String!
		$prBody: String!
	) {
		createRef(input: { repositoryId: $repoId, name: $refName, oid: $baseOid }) {
			ref { name }
		}
		commit: createCommitOnBranch(
			input: {
				branch: { repositoryNameWithOwner: $nameWithOwner, branchName: $branchName }
				message: { headline: $commitMessage }
				fileChanges: { additions: [{ path: $path, contents: $content }] }
				expectedHeadOid: $baseOid
			}
		) {
			commit { oid }
		}
		pr: createPullRequest(
			input: {
				repositoryId: $repoId
				baseRefName: $baseRefName
				headRefName: $branchName
				title: $prTitle
				body: $prBody
			}
		) {
			pullRequest { url }
		}
	}
`

// OpenPullRequest publishes a submission as a branch, a commit replacing the
// page, and a pull request against the base branch. It returns the pull
// request's URL.
func (c *Client) OpenPullRequest(ctx context.Context, sub docs.Submission) (string, error) {
	var head struct {
		Repository struct {
			ID  string `json:"id"`
			Ref *struct {
				Target struct {
					OID string `json:"oid"`
				} `json:"target"`
			} `json:"ref"`
		} `json:"repository"`
	}
	err := c.graphql(ctx, repoHeadQuery, map[string]any{
		"owner":   c.cfg.Owner,
		"repo":    c.cfg.Repo,
		"baseRef": "refs/heads/" + c.cfg.BaseBranch,
	}, &head)
	if err != nil {
		return "", err
	}
	if head.Repository.Ref == nil {
		return "", &UpstreamError{
			Summary: "GitHub GraphQL error",
			Detail:  "base branch " + c.cfg.BaseBranch + " not found",
		}
	}

	branch := branchName(sub)
	var result struct {
		PR struct {
			PullRequest struct {
				URL string `json:"url"`
			} `json:"pullRequest"`
		} `json:"pr"`
	}
	err = c.graphql(ctx, openSubmissionMutation, map[string]any{
		"repoId":        head.Repository.ID,
		"refName":       "refs/heads/" + branch,
		"baseOid":       head.Repository.Ref.Target.OID,
		"nameWithOwner": c.cfg.Owner + "/" + c.cfg.Repo,
		"branchName":    branch,
		"path":          docsRoot + "/" + sub.SourcePath,
		// The submission is the full page the reader edited, so it replaces
		// the source file outright.
		"content":       encodeContent(sub.Content),
		"commitMessage": "Edit suggestion: " + sub.Title,
		"baseRefName":   c.cfg.BaseBranch,
		"prTitle":       "Docs edit: " + sub.Title,
		"prBody":        pullRequestBody(sub),
	}, &result)
	if err != nil {
		return "", err
	}
	return result.PR.PullRequest.URL, nil
}

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
