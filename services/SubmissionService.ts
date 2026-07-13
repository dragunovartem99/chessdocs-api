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

// Edit suggestion: the description is the full replacement content the user edited,
// so it overwrites the existing source file rather than landing in a new file.
async function commitEdit(submission: Submission, branch: string): Promise<void> {
	const { title, description, sourcePath } = submission;
	const { sha } = await fetchSourceFile(sourcePath!);

	await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/docs/${sourcePath}`,
		{
			method: "PUT",
			body: JSON.stringify({
				message: `Edit suggestion: ${title}`,
				content: toBase64(description),
				branch,
				sha,
			}),
		}
	);
}

// New submission: there's no existing file to edit, so it's recorded as a new
// front-matter file for maintainers to triage and turn into docs.
async function commitNewSubmission(submission: Submission, branch: string): Promise<void> {
	const { title, description, authorName, authorContact, lang } = submission;
	const path = `submissions/${branch.replace("submission/", "")}.md`;

	const fileContent = [
		"---",
		`title: ${JSON.stringify(title)}`,
		`lang: ${lang}`,
		`authorName: ${JSON.stringify(authorName)}`,
		`authorContact: ${JSON.stringify(authorContact)}`,
		`submittedAt: ${new Date().toISOString()}`,
		"---",
		"",
		description,
		"",
	].join("\n");

	await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/${path}`,
		{
			method: "PUT",
			body: JSON.stringify({
				message: `Submission: ${title}`,
				content: toBase64(fileContent),
				branch,
			}),
		}
	);
}

export async function openSubmissionPr(submission: Submission): Promise<string> {
	const { title, authorName, authorContact, sourcePath } = submission;
	const branch = `submission/${Date.now()}-${slugify(title)}`;
	const prTitle = sourcePath
		? `Edit suggestion for ${sourcePath}: ${title}`
		: `Submission: ${title}`;

	await createBranch(branch);

	if (sourcePath) await commitEdit(submission, branch);
	else await commitNewSubmission(submission, branch);

	const pr = (await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/pulls`,
		{
			method: "POST",
			body: JSON.stringify({
				title: prTitle,
				head: branch,
				base: GITHUB_BASE_BRANCH,
				body: `${sourcePath ? `Edit suggestion for \`docs/${sourcePath}\`.` : "New content submission."}${authorName ? ` From ${authorName}.` : ""}${authorContact ? `\n\nContact: ${authorContact}` : ""}`,
			}),
		}
	)) as { html_url: string };

	return pr.html_url;
}
