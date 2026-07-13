import cors from "@fastify/cors";
import Fastify from "fastify";

import { ALLOWED_ORIGIN } from "./config/env.ts";
import contributeRoutes from "./routes/contributeRoutes.ts";

const app = Fastify({ logger: true });

await app.register(cors, { origin: ALLOWED_ORIGIN, methods: ["GET", "POST", "OPTIONS"] });
await app.register(contributeRoutes);

export default app;
