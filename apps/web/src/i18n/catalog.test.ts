import { describe, expect, it, vi } from "vitest";
import {
	type CatalogLoaders,
	catalogPath,
	localeChainLeastToMostSpecific,
	mergeCatalogs,
} from "./catalog";

type TextCatalog = Record<string, string>;

describe("localeChainLeastToMostSpecific", () => {
	it("returns language then region", () => {
		expect(localeChainLeastToMostSpecific("en-GB")).toEqual(["en", "en-GB"]);
	});

	it("collapses a bare language to a single entry", () => {
		expect(localeChainLeastToMostSpecific("ar")).toEqual(["ar"]);
	});

	it("treats a pseudo-locale as a region of its language", () => {
		expect(localeChainLeastToMostSpecific("en-XA")).toEqual(["en", "en-XA"]);
	});
});

describe("mergeCatalogs", () => {
	const base: TextCatalog = { "a.one": "base one", "a.two": "base two" };

	it("returns the base catalog when no loader matches", async () => {
		expect(await mergeCatalogs("en", base, {})).toEqual(base);
	});

	it("overlays a sparse regional catalog on top of the base", async () => {
		const loaders: CatalogLoaders<string> = {
			[catalogPath("en-GB")]: () =>
				Promise.resolve({ "a.two": "regional two" }),
		};

		expect(await mergeCatalogs("en-GB", base, loaders)).toEqual({
			"a.one": "base one",
			"a.two": "regional two",
		});
	});

	it("loads language and region in parallel while preserving precedence", async () => {
		let resolveLanguage: (catalog: TextCatalog) => void = () => {};
		const language = new Promise<TextCatalog>((resolve) => {
			resolveLanguage = resolve;
		});
		const loadRegion = vi.fn(() => Promise.resolve({ "a.two": "region two" }));
		const loaders: CatalogLoaders<string> = {
			[catalogPath("en")]: () => language,
			[catalogPath("en-GB")]: loadRegion,
		};

		const merging = mergeCatalogs("en-GB", base, loaders);
		expect(loadRegion).toHaveBeenCalledOnce();
		resolveLanguage({ "a.two": "language two" });
		const merged = await merging;

		expect(merged["a.two"]).toBe("region two");
	});

	it("fully replaces the base for a translated language", async () => {
		const loaders: CatalogLoaders<string> = {
			[catalogPath("ar")]: () =>
				Promise.resolve({ "a.one": "واحد", "a.two": "اثنان" }),
		};

		expect(await mergeCatalogs("ar", base, loaders)).toEqual({
			"a.one": "واحد",
			"a.two": "اثنان",
		});
	});

	it("keeps base entries a sparse translation does not override", async () => {
		const loaders: CatalogLoaders<string> = {
			[catalogPath("ar")]: () => Promise.resolve({ "a.one": "واحد" }),
		};

		const merged = await mergeCatalogs("ar", base, loaders);

		expect(merged["a.two"]).toBe("base two");
	});

	it("does not mutate the base catalog", async () => {
		const loaders: CatalogLoaders<string> = {
			[catalogPath("ar")]: () => Promise.resolve({ "a.one": "واحد" }),
		};

		await mergeCatalogs("ar", base, loaders);

		expect(base["a.one"]).toBe("base one");
	});

	it("does not load catalogs outside the chain", async () => {
		const unrelated = vi.fn(() => Promise.resolve({ "a.one": "wrong" }));
		const loaders: CatalogLoaders<string> = { [catalogPath("ar")]: unrelated };

		await mergeCatalogs("en-GB", base, loaders);

		expect(unrelated).not.toHaveBeenCalled();
	});
});
