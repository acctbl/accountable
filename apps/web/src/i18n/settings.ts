export const SOURCE_LOCALE = "en";

export const SELECTABLE_LOCALES = ["en-US", "en-GB", "ar"] as const;

export const PSEUDO_LOCALES = ["en-XA", "en-XB"] as const;

export type Locale =
	| typeof SOURCE_LOCALE
	| (typeof SELECTABLE_LOCALES)[number]
	| (typeof PSEUDO_LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en-US";
