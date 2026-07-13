import cors from "@fastify/cors";
import Fastify from "fastify";
import type { FastifyError } from "fastify";

import { ContributeController } from "./controllers/ContributeController.ts";
import type { ContributeServices } from "./controllers/ContributeController.ts";
import { HttpError } from "./errors/HttpError.ts";
import { contributeRoutes } from "./routes/contributeRoutes.ts";

type AppOptions = {
	allowedOrigin: string;
	services: ContributeServices;
	logger?: boolean;
};

export async function buildApp(options: AppOptions) {
	const app = Fastify({ logger: options.logger ?? true });

	await app.register(cors, {
		origin: options.allowedOrigin,
		methods: ["GET", "POST", "OPTIONS"],
	});

	// Must be set before routes are registered: each route binds
	// the error handler of its context at registration time.
	app.setErrorHandler((error: FastifyError, req, reply) => {
		if (error instanceof HttpError) {
			if (error.statusCode >= 500) req.log.error(error);
			return reply.status(error.statusCode).send({ error: error.message });
		}
		if (error.validation) {
			return reply.status(400).send({ error: error.message });
		}
		req.log.error(error);
		return reply.status(500).send({ error: "Internal server error" });
	});

	const controller = new ContributeController(options.services);
	await app.register(contributeRoutes(controller));

	return app;
}
