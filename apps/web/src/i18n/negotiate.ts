import { match } from "@formatjs/intl-localematcher";
import { FORMATTING_LOCALE_COOKIE, LOCALE_COOKIE } from "./bootstrap";
import {
	DEFAULT_LOCALE,
	isLocale,
	type Locale,
	SUPPORTED_LOCALES,
} from "./config";

const LOCALE_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 365;

const SUPPORTED_LANGUAGES = new Set(
	SUPPORTED_LOCALES.map((locale) => locale.split("-")[0]?.toLowerCase()),
);

function canonicalize(locale: string): string | null {
	try {
		return Intl.getCanonicalLocales(locale)[0] ?? null;
	} catch {
		return null;
	}
}

function readCookie(name: string): string | null {
	const prefix = `${name}=`;

	for (const entry of document.cookie.split("; ")) {
		if (entry.startsWith(prefix)) {
			try {
				return decodeURIComponent(entry.slice(prefix.length));
			} catch {
				return null;
			}
		}
	}

	return null;
}

function bestMatch(requested: readonly string[]): Locale | null {
	const valid = requested
		.map(canonicalize)
		.filter((locale): locale is string => locale !== null)
		.filter((locale) =>
			SUPPORTED_LANGUAGES.has(locale.split("-")[0]?.toLowerCase()),
		);

	if (valid.length === 0) {
		return null;
	}

	try {
		const matched = match(valid, SUPPORTED_LOCALES, DEFAULT_LOCALE);

		return isLocale(matched) ? matched : null;
	} catch {
		return null;
	}
}

export function negotiate(search = window.location.search): Locale {
	const requestedInUrl = new URLSearchParams(search).get("lang");
	if (requestedInUrl) {
		const matched = bestMatch([requestedInUrl]);
		if (matched) {
			return matched;
		}
	}

	const storedPreference = readCookie(LOCALE_COOKIE);
	if (storedPreference) {
		const matched = bestMatch([storedPreference]);
		if (matched) {
			return matched;
		}
	}

	if (navigator.languages.length > 0) {
		const matched = bestMatch(navigator.languages);
		if (matched) {
			return matched;
		}
	}

	return DEFAULT_LOCALE;
}

function writePreferenceCookie(name: string, locale: Locale) {
	// biome-ignore lint/suspicious/noDocumentCookie: Cookie Store API is async; preference reads need a synchronous value before first paint.
	document.cookie = `${name}=${encodeURIComponent(locale)}; path=/; max-age=${LOCALE_COOKIE_MAX_AGE_SECONDS}; SameSite=Lax`;
}

export function writeLocaleCookie(locale: Locale) {
	writePreferenceCookie(LOCALE_COOKIE, locale);
}

export function readFormattingLocalePreference(): Locale | null {
	const storedPreference = readCookie(FORMATTING_LOCALE_COOKIE);
	if (!storedPreference) {
		return null;
	}

	return bestMatch([storedPreference]);
}

export function writeFormattingLocaleCookie(locale: Locale) {
	writePreferenceCookie(FORMATTING_LOCALE_COOKIE, locale);
}

export function updateLocaleInUrl(locale: Locale) {
	const url = new URL(window.location.href);

	if (!url.searchParams.has("lang")) {
		return;
	}

	url.searchParams.set("lang", locale);
	window.history.replaceState(
		window.history.state,
		"",
		`${url.pathname}${url.search}${url.hash}`,
	);
}
