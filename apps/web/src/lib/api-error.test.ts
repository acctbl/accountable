import { LocalizedErrorSchema } from "@accountable/proto/accountable/type/v1/localized_error_pb";
import { Code, ConnectError } from "@connectrpc/connect";
import { createIntl } from "react-intl";
import { describe, expect, it } from "vitest";
import { apiErrorMessages, formatApiError, parseApiError } from "./api-error";

const intl = createIntl({
	locale: "en",
	messages: {
		[apiErrorMessages.unknown.id]: apiErrorMessages.unknown.defaultMessage,
	},
	onError: () => {},
});

describe("parseApiError", () => {
	it("reads code and message key from a localized Connect error", () => {
		const error = new ConnectError("log-only", Code.Unavailable, undefined, [
			{
				desc: LocalizedErrorSchema,
				value: { messageKey: "errors.unknown" },
			},
		]);

		expect(parseApiError(error)).toEqual({
			code: Code.Unavailable,
			messageKey: "errors.unknown",
		});
	});

	it("returns a null message key when no localized detail is present", () => {
		expect(parseApiError(new ConnectError("boom", Code.Internal))).toEqual({
			code: Code.Internal,
			messageKey: null,
		});
	});
});

describe("formatApiError", () => {
	it("renders the localized message for a known key", () => {
		const error = new ConnectError("log-only", Code.Unavailable, undefined, [
			{
				desc: LocalizedErrorSchema,
				value: { messageKey: "errors.unknown" },
			},
		]);

		expect(formatApiError(intl, error)).toBe(
			"Something went wrong. Please try again.",
		);
	});

	it("falls back for unknown keys and raw transport errors", () => {
		const unknownKey = new ConnectError("log-only", Code.Internal, undefined, [
			{
				desc: LocalizedErrorSchema,
				value: { messageKey: "errors.not.in.catalog" },
			},
		]);

		expect(formatApiError(intl, unknownKey)).toBe(
			"Something went wrong. Please try again.",
		);
		expect(formatApiError(intl, new Error("[internal] boom"))).toBe(
			"Something went wrong. Please try again.",
		);
	});
});
