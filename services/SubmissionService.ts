import { GITHUB_BASE_BRANCH, GITHUB_OWNER, GITHUB_REPO } from "../config/env.ts";
import type { Submission } from "../utils/validate.ts";
import { fetchSourceFile, githubRequest, toBase64 } from "./GithubService.ts";

function slugify(title: string): string {
	const base = title
		.toLowerCase()
		.normalize("NFKD")
		.replaceAll(/[̀-ͯ]/gu, "")
		.replaceAll(/[^a-z0-9]+/gu, "-")
		.replaceAll(/^-+|-+$/gu, "");

	return base || "submission";
}

async function createBranch(branch: string): Promise<void> {
	const baseRef = (await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/git/ref/heads/${GITHUB_BASE_BRANCH}`,
		{ method: "GET" }
	)) as { object: { sha: string } };

	await githubRequest(`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/git/refs`, {
		method: "POST",
		body: JSON.stringify({ ref: `refs/heads/${branch}`, sha: baseRef.object.sha }),
	});
}

// The description is the full replacement content the user edited,
// so it overwrites the existing source file.
async function commitEdit(submission: Submission, branch: string): Promise<void> {
	const { title, description, sourcePath } = submission;
	const { sha } = await fetchSourceFile(sourcePath);
	const content = description.endsWith("\n") ? description : `${description}\n`;

	await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/docs/${sourcePath}`,
		{
			method: "PUT",
			body: JSON.stringify({
				message: `Edit suggestion: ${title}`,
				content: toBase64(content),
				branch,
				sha,
			}),
		}
	);
}

export async function openSubmissionPr(submission: Submission): Promise<string> {
	const { title, authorName, authorContact, sourcePath } = submission;
	const branch = `submission/${Date.now()}-${slugify(title)}`;

	await createBranch(branch);
	await commitEdit(submission, branch);

	const pr = (await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/pulls`,
		{
			method: "POST",
			body: JSON.stringify({
				title: `Edit suggestion for ${sourcePath}: ${title}`,
				head: branch,
				base: GITHUB_BASE_BRANCH,
				body: `Edit suggestion for \`docs/${sourcePath}\`.${authorName ? ` From ${authorName}.` : ""}${authorContact ? `\n\nContact: ${authorContact}` : ""}`,
			}),
		}
	)) as { html_url: string };

	return pr.html_url;
}
