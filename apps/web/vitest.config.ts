import { defineConfig } from "vitest/config";

export default defineConfig({
	test: {
		projects: [
			{
				test: {
					name: "unit",
					include: ["src/**/*.test.ts"],
					environment: "happy-dom",
				},
			},
			{
				test: {
					name: "component",
					include: ["src/**/*.test.tsx"],
					environment: "happy-dom",
				},
			},
		],
	},
});
