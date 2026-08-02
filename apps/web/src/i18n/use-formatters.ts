import * as React from "react";
import { createFormatters } from "./formatters";
import { useLocale } from "./locale-provider";

export function useFormatters() {
	const { formattingLocale } = useLocale();

	return React.useMemo(
		() => createFormatters(formattingLocale),
		[formattingLocale],
	);
}
