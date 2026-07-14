import { GITHUB_BASE_BRANCH, GITHUB_OWNER, GITHUB_REPO } from "../config/env.ts";
import type { Submission } from "../models/Submission.ts";
import { githubGraphql, toBase64 } from "./GithubService.ts";

const REPO_HEAD_QUERY = `
	query($owner: String!, $repo: String!, $baseRef: String!) {
		repository(owner: $owner, name: $repo) {
			id
			ref(qualifiedName: $baseRef) {
				target { oid }
			}
		}
	}
`;

const OPEN_SUBMISSION_MUTATION = `
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
`;

function slugify(text: string): string {
	const base = text
		.toLowerCase()
		.normalize("NFKD")
		.replaceAll(/[̀-ͯ]/gu, "")
		.replaceAll(/[^a-z0-9]+/gu, "-")
		.replaceAll(/^-+|-+$/gu, "");

	return base || "submission";
}

function branchName(submission: Submission): string {
	const { sourcePath, title } = submission;
	const docSlug = slugify(sourcePath.replace(/\.md$/u, "").replaceAll("/", "-"));
	const shortId = Date.now().toString(36);
	return `edit/${docSlug}/${slugify(title)}-${shortId}`;
}

function prTitle(submission: Submission): string {
	return `Docs edit: ${submission.title}`;
}

function prBody(submission: Submission): string {
	const { sourcePath, author } = submission;
	const lines = [
		`Suggested edit to \`docs/${sourcePath}\``,
		"",
		`**Submitted by:** ${author?.name || "Anonymous"}`,
	];
	if (author?.contact) lines.push(`**Contact:** ${author.contact}`);
	return lines.join("\n");
}

export async function openSubmissionPr(submission: Submission): Promise<string> {
	const { sourcePath, content: submittedContent } = submission;
	const branch = branchName(submission);
	const baseRef = `refs/heads/${GITHUB_BASE_BRANCH}`;

	const head = await githubGraphql<{
		repository: { id: string; ref: { target: { oid: string } } };
	}>(REPO_HEAD_QUERY, { owner: GITHUB_OWNER, repo: GITHUB_REPO, baseRef });
	const { id: repoId, ref } = head.repository;
	const baseOid = ref.target.oid;

	const content = submittedContent.endsWith("\n") ? submittedContent : `${submittedContent}\n`;

	// The submitted content is the full replacement the user edited,
	// so it overwrites the existing source file.
	const result = await githubGraphql<{ pr: { pullRequest: { url: string } } }>(
		OPEN_SUBMISSION_MUTATION,
		{
			repoId,
			refName: `refs/heads/${branch}`,
			baseOid,
			nameWithOwner: `${GITHUB_OWNER}/${GITHUB_REPO}`,
			branchName: branch,
			path: `docs/${sourcePath}`,
			content: toBase64(content),
			commitMessage: `Edit suggestion: ${submission.title}`,
			baseRefName: GITHUB_BASE_BRANCH,
			prTitle: prTitle(submission),
			prBody: prBody(submission),
		}
	);

	return result.pr.pullRequest.url;
}
