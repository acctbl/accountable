import { cn } from "@accountable/ui/lib/utils";
import { useId } from "react";
import { useIntl } from "react-intl";
import { isLocale, type Locale, SELECTABLE_LOCALES } from "@/i18n/config";
import { useLocale } from "@/i18n/locale-provider";

function autonymOf(locale: Locale): string {
	const displayNames = new Intl.DisplayNames([locale], { type: "language" });

	return displayNames.of(locale) ?? locale;
}

export function LocaleSwitcher({ className }: { className?: string }) {
	const { loadError, locale, pendingLocale, setLocale } = useLocale();
	const intl = useIntl();
	const errorId = useId();
	const displayedLocale = pendingLocale ?? locale;
	const isSelectable = SELECTABLE_LOCALES.some(
		(selectable) => selectable === displayedLocale,
	);
	const loadErrorMessage = intl.formatMessage({
		id: "localeSwitcher.loadError",
		defaultMessage: "That language could not be loaded. Please try again.",
	});

	return (
		<>
			<select
				aria-busy={pendingLocale ? true : undefined}
				aria-describedby={loadError ? errorId : undefined}
				aria-invalid={loadError ? true : undefined}
				aria-label={intl.formatMessage({
					id: "localeSwitcher.label",
					defaultMessage: "Language",
				})}
				title={loadError ? loadErrorMessage : undefined}
				value={displayedLocale}
				onChange={(event) => {
					if (isLocale(event.target.value)) {
						setLocale(event.target.value);
					}
				}}
				className={cn(
					"h-7 rounded-md border border-border bg-transparent px-2 text-xs/relaxed outline-none",
					"focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30",
					className,
				)}
			>
				{!isSelectable ? (
					<option value={displayedLocale} lang={displayedLocale}>
						{autonymOf(displayedLocale)}
					</option>
				) : null}
				{SELECTABLE_LOCALES.map((selectable) => (
					<option key={selectable} value={selectable} lang={selectable}>
						{autonymOf(selectable)}
					</option>
				))}
			</select>
			<span id={errorId} className="sr-only" role="status">
				{loadError ? loadErrorMessage : null}
			</span>
		</>
	);
}
