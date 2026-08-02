import { expect, test } from "@playwright/test";

test("home page connects to the API", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByText("Connected (dev)")).toBeVisible();
});
