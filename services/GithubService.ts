import { GITHUB_BASE_BRANCH, GITHUB_OWNER, GITHUB_REPO, GITHUB_TOKEN } from "../config/env.ts";
import { UpstreamError } from "../errors/UpstreamError.ts";

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
		// The public message stays generic; details go to the log via cause.
		throw new UpstreamError(`GitHub API error (${response.status})`, {
			cause: new Error(body),
		});
	}
	return response.json();
}

export async function githubGraphql<T>(
	query: string,
	variables: Record<string, unknown>
): Promise<T> {
	const response = await fetch("https://api.github.com/graphql", {
		method: "POST",
		headers: githubHeaders(),
		body: JSON.stringify({ query, variables }),
	});
	if (!response.ok) {
		const body = await response.text();
		throw new UpstreamError(`GitHub GraphQL error (${response.status})`, {
			cause: new Error(body),
		});
	}

	const { data, errors } = (await response.json()) as {
		data: T | null;
		errors?: { message: string }[];
	};
	if (errors?.length) {
		throw new UpstreamError("GitHub GraphQL error", {
			cause: new Error(errors.map((e) => e.message).join("; ")),
		});
	}
	return data as T;
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
