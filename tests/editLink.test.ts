import assert from "node:assert/strict";
import { test } from "node:test";

import { isEditLinkDisabled } from "../utils/editLink.ts";

test("flags a page whose frontmatter sets editLink to false", () => {
	assert.equal(isEditLinkDisabled("---\nlayout: home\neditLink: false\n---\n# Home"), true);
});

test("allows a page with editLink set to a non-false value", () => {
	assert.equal(isEditLinkDisabled("---\neditLink: true\n---\n# Opening"), false);
});

test("allows a page with no frontmatter", () => {
	assert.equal(isEditLinkDisabled("# Blunder\n\nA bad move."), false);
});

test("allows a page whose frontmatter omits editLink", () => {
	assert.equal(isEditLinkDisabled("---\nlayout: home\n---\n# Home"), false);
});

test("ignores an editLink: false that appears outside the frontmatter block", () => {
	assert.equal(isEditLinkDisabled("# Title\n\nExample: `editLink: false`"), false);
});
