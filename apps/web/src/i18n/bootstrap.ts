import { DEFAULT_LOCALE, PSEUDO_LOCALES, SELECTABLE_LOCALES } from "./settings";

export const LOCALE_COOKIE = "acctbl-locale";
export const FORMATTING_LOCALE_COOKIE = "acctbl-formatting-locale";

const RIGHT_TO_LEFT_LANGUAGES = ["ar", "fa", "he", "ur"];

const RIGHT_TO_LEFT_PSEUDO_LOCALES = ["en-XB"];

export function languageOf(locale: string): string {
	return locale.split("-")[0]?.toLowerCase() ?? locale;
}

export function getDirection(locale: string): "ltr" | "rtl" {
	if (RIGHT_TO_LEFT_PSEUDO_LOCALES.includes(locale)) {
		return "rtl";
	}

	return RIGHT_TO_LEFT_LANGUAGES.includes(languageOf(locale)) ? "rtl" : "ltr";
}

export function localeBootstrapScript(includePseudoLocales = false): string {
	const rightToLeftLanguages = JSON.stringify(RIGHT_TO_LEFT_LANGUAGES);
	const rightToLeftPseudoLocales = JSON.stringify(RIGHT_TO_LEFT_PSEUDO_LOCALES);
	const supportedLocales = JSON.stringify([
		...SELECTABLE_LOCALES,
		...(includePseudoLocales ? PSEUDO_LOCALES : []),
	]);
	const defaultLocale = JSON.stringify(DEFAULT_LOCALE);

	return `
(() => {
  const supported = ${supportedLocales};
  const supportedLanguages = new Set(
    supported.map((locale) => locale.split("-")[0].toLowerCase()),
  );

  const canonicalize = (value) => {
    if (!value) return null;

    try {
      const [canonical] = Intl.getCanonicalLocales(value);
      return canonical ?? null;
    } catch {
      return null;
    }
  };

  const bestSupported = (requested) => {
    const canonical = requested
      .map(canonicalize)
      .filter((locale) => locale !== null);

    for (const locale of canonical) {
      const exact = supported.find(
        (supportedLocale) => supportedLocale.toLowerCase() === locale.toLowerCase(),
      );
      if (exact) return exact;

      const language = locale.split("-")[0].toLowerCase();
      if (!supportedLanguages.has(language)) continue;

      const languageMatch = supported.find(
        (supportedLocale) =>
          supportedLocale.split("-")[0].toLowerCase() === language,
      );
      if (languageMatch) return languageMatch;
    }

    return null;
  };

  const stored = document.cookie.match(/(?:^|; )${LOCALE_COOKIE}=([^;]*)/);
  let storedLocale = null;
  if (stored) {
    try {
      storedLocale = decodeURIComponent(stored[1]);
    } catch {
      storedLocale = null;
    }
  }

  const locale =
    bestSupported([new URLSearchParams(window.location.search).get("lang")]) ??
    bestSupported([storedLocale]) ??
    bestSupported(window.navigator.languages) ??
    ${defaultLocale};
  const language = locale.split("-")[0].toLowerCase();
  const rightToLeft =
    ${rightToLeftPseudoLocales}.includes(locale) ||
    ${rightToLeftLanguages}.includes(language);

  document.documentElement.lang = locale;
  document.documentElement.dir = rightToLeft ? "rtl" : "ltr";
})();
`.trim();
}
