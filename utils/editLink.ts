// Mirrors chessdocs' modules/edit/utils/paths.ts: a page is non-editable when
// its frontmatter sets `editLink: false` or `dev: true`. Kept identical so the two never drift.
export function isEditLinkDisabled(content: string): boolean {
	const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/u.exec(content);
	return frontmatter ? /^editLink:\s*false\s*$/mu.test(frontmatter[1]!) : false;
}

export function isDevOnly(content: string): boolean {
	const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/u.exec(content);
	return frontmatter ? /^dev:\s*true\s*$/mu.test(frontmatter[1]!) : false;
}

export function isNonEditable(content: string): boolean {
	return isEditLinkDisabled(content) || isDevOnly(content);
}
