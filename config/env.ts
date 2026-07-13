function requireString(key: string): string {
	const value = process.env[key];
	if (!value) throw new Error(`Missing required environment variable: ${key}`);
	return value;
}

function requireNumber(key: string): number {
	const value = Number(requireString(key));
	if (!Number.isFinite(value)) throw new Error(`Environment variable ${key} must be a number`);
	return value;
}

export const GITHUB_TOKEN = requireString("GITHUB_TOKEN");
export const GITHUB_OWNER = requireString("GITHUB_OWNER");
export const GITHUB_REPO = requireString("GITHUB_REPO");
export const GITHUB_BASE_BRANCH = requireString("GITHUB_BASE_BRANCH");
export const ALLOWED_ORIGIN = requireString("ALLOWED_ORIGIN");
export const PORT = requireNumber("PORT");
