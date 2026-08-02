import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FORMATTING_LOCALE_COOKIE, LOCALE_COOKIE } from "./bootstrap";
import { DEFAULT_LOCALE } from "./config";
import {
	negotiate,
	readFormattingLocalePreference,
	updateLocaleInUrl,
	writeLocaleCookie,
} from "./negotiate";

function setCookie(value: string) {
	Object.defineProperty(document, "cookie", {
		value,
		configurable: true,
		writable: true,
	});
}

function setBrowserLanguages(languages: readonly string[]) {
	Object.defineProperty(navigator, "languages", {
		value: languages,
		configurable: true,
	});
}

beforeEach(() => {
	setCookie("");
	setBrowserLanguages(["en-US"]);
	window.history.replaceState({}, "", "/");
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe("negotiate", () => {
	it("prefers the lang query parameter over everything else", () => {
		setCookie(`${LOCALE_COOKIE}=en-GB`);
		setBrowserLanguages(["en-US"]);

		expect(negotiate("?lang=ar")).toBe("ar");
	});

	it("falls back to the cookie when no query parameter is present", () => {
		setCookie(`${LOCALE_COOKIE}=en-GB`);

		expect(negotiate("")).toBe("en-GB");
	});

	it("falls back to browser languages when no cookie is stored", () => {
		setBrowserLanguages(["ar-EG", "en-US"]);

		expect(negotiate("")).toBe("ar");
	});

	it("falls back to the default locale when nothing matches", () => {
		setBrowserLanguages(["ja-JP"]);

		expect(negotiate("")).toBe(DEFAULT_LOCALE);
	});

	it("ignores an unsupported query parameter and continues negotiating", () => {
		setCookie(`${LOCALE_COOKIE}=ar`);

		expect(negotiate("?lang=ja-JP")).toBe("ar");
	});

	it("ignores a malformed query parameter and continues negotiating", () => {
		setCookie(`${LOCALE_COOKIE}=ar`);

		expect(negotiate("?lang=en_US")).toBe("ar");
	});

	it("matches an unlisted english region to the closest supported one", () => {
		setBrowserLanguages(["en-AU"]);

		expect(negotiate("")).toBe("en-GB");
	});

	it("matches an arabic region to the base arabic locale", () => {
		setBrowserLanguages(["ar-EG"]);

		expect(negotiate("")).toBe("ar");
	});

	it("never negotiates to the source locale", () => {
		for (const requested of ["en", "en-AU", "ja-JP", "ar-EG"]) {
			expect(negotiate(`?lang=${requested}`)).not.toBe("en");
		}
	});

	it("ignores malformed encoded cookies without throwing", () => {
		setCookie(`${LOCALE_COOKIE}=%E0%A4%A`);
		setBrowserLanguages(["ar"]);

		expect(negotiate("")).toBe("ar");
	});

	it("reads the locale cookie alongside unrelated cookies", () => {
		setCookie(`acctbl-theme=dark; ${LOCALE_COOKIE}=ar; other=1`);

		expect(negotiate("")).toBe("ar");
	});

	it("uses the default locale when the browser reports no languages", () => {
		setBrowserLanguages([]);

		expect(negotiate("")).toBe(DEFAULT_LOCALE);
	});
});

describe("writeLocaleCookie", () => {
	it("writes a year-long site-wide cookie", () => {
		const written: string[] = [];
		Object.defineProperty(document, "cookie", {
			configurable: true,
			get: () => written.join("; "),
			set: (value: string) => {
				written.push(value);
			},
		});

		writeLocaleCookie("ar");

		expect(written[0]).toContain(`${LOCALE_COOKIE}=ar`);
		expect(written[0]).toContain("path=/");
		expect(written[0]).toContain("max-age=31536000");
		expect(written[0]).toContain("SameSite=Lax");
	});
});

describe("readFormattingLocalePreference", () => {
	it("ignores an unsupported formatting locale preference", () => {
		setCookie(`${FORMATTING_LOCALE_COOKIE}=ja-JP`);

		expect(readFormattingLocalePreference()).toBeNull();
	});
});

describe("updateLocaleInUrl", () => {
	it("updates an existing locale override and preserves other URL state", () => {
		window.history.replaceState(
			{ retained: true },
			"",
			"/reports?tab=overdue&lang=ar#summary",
		);

		updateLocaleInUrl("en-GB");

		expect(window.location.pathname).toBe("/reports");
		expect(window.location.search).toBe("?tab=overdue&lang=en-GB");
		expect(window.location.hash).toBe("#summary");
		expect(window.history.state).toEqual({ retained: true });
	});

	it("leaves URLs without a locale override unchanged", () => {
		window.history.replaceState({}, "", "/reports?tab=overdue");

		updateLocaleInUrl("ar");

		expect(window.location.href).toBe(
			"http://localhost:3000/reports?tab=overdue",
		);
	});
});
