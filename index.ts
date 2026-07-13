import { buildApp } from "./app.ts";
import { ALLOWED_ORIGIN, PORT } from "./config/env.ts";
import { fetchSourceFile } from "./services/GithubService.ts";
import { openSubmissionPr } from "./services/SubmissionService.ts";

const app = await buildApp({
	allowedOrigin: ALLOWED_ORIGIN,
	services: { fetchSourceFile, openSubmissionPr },
});

async function shutdown(signal: string) {
	app.log.info(`Received ${signal}, shutting down`);
	await app.close();
	process.exit(0);
}

process.on("SIGTERM", () => void shutdown("SIGTERM"));
process.on("SIGINT", () => void shutdown("SIGINT"));

try {
	await app.listen({ port: PORT, host: "0.0.0.0" });
} catch (error) {
	app.log.error(error);
	process.exit(1);
}
