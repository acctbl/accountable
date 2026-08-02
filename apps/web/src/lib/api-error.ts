import { LocalizedErrorSchema } from "@accountable/proto/accountable/type/v1/localized_error_pb";
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
	messageKey: string | null;
};

export function parseApiError(error: unknown): ApiError {
	const connectError = ConnectError.from(error);
	const [localized] = connectError.findDetails(LocalizedErrorSchema);

	return {
		code: connectError.code,
		messageKey: localized?.messageKey ? localized.messageKey : null,
	};
}

export function formatApiError(intl: IntlShape, error: unknown): string {
	const { messageKey } = parseApiError(error);
	const descriptor =
		(messageKey ? apiErrorMessagesById[messageKey] : undefined) ??
		apiErrorMessages.unknown;

	return intl.formatMessage(descriptor);
}
