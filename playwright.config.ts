import { defineConfig, devices } from "@playwright/test";

const port = 4173;
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
	testDir: "tests",
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? "github" : "list",
	use: {
		baseURL,
		trace: "on-first-retry",
	},
	webServer: {
		command: "bash scripts/playwright-stack.sh",
		url: baseURL,
		reuseExistingServer: !process.env.CI,
		timeout: 120_000,
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
