const REQUIRED_KEYS = [
	"GITHUB_TOKEN",
	"GITHUB_OWNER",
	"GITHUB_REPO",
	"GITHUB_BASE_BRANCH",
	"ALLOWED_ORIGIN",
] as const;

for (const key of REQUIRED_KEYS) {
	if (!process.env[key]) throw new Error(`Missing required environment variable: ${key}`);
}

export const GITHUB_TOKEN = process.env.GITHUB_TOKEN!;
export const GITHUB_OWNER = process.env.GITHUB_OWNER!;
export const GITHUB_REPO = process.env.GITHUB_REPO!;
export const GITHUB_BASE_BRANCH = process.env.GITHUB_BASE_BRANCH!;
export const ALLOWED_ORIGIN = process.env.ALLOWED_ORIGIN!;
