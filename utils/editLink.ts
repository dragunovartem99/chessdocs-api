// Mirrors chessdocs' modules/edit/utils/paths.ts: a page is non-editable when
// its frontmatter sets `editLink: false`. Kept identical so the two never drift.
export function isEditLinkDisabled(content: string): boolean {
	const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/u.exec(content);
	return frontmatter ? /^editLink:\s*false\s*$/mu.test(frontmatter[1]!) : false;
}
