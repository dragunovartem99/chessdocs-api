const MAX_TITLE_LENGTH = 120;
const MAX_DESCRIPTION_LENGTH = 4000;
const MAX_CONTACT_LENGTH = 120;
// docs-relative path, e.g. "en/glossary/blockade.md" — validated to prevent path traversal
// or pointing outside the docs tree; never used to choose which file the server writes to.
export const SOURCE_PATH_PATTERN = /^[a-z]{2}\/[\w/-]+\.md$/u;

export type SubmissionBody = {
	title?: unknown;
	description?: unknown;
	authorName?: unknown;
	authorContact?: unknown;
	lang?: unknown;
	sourcePath?: unknown;
};

export type Submission = {
	title: string;
	description: string;
	authorName: string;
	authorContact: string;
	lang: string;
	sourcePath?: string;
};

export function validate(body: SubmissionBody): Submission | null {
	const { title, description, authorName, authorContact, lang, sourcePath } = body;

	if (typeof title !== "string" || !title.trim() || title.length > MAX_TITLE_LENGTH) return null;
	if (
		typeof description !== "string" ||
		!description.trim() ||
		description.length > MAX_DESCRIPTION_LENGTH
	)
		return null;
	if (
		authorName !== undefined &&
		(typeof authorName !== "string" || authorName.length > MAX_CONTACT_LENGTH)
	)
		return null;
	if (
		authorContact !== undefined &&
		authorContact !== "" &&
		(typeof authorContact !== "string" ||
			authorContact.length > MAX_CONTACT_LENGTH ||
			!/^[^\s@]+@[^\s@]+\.[^\s@]+$/u.test(authorContact))
	)
		return null;
	if (
		sourcePath !== undefined &&
		(typeof sourcePath !== "string" || !SOURCE_PATH_PATTERN.test(sourcePath))
	)
		return null;

	return {
		title: title.trim(),
		description: description.trim(),
		authorName: typeof authorName === "string" ? authorName.trim() : "",
		authorContact: typeof authorContact === "string" ? authorContact.trim() : "",
		lang: typeof lang === "string" && /^[a-z]{2}$/u.test(lang) ? lang : "en",
		sourcePath: typeof sourcePath === "string" ? sourcePath : undefined,
	};
}
