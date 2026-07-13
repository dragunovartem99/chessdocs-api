import type { FastifyInstance } from "fastify";

import { ContributeController } from "../controllers/ContributeController.ts";

const contributeController = new ContributeController();

export default function contributeRoutes(app: FastifyInstance) {
	app.get("/", contributeController.getSource.bind(contributeController));
	app.post("/", contributeController.createSubmission.bind(contributeController));
}
