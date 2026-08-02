import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test("home page has no accessibility violations", async ({ page }) => {
	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Project ready!" }),
	).toBeVisible();
	await expect(page.getByText("Connected (dev)")).toBeVisible();

	const results = await new AxeBuilder({ page }).analyze();
	expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual(
		[],
	);
});
