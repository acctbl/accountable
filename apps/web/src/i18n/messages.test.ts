import { createIntl } from "react-intl";
import { describe, expect, it } from "vitest";
import {
	isLocale,
	PSEUDO_LOCALES,
	SELECTABLE_LOCALES,
	SOURCE_LOCALE,
} from "./config";
import arSource from "./messages/ar.json";
import enSource from "./messages/en.json";

type SourceCatalog = Record<string, { defaultMessage: string }>;

const PLURAL_FORMS =
	"{count, plural, zero {ZERO} one {ONE} two {TWO} few {FEW} many {MANY} other {OTHER}}";

function toMessages(catalog: SourceCatalog): Record<string, string> {
	return Object.fromEntries(
		Object.entries(catalog).map(([id, message]) => [
			id,
			message.defaultMessage,
		]),
	);
}

function intlFor(locale: string, messages: Record<string, string>) {
	return createIntl({ locale, messages, onError: () => {} });
}

describe("config", () => {
	it("does not offer the source locale as a user choice", () => {
		expect(SELECTABLE_LOCALES).not.toContain(SOURCE_LOCALE);
	});

	it("does not offer pseudo-locales as product choices", () => {
		for (const locale of PSEUDO_LOCALES) {
			expect(SELECTABLE_LOCALES).not.toContain(locale);
		}
	});

	it("accepts every selectable locale", () => {
		for (const locale of SELECTABLE_LOCALES) {
			expect(isLocale(locale)).toBe(true);
		}
	});

	it("rejects locales that are not supported", () => {
		expect(isLocale("zu-ZA")).toBe(false);
		expect(isLocale("")).toBe(false);
	});
});

describe("arabic catalog", () => {
	it("translates every message in the source catalog", () => {
		expect(Object.keys(arSource).sort()).toEqual(Object.keys(enSource).sort());
	});

	it("writes every message in arabic script", () => {
		for (const [id, message] of Object.entries(toMessages(arSource))) {
			expect(message, id).toMatch(/[؀-ۿ]/);
		}
	});

	it("leaves no message identical to its english source", () => {
		const english = toMessages(enSource);

		for (const [id, message] of Object.entries(toMessages(arSource))) {
			expect(message, id).not.toBe(english[id]);
		}
	});

	it("parses as valid ICU and renders", () => {
		const intl = intlFor("ar", toMessages(arSource));

		for (const id of Object.keys(arSource)) {
			expect(
				intl.formatMessage(
					{ id },
					{ key: (chunks: string[]) => chunks.join("") },
				),
			).toBeTruthy();
		}
	});

	it("renders arabic rather than the english source", () => {
		const intl = intlFor("ar", toMessages(arSource));

		expect(intl.formatMessage({ id: "home.description" })).toBe(
			"واجهة النقل المحلية",
		);
	});
});

describe("plural rules", () => {
	it("selects all six arabic categories", () => {
		const intl = intlFor("ar", { plural: PLURAL_FORMS });
		const render = (count: number) =>
			intl.formatMessage({ id: "plural" }, { count });

		expect(render(0)).toBe("ZERO");
		expect(render(1)).toBe("ONE");
		expect(render(2)).toBe("TWO");
		expect(render(3)).toBe("FEW");
		expect(render(11)).toBe("MANY");
		expect(render(100)).toBe("OTHER");
	});

	it("selects the two english categories", () => {
		const intl = intlFor("en", { plural: PLURAL_FORMS });
		const render = (count: number) =>
			intl.formatMessage({ id: "plural" }, { count });

		expect(render(0)).toBe("OTHER");
		expect(render(1)).toBe("ONE");
		expect(render(2)).toBe("OTHER");
	});
});
