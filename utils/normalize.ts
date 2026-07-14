import type { Submission } from "../models/Submission.ts";

export function normalizeSubmission(submission: Submission): Submission {
	return {
		...submission,
		title: submission.title.trim(),
		content: submission.content.trim(),
		author: submission.author && {
			name: submission.author.name?.trim(),
			contact: submission.author.contact?.trim(),
		},
	};
}
