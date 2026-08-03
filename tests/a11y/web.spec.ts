import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

test("existing shell reflows and has no axe violations", async ({ page }) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.setViewportSize({ width: 320, height: 720 });
	await page.goto("/");
	await expect(page.getByRole("main")).toHaveCount(1);

	const overflow = await page.evaluate(
		() => document.documentElement.scrollWidth - window.innerWidth,
	);
	expect(overflow).toBeLessThanOrEqual(0);
	const results = await new AxeBuilder({ page })
		.withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
		.analyze();
	expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual(
		[],
	);
});
