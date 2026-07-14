import assert from "node:assert/strict";
import { after, test } from "node:test";

import { createTestApp, failingServices } from "./helpers.ts";

const { app, captured } = await createTestApp();
const { app: failingApp } = await createTestApp(failingServices());
after(async () => {
	await app.close();
	await failingApp.close();
});

const validSubmission = {
	title: "Fix the blockade article",
	content: "# Blockade\n\nUpdated text.",
	sourcePath: "en/glossary/blockade.md",
};

test("returns the source file as plain text", async () => {
	const response = await app.inject({
		method: "GET",
		url: "/?path=en/glossary/blockade.md",
	});

	assert.equal(response.statusCode, 200);
	assert.match(response.headers["content-type"] as string, /text\/plain/u);
	assert.equal(response.body, "# Source of en/glossary/blockade.md\n");
});

test("rejects a path outside the docs tree", async () => {
	const paths = ["../secrets.md", "en/../../etc/passwd.md", "en/notes.txt", ""];
	const responses = await Promise.all(
		paths.map((path) => app.inject({ method: "GET", url: `/?path=${encodeURIComponent(path)}` })),
	);
	responses.forEach((response, i) => {
		assert.equal(response.statusCode, 400, `expected 400 for ${JSON.stringify(paths[i])}`);
	});
});

test("opens a pull request for a valid submission", async () => {
	const response = await app.inject({
		method: "POST",
		url: "/",
		payload: { ...validSubmission, title: "  Fix the blockade article  " },
	});

	assert.equal(response.statusCode, 201);
	assert.deepEqual(response.json(), { url: "https://github.com/example/repo/pull/1" });

	const submission = captured.submissions.at(-1)!;
	assert.equal(submission.title, "Fix the blockade article");
	assert.equal(submission.lang, "en");
});

test("rejects a whitespace-only title", async () => {
	const response = await app.inject({
		method: "POST",
		url: "/",
		payload: { ...validSubmission, title: "   " },
	});

	assert.equal(response.statusCode, 400);
});

test("rejects an invalid contact email", async () => {
	const response = await app.inject({
		method: "POST",
		url: "/",
		payload: { ...validSubmission, author: { contact: "not-an-email" } },
	});

	assert.equal(response.statusCode, 400);
});

test("rejects a submission without sourcePath", async () => {
	const { sourcePath: _sourcePath, ...withoutPath } = validSubmission;
	const response = await app.inject({ method: "POST", url: "/", payload: withoutPath });

	assert.equal(response.statusCode, 400);
});

test("maps GitHub failures to 502", async () => {
	const getResponse = await failingApp.inject({
		method: "GET",
		url: "/?path=en/glossary/blockade.md",
	});
	assert.equal(getResponse.statusCode, 502);
	assert.deepEqual(getResponse.json(), { error: "GitHub API error (500)" });

	const postResponse = await failingApp.inject({
		method: "POST",
		url: "/",
		payload: validSubmission,
	});
	assert.equal(postResponse.statusCode, 502);
});
