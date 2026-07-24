import type { FastifyInstance } from "fastify";

import { ContributeController } from "../controllers/ContributeController.ts";
import type { Submission } from "../models/Submission.ts";
import { bodySchema, querySchema } from "../schemas/openapi.ts";

export function contributeRoutes(controller: ContributeController) {
	return function routes(app: FastifyInstance) {
		app.get<{ Querystring: { path: string } }>(
			"/",
			{ schema: { querystring: querySchema("/", "get") } },
			controller.getSource.bind(controller)
		);
		app.post<{ Body: Submission }>(
			"/",
			{
				schema: { body: bodySchema("/", "post") },
				// Opening PRs spends the server GITHUB_TOKEN, so cap it far below the global default.
				config: { rateLimit: { max: 5, timeWindow: "10 minutes" } },
			},
			controller.createSubmission.bind(controller)
		);
	};
}
