import * as React from "react";

// These values deliberately exclude interface language and formatting locale,
// which are owned by LocaleProvider. They must never be inferred from one
// another: each has its own source and may be unset independently.
export type RegionalSettings = {
	timeZone: string;
	currency: string | null;
	jurisdiction: string | null;
};

const RegionalSettingsContext = React.createContext<RegionalSettings | null>(
	null,
);

export function RegionalSettingsProvider({
	children,
	value,
}: {
	children: React.ReactNode;
	value: RegionalSettings;
}) {
	return (
		<RegionalSettingsContext.Provider value={value}>
			{children}
		</RegionalSettingsContext.Provider>
	);
}

export function useRegionalSettings(): RegionalSettings {
	const settings = React.useContext(RegionalSettingsContext);
	if (!settings) {
		throw new Error(
			"useRegionalSettings must be used within RegionalSettingsProvider",
		);
	}
	return settings;
}
