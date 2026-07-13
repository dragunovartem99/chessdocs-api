import assert from "node:assert/strict";
import { test } from "node:test";

import { normalizeSubmission } from "../utils/normalize.ts";

test("trims text fields and keeps the rest", () => {
	const normalized = normalizeSubmission({
		title: "  Title  ",
		content: "  Body\n",
		authorName: " Artem ",
		authorContact: " a@b.cd ",
		lang: "ru",
		sourcePath: "ru/glossary/fork.md",
	});

	assert.deepEqual(normalized, {
		title: "Title",
		content: "Body",
		authorName: "Artem",
		authorContact: "a@b.cd",
		lang: "ru",
		sourcePath: "ru/glossary/fork.md",
	});
});

test("leaves optional fields undefined", () => {
	const normalized = normalizeSubmission({
		title: "Title",
		content: "Body",
		lang: "en",
		sourcePath: "en/glossary/fork.md",
	});

	assert.equal(normalized.authorName, undefined);
	assert.equal(normalized.authorContact, undefined);
});
