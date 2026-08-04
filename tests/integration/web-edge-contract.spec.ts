import { expect, test } from "@playwright/test";

// The preview stack simulates the CloudFront contract defined in
// infra/opentofu/web.tf; these assertions pin that contract in CI.
const cacheControl = {
	runtimeConfig: "no-store",
	hashedAsset: "public, max-age=31536000, immutable",
	document: "no-cache",
} as const;

test("serves the runtime config artifact uncached with the strict schema", async ({
	request,
}) => {
	const response = await request.get("/_runtime/config.json");
	expect(response.status()).toBe(200);
	expect(response.headers()["cache-control"]).toBe(cacheControl.runtimeConfig);
	expect(response.headers()["content-type"]).toContain("application/json");

	const body = await response.json();
	expect(Object.keys(body).sort()).toEqual([
		"api_base_url",
		"architecture_probe",
		"configuration_revision",
		"schema_version",
	]);
	expect(body.schema_version).toBe(1);
});

test("serves documents and deep links as revalidated SPA fallbacks", async ({
	request,
}) => {
	const root = await request.get("/");
	expect(root.status()).toBe(200);
	expect(root.headers()["cache-control"]).toBe(cacheControl.document);

	const deepLink = await request.get("/a/deep/route");
	expect(deepLink.status()).toBe(200);
	expect(deepLink.headers()["content-type"]).toContain("text/html");
	expect(deepLink.headers()["cache-control"]).toBe(cacheControl.document);
});

test("serves hashed assets as immutable", async ({ request }) => {
	const root = await request.get("/");
	const assetPath = (await root.text()).match(/\/assets\/[^"]+\.js/)?.[0];
	if (!assetPath) {
		throw new Error("index.html references no hashed asset");
	}

	const asset = await request.get(assetPath);
	expect(asset.status()).toBe(200);
	expect(asset.headers()["cache-control"]).toBe(cacheControl.hashedAsset);
});
