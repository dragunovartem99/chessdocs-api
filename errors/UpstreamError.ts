import { HttpError } from "./HttpError.ts";

export class UpstreamError extends HttpError {
	constructor(message: string, options?: ErrorOptions) {
		super(502, message, options);
	}
}
