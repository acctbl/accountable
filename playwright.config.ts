import { defineConfig, devices } from "@playwright/test";

const port = 4173;
const baseURL = `https://127.0.0.1:${port}`;

export default defineConfig({
	testDir: "tests",
	fullyParallel: true,
	workers: 3,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? "github" : "list",
	expect: { timeout: 10_000 },
	use: {
		baseURL,
		ignoreHTTPSErrors: true,
		trace: "on-first-retry",
	},
	webServer: {
		command: "bash scripts/with-test-postgres.sh bash scripts/playwright-stack.sh",
		url: baseURL,
		ignoreHTTPSErrors: true,
		gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
		reuseExistingServer: false,
		timeout: 120_000,
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
		{
			name: "firefox",
			use: { ...devices["Desktop Firefox"] },
		},
		{
			name: "webkit",
			use: { ...devices["Desktop Safari"] },
		},
	],
});
