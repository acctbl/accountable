import { ProblemDetailSchema } from "@accountable/proto/accountable/error/v1/problem_pb";
import { Code, ConnectError } from "@connectrpc/connect";
import { createIntl } from "react-intl";
import { describe, expect, it } from "vitest";
import { apiErrorMessages, formatApiError, parseApiError } from "./api-error";

const intl = createIntl({
	locale: "en",
	messages: Object.fromEntries(
		Object.values(apiErrorMessages).map((message) => [
			message.id,
			message.defaultMessage,
		]),
	),
	onError: () => {},
});

describe("parseApiError", () => {
	it("reads the safe shared problem detail", () => {
		const error = new ConnectError(
			"request failed",
			Code.Unavailable,
			undefined,
			[
				{
					desc: ProblemDetailSchema,
					value: {
						code: "unavailable",
						messageKey: "errors.unavailable",
						problemId: "problem-1",
						requestId: "request-1",
						fieldViolations: [
							{ fieldPath: "command.amount", code: "must_be_positive" },
						],
					},
				},
			],
		);
		expect(parseApiError(error)).toMatchObject({
			code: Code.Unavailable,
			category: "unavailable",
			messageKey: "errors.unavailable",
			problemId: "problem-1",
			requestId: "request-1",
			fieldViolations: [
				{ fieldPath: "command.amount", code: "must_be_positive" },
			],
		});
		expect(formatApiError(intl, error)).toBe(
			"The API is temporarily unavailable.",
		);
	});

	it("falls back without exposing raw transport text", () => {
		const error = new Error("sql: provider secret path");
		expect(formatApiError(intl, error)).toBe(
			"Something went wrong. Please try again.",
		);
	});
});
