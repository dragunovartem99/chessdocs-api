import type { FastifyReply, FastifyRequest } from "fastify";

import type { Submission } from "../models/Submission.ts";
import type { fetchSourceFile } from "../services/GithubService.ts";
import type { openSubmissionPr } from "../services/SubmissionService.ts";
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
		return reply.type("text/plain").send(content);
	}

	async createSubmission(req: FastifyRequest<{ Body: Submission }>, reply: FastifyReply) {
		const url = await this.#services.openSubmissionPr(normalizeSubmission(req.body));
		return reply.status(201).send({ url });
	}
}
