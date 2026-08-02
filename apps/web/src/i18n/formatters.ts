const LATIN_NUMBERING_SYSTEM = "latn";

type MoneyOptions = {
	useAccountingNegatives?: boolean;
	hideFractionDigits?: boolean;
};

type DateStyle = "short" | "medium" | "long" | "full";

type ListType = "conjunction" | "disjunction";

export type Formatters = ReturnType<typeof createFormatters>;

export function createFormatters(locale: string) {
	const numberFormats = new Map<string, Intl.NumberFormat>();
	const dateFormats = new Map<string, Intl.DateTimeFormat>();
	const relativeTimeFormat = new Intl.RelativeTimeFormat(locale, {
		numeric: "auto",
	});
	const collator = new Intl.Collator(locale, {
		numeric: true,
		sensitivity: "base",
	});

	function numberFormat(options: Intl.NumberFormatOptions): Intl.NumberFormat {
		const key = JSON.stringify(options);
		const cached = numberFormats.get(key);

		if (cached) {
			return cached;
		}

		const created = new Intl.NumberFormat(locale, {
			numberingSystem: LATIN_NUMBERING_SYSTEM,
			...options,
		});
		numberFormats.set(key, created);

		return created;
	}

	function dateFormat(
		options: Intl.DateTimeFormatOptions,
	): Intl.DateTimeFormat {
		const key = JSON.stringify(options);
		const cached = dateFormats.get(key);

		if (cached) {
			return cached;
		}

		const created = new Intl.DateTimeFormat(locale, {
			numberingSystem: LATIN_NUMBERING_SYSTEM,
			...options,
		});
		dateFormats.set(key, created);

		return created;
	}

	function currencyFractionDigits(currency: string): number {
		return (
			numberFormat({ style: "currency", currency }).resolvedOptions()
				.maximumFractionDigits ?? 2
		);
	}

	return {
		locale,

		money(minorUnits: number, currency: string, options?: MoneyOptions) {
			const fractionDigits = currencyFractionDigits(currency);

			return numberFormat({
				style: "currency",
				currency,
				currencySign: options?.useAccountingNegatives
					? "accounting"
					: "standard",
				...(options?.hideFractionDigits
					? { maximumFractionDigits: 0, minimumFractionDigits: 0 }
					: {}),
			}).format(minorUnits / 10 ** fractionDigits);
		},

		number(value: number, options: Intl.NumberFormatOptions = {}) {
			return numberFormat(options).format(value);
		},

		percent(value: number, options: Intl.NumberFormatOptions = {}) {
			return numberFormat({ style: "percent", ...options }).format(value);
		},

		dateInZone(
			value: Date | number,
			timeZone: string,
			dateStyle: DateStyle = "medium",
		) {
			return dateFormat({ timeZone, dateStyle }).format(value);
		},

		dateTimeInZone(
			value: Date | number,
			timeZone: string,
			dateStyle: DateStyle = "medium",
		) {
			return dateFormat({ timeZone, dateStyle, timeStyle: "short" }).format(
				value,
			);
		},

		relativeTime(value: number, unit: Intl.RelativeTimeFormatUnit) {
			return relativeTimeFormat.format(value, unit);
		},

		list(items: string[], type: ListType = "conjunction") {
			return new Intl.ListFormat(locale, { type }).format(items);
		},

		compare(a: string, b: string) {
			return collator.compare(a, b);
		},

		sort<T>(items: readonly T[], key: (item: T) => string): T[] {
			return [...items].sort((a, b) => collator.compare(key(a), key(b)));
		},
	};
}
