import type { FastifyReply, FastifyRequest } from "fastify";

import { HttpError } from "../errors/HttpError.ts";
import type { Submission } from "../models/Submission.ts";
import type { fetchSourceFile } from "../services/GithubService.ts";
import type { openSubmissionPr } from "../services/SubmissionService.ts";
import { isNonEditable } from "../utils/editLink.ts";
import { normalizeSubmission } from "../utils/normalize.ts";

export type ContributeServices = {
	fetchSourceFile: typeof fetchSourceFile;
	openSubmissionPr: typeof openSubmissionPr;
};

export class ContributeController {
	#services: ContributeServices;

	constructor(services: ContributeServices) {
		this.#services = services;
	}

	async getSource(req: FastifyRequest<{ Querystring: { path: string } }>, reply: FastifyReply) {
		const { content } = await this.#services.fetchSourceFile(req.query.path);
		if (isNonEditable(content)) {
			throw new HttpError(403, "This page is not editable");
		}
		return reply.type("text/plain").send(content);
	}

	async createSubmission(req: FastifyRequest<{ Body: Submission }>, reply: FastifyReply) {
		const submission = normalizeSubmission(req.body);
		const { content } = await this.#services.fetchSourceFile(submission.sourcePath);
		if (isNonEditable(content)) {
			throw new HttpError(403, "This page is not editable");
		}

		const url = await this.#services.openSubmissionPr(submission);
		return reply.status(201).send({ url });
	}
}
