import app from "./app.ts";
import { PORT } from "./config/server.ts";

try {
	await app.listen({ port: PORT, host: "0.0.0.0" });
} catch (error) {
	app.log.error(error);
	process.exit(1);
}
