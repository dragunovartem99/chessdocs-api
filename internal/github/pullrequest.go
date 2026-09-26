package github

import (
	"context"
	"embed"

	"github.com/dragunovartem99/chessdocs-api/internal/docs"
)

//go:embed queries/*.graphql
var queriesFS embed.FS

var repoHeadQuery = mustReadQuery("repo_head.graphql")

// openSubmissionMutation branches, commits and opens the pull request in one
// round trip. createCommitOnBranch is used rather than the git plumbing because
// GitHub signs what it writes; expectedHeadOid holds it to the commit the branch
// was just cut from, so a retry cannot stack a second edit onto the same branch.
var openSubmissionMutation = mustReadQuery("open_submission.graphql")

func mustReadQuery(name string) string {
	content, err := queriesFS.ReadFile("queries/" + name)
	if err != nil {
		panic(err)
	}
	return string(content)
}

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
