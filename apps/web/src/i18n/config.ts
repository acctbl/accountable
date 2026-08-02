export {
	FORMATTING_LOCALE_COOKIE,
	getDirection,
	LOCALE_COOKIE,
	languageOf,
} from "./bootstrap";

import {
	DEFAULT_LOCALE,
	type Locale,
	PSEUDO_LOCALES,
	SELECTABLE_LOCALES,
	SOURCE_LOCALE,
} from "./settings";

export {
	DEFAULT_LOCALE,
	type Locale,
	PSEUDO_LOCALES,
	SELECTABLE_LOCALES,
	SOURCE_LOCALE,
};

export const PSEUDO_LOCALES_ENABLED = import.meta.env.DEV;

export const SUPPORTED_LOCALES: readonly Locale[] = PSEUDO_LOCALES_ENABLED
	? [...SELECTABLE_LOCALES, ...PSEUDO_LOCALES]
	: SELECTABLE_LOCALES;

export function isLocale(value: string): value is Locale {
	return SUPPORTED_LOCALES.some((supported) => supported === value);
}
