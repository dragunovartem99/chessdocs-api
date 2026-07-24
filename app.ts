import cors from "@fastify/cors";
import rateLimit from "@fastify/rate-limit";
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
	const app = Fastify({
		logger: options.logger ?? true,
		// Behind Caddy — derive req.ip from X-Forwarded-For so rate limits key on the real client.
		trustProxy: true,
		// content is maxLength 4000; reject oversized payloads before validation.
		bodyLimit: 16 * 1024,
	});

	await app.register(cors, {
		origin: options.allowedOrigin,
		methods: ["GET", "POST", "OPTIONS"],
	});

	// In-memory store is fine for the single-container deploy; switch to a
	// shared store only if the API is ever scaled horizontally. The 429 is
	// shaped into the API's { error } schema by the error handler below.
	await app.register(rateLimit, { max: 60, timeWindow: "1 minute" });

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
		// Client errors raised by plugins (e.g. @fastify/rate-limit's 429) carry
		// their own status and a safe message; pass them through unchanged.
		if (
			typeof error.statusCode === "number" &&
			error.statusCode >= 400 &&
			error.statusCode < 500
		) {
			return reply.status(error.statusCode).send({ error: error.message });
		}
		req.log.error(error);
		return reply.status(500).send({ error: "Internal server error" });
	});

	const controller = new ContributeController(options.services);
	await app.register(contributeRoutes(controller));

	return app;
}
