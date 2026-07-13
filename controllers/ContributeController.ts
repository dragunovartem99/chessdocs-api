import type { FastifyReply, FastifyRequest } from "fastify";

import { fetchSourceFile } from "../services/GithubService.ts";
import { openSubmissionPr } from "../services/SubmissionService.ts";
import { SOURCE_PATH_PATTERN, validate } from "../utils/validate.ts";
import type { SubmissionBody } from "../utils/validate.ts";

export class ContributeController {
	async getSource(req: FastifyRequest, reply: FastifyReply) {
		const { path } = req.query as { path?: string };
		if (!path || !SOURCE_PATH_PATTERN.test(path)) return reply.status(400).send("Invalid path");

		try {
			const { content } = await fetchSourceFile(path);
			return reply.type("text/plain").send(content);
		} catch (error) {
			req.log.error(error);
			return reply.status(502).send("Failed to fetch source");
		}
	}

	async createSubmission(req: FastifyRequest, reply: FastifyReply) {
		const submission = validate(req.body as SubmissionBody);
		if (!submission) return reply.status(400).send("Invalid submission");

		try {
			const url = await openSubmissionPr(submission);
			return reply.status(201).send({ url });
		} catch (error) {
			req.log.error(error);
			return reply.status(502).send("Failed to create submission");
		}
	}
}
