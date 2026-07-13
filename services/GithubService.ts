import { GITHUB_BASE_BRANCH, GITHUB_OWNER, GITHUB_REPO, GITHUB_TOKEN } from "../config/env.ts";

export function toBase64(input: string): string {
	const bytes = new TextEncoder().encode(input);
	let binary = "";
	for (const byte of bytes) binary += String.fromCodePoint(byte);
	return btoa(binary);
}

function fromBase64(input: string): string {
	const binary = atob(input.replaceAll("\n", ""));
	const bytes = Uint8Array.from(binary, (char) => char.codePointAt(0)!);
	return new TextDecoder().decode(bytes);
}

function githubHeaders(): HeadersInit {
	return {
		"Authorization": `Bearer ${GITHUB_TOKEN}`,
		"Accept": "application/vnd.github+json",
		"Content-Type": "application/json",
		"User-Agent": "chessdocs-api",
	};
}

export async function githubRequest(url: string, init: RequestInit) {
	const response = await fetch(url, { ...init, headers: githubHeaders() });
	if (!response.ok) {
		const body = await response.text();
		throw new Error(`GitHub API error (${response.status}): ${body}`);
	}
	return response.json();
}

export async function fetchSourceFile(
	sourcePath: string
): Promise<{ content: string; sha: string }> {
	const file = (await githubRequest(
		`https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/docs/${sourcePath}?ref=${GITHUB_BASE_BRANCH}`,
		{ method: "GET" }
	)) as { content: string; sha: string };

	return { content: fromBase64(file.content), sha: file.sha };
}
