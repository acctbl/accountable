import { describe, expect, it } from "vitest";
import { createFormatters } from "./formatters";

const UTC = "UTC";
const LAGOS = "Africa/Lagos";
const NEW_YEARS_EVE_UTC = Date.UTC(2026, 0, 1, 2, 30);

describe("digits", () => {
	it("uses latin digits for arabic instead of arabic-indic", () => {
		const arabic = createFormatters("ar");

		expect(arabic.number(1234.5)).toMatch(/1.?234/);
		expect(arabic.number(1234.5)).not.toMatch(/[٠-٩]/);
	});

	it("uses latin digits in arabic dates", () => {
		const arabic = createFormatters("ar");

		expect(arabic.dateInZone(NEW_YEARS_EVE_UTC, UTC)).not.toMatch(/[٠-٩]/);
	});
});

describe("money", () => {
	it("treats the amount as minor units", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.money(123456, "USD")).toBe("$1,234.56");
	});

	it("uses the currency's own fraction digits", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.money(123456, "JPY")).toBe("¥123,456");
	});

	it("renders negatives with a minus sign by default", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.money(-123456, "USD")).toBe("-$1,234.56");
	});

	it("renders negatives in parentheses when accounting is requested", () => {
		const formatters = createFormatters("en-US");

		expect(
			formatters.money(-123456, "USD", { useAccountingNegatives: true }),
		).toBe("($1,234.56)");
	});

	it("drops fraction digits on request", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.money(123456, "USD", { hideFractionDigits: true })).toBe(
			"$1,235",
		);
	});
});

describe("dates", () => {
	it("puts the same instant on different calendar days per zone", () => {
		const formatters = createFormatters("en-GB");

		expect(formatters.dateInZone(NEW_YEARS_EVE_UTC, UTC)).toBe("1 Jan 2026");
		expect(formatters.dateInZone(NEW_YEARS_EVE_UTC, "Pacific/Honolulu")).toBe(
			"31 Dec 2025",
		);
	});

	it("respects the requested zone rather than the host zone", () => {
		const formatters = createFormatters("en-GB");

		expect(formatters.dateTimeInZone(NEW_YEARS_EVE_UTC, LAGOS)).toContain(
			"03:30",
		);
	});

	it("orders dates the british way for en-GB", () => {
		const formatters = createFormatters("en-GB");

		expect(formatters.dateInZone(NEW_YEARS_EVE_UTC, UTC, "short")).toBe(
			"01/01/2026",
		);
	});

	it("orders dates the american way for en-US", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.dateInZone(NEW_YEARS_EVE_UTC, UTC, "short")).toBe(
			"1/1/26",
		);
	});
});

describe("collation", () => {
	it("sorts accented names next to their base letter", () => {
		const formatters = createFormatters("en-US");

		expect(formatters.sort(["Zoe", "Ålund", "Adams"], (name) => name)).toEqual([
			"Adams",
			"Ålund",
			"Zoe",
		]);
	});

	it("sorts numbers within strings numerically", () => {
		const formatters = createFormatters("en-US");

		expect(
			formatters.sort(["Invoice 10", "Invoice 2"], (name) => name),
		).toEqual(["Invoice 2", "Invoice 10"]);
	});

	it("differs from code point ordering", () => {
		const formatters = createFormatters("en-US");
		const names = ["Zoe", "Ålund"];

		expect(formatters.sort(names, (name) => name)).not.toEqual(
			[...names].sort(),
		);
	});
});

describe("lists", () => {
	it("joins with the locale conjunction", () => {
		expect(createFormatters("en-GB").list(["a", "b", "c"])).toBe("a, b and c");
	});

	it("joins with the locale disjunction", () => {
		expect(createFormatters("en-US").list(["a", "b"], "disjunction")).toBe(
			"a or b",
		);
	});
});
