import { buildApp } from "../app.ts";
import type { ContributeServices } from "../controllers/ContributeController.ts";
import { UpstreamError } from "../errors/UpstreamError.ts";
import type { Submission } from "../models/Submission.ts";

export type Captured = { submissions: Submission[]; fetchedPaths: string[] };

export async function createTestApp(overrides: Partial<ContributeServices> = {}) {
	const captured: Captured = { submissions: [], fetchedPaths: [] };

	const services: ContributeServices = {
		async fetchSourceFile(sourcePath) {
			captured.fetchedPaths.push(sourcePath);
			return { content: `# Source of ${sourcePath}\n`, sha: "abc123" };
		},
		async openSubmissionPr(submission) {
			captured.submissions.push(submission);
			return "https://github.com/example/repo/pull/1";
		},
		...overrides,
	};

	const app = await buildApp({ allowedOrigin: "*", services, logger: false });
	return { app, captured };
}

export function failingServices(): ContributeServices {
	return {
		async fetchSourceFile() {
			throw new UpstreamError("GitHub API error (500)");
		},
		async openSubmissionPr() {
			throw new UpstreamError("GitHub GraphQL error");
		},
	};
}
