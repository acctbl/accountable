import { beforeEach, describe, expect, it } from "vitest";
import { getDirection, languageOf, localeBootstrapScript } from "./bootstrap";

function setCookie(value: string) {
	Object.defineProperty(document, "cookie", {
		value,
		configurable: true,
		writable: true,
	});
}

function setBrowserLanguages(languages: readonly string[]) {
	Object.defineProperty(window.navigator, "languages", {
		value: languages,
		configurable: true,
	});
}

function runBootstrap({
	cookie = "",
	search = "",
	languages = ["en-US"],
	includePseudoLocales = false,
}: {
	cookie?: string;
	search?: string;
	languages?: readonly string[];
	includePseudoLocales?: boolean;
} = {}) {
	document.documentElement.lang = "";
	document.documentElement.dir = "";
	window.history.replaceState({}, "", `/${search}`);
	setCookie(cookie);
	setBrowserLanguages(languages);

	new Function(
		"window",
		"document",
		localeBootstrapScript(includePseudoLocales),
	)(window, document);

	return {
		lang: document.documentElement.lang,
		dir: document.documentElement.dir,
	};
}

beforeEach(() => {
	window.history.replaceState({}, "", "/");
});

describe("locale metadata", () => {
	it("extracts and normalizes the primary language subtag", () => {
		expect(languageOf("en-GB")).toBe("en");
		expect(languageOf("AR-EG")).toBe("ar");
	});

	it("assigns direction consistently", () => {
		for (const locale of ["ar", "ar-EG", "he", "fa-IR", "ur", "en-XB"]) {
			expect(getDirection(locale), locale).toBe("rtl");
		}

		for (const locale of ["en", "en-GB", "en-XA"]) {
			expect(getDirection(locale), locale).toBe("ltr");
		}
	});
});

describe("localeBootstrapScript", () => {
	it("uses the URL before the cookie and browser preferences", () => {
		expect(
			runBootstrap({
				search: "?lang=ar",
				cookie: "acctbl-locale=en-GB",
				languages: ["en-US"],
			}),
		).toEqual({ lang: "ar", dir: "rtl" });
	});

	it("uses the cookie before browser preferences", () => {
		expect(
			runBootstrap({
				cookie: "acctbl-theme=dark; acctbl-locale=ar",
				languages: ["en-US"],
			}),
		).toEqual({ lang: "ar", dir: "rtl" });
	});

	it("uses browser preferences on a first visit", () => {
		expect(runBootstrap({ languages: ["ar-EG", "en-US"] })).toEqual({
			lang: "ar",
			dir: "rtl",
		});
	});

	it("ignores malformed URL and cookie values", () => {
		expect(
			runBootstrap({
				search: "?lang=en_US",
				cookie: "acctbl-locale=%E0%A4%A",
				languages: ["ar"],
			}),
		).toEqual({ lang: "ar", dir: "rtl" });
	});

	it("falls back to the default for unsupported preferences", () => {
		expect(runBootstrap({ languages: ["ja-JP"] })).toEqual({
			lang: "en-US",
			dir: "ltr",
		});
	});

	it("supports both pseudo-locales when explicitly enabled", () => {
		expect(
			runBootstrap({
				search: "?lang=en-XA",
				includePseudoLocales: true,
			}),
		).toEqual({ lang: "en-XA", dir: "ltr" });
		expect(
			runBootstrap({
				search: "?lang=en-XB",
				includePseudoLocales: true,
			}),
		).toEqual({ lang: "en-XB", dir: "rtl" });
	});
});
