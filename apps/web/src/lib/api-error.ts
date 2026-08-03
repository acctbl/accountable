import { ProblemDetailSchema } from "@accountable/proto/accountable/error/v1/problem_pb";
import { type Code, ConnectError } from "@connectrpc/connect";
import {
	defineMessages,
	type IntlShape,
	type MessageDescriptor,
} from "react-intl";

export const apiErrorMessages = defineMessages({
	unknown: {
		id: "errors.unknown",
		defaultMessage: "Something went wrong. Please try again.",
	},
	invalidInput: {
		id: "errors.invalidInput",
		defaultMessage: "The probe input was not valid.",
	},
	unavailable: {
		id: "errors.unavailable",
		defaultMessage: "The API is temporarily unavailable.",
	},
	deadlineExceeded: {
		id: "errors.deadlineExceeded",
		defaultMessage: "The request took too long to complete.",
	},
	internal: {
		id: "errors.internal",
		defaultMessage: "The API could not complete the request.",
	},
});

const apiErrorMessagesById: Record<string, MessageDescriptor> =
	Object.fromEntries(
		Object.values(apiErrorMessages).map((descriptor) => [
			descriptor.id,
			descriptor,
		]),
	);

export type ApiError = {
	code: Code;
	category: string | null;
	messageKey: string | null;
	problemId: string | null;
	requestId: string | null;
	fieldViolations: ReadonlyArray<{ fieldPath: string; code: string }>;
};

export function parseApiError(error: unknown): ApiError {
	const connectError = ConnectError.from(error);
	const [problem] = connectError.findDetails(ProblemDetailSchema);
	return {
		code: connectError.code,
		category: problem?.code || null,
		messageKey: problem?.messageKey || null,
		problemId: problem?.problemId || null,
		requestId: problem?.requestId || null,
		fieldViolations: problem?.fieldViolations ?? [],
	};
}

export function formatApiError(intl: IntlShape, error: unknown): string {
	const { messageKey } = parseApiError(error);
	const descriptor =
		(messageKey ? apiErrorMessagesById[messageKey] : undefined) ??
		apiErrorMessages.unknown;
	return intl.formatMessage(descriptor);
}
