import type { Submission } from "../models/Submission.ts";

export function normalizeSubmission(submission: Submission): Submission {
	return {
		...submission,
		title: submission.title.trim(),
		content: submission.content.trim(),
		authorName: submission.authorName?.trim(),
		authorContact: submission.authorContact?.trim(),
	};
}
