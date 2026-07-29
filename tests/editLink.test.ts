import assert from "node:assert/strict";
import { test } from "node:test";

import { isDevOnly, isEditLinkDisabled, isNonEditable } from "../utils/editLink.ts";

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

test("flags a page whose frontmatter sets dev to true", () => {
	assert.equal(isDevOnly("---\ndev: true\n---\n# PGN Editor"), true);
});

test("allows a page with no dev flag", () => {
	assert.equal(isDevOnly("---\nlayout: home\n---\n# Home"), false);
});

test("isNonEditable flags a page marked dev: true even without editLink: false", () => {
	assert.equal(isNonEditable("---\ndev: true\n---\n# PGN Editor"), true);
});

test("isNonEditable flags a page marked editLink: false even without dev: true", () => {
	assert.equal(isNonEditable("---\neditLink: false\n---\n# Home"), true);
});

test("isNonEditable allows a regular page", () => {
	assert.equal(isNonEditable("---\nlayout: home\n---\n# Home"), false);
});
