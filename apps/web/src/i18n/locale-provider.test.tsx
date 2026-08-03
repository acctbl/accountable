import {
	act,
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { FormattedMessage, IntlProvider } from "react-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FORMATTING_LOCALE_COOKIE, LOCALE_COOKIE } from "./bootstrap";
import { SOURCE_LOCALE } from "./config";
import {
	type InitialLocaleState,
	type LocaleBundle,
	LocaleProvider,
	loadInitialLocaleState,
	loadLocaleBundle,
	useLocale,
} from "./locale-provider";
import { useFormatters } from "./use-formatters";

function Subject({ controls = false }: { controls?: boolean }) {
	const {
		formattingLocale,
		loadError,
		locale,
		messages,
		pendingLocale,
		setFormattingLocale,
		setLocale,
	} = useLocale();
	const formatters = useFormatters();

	return (
		<IntlProvider
			locale={locale}
			defaultLocale={SOURCE_LOCALE}
			messages={messages}
			onError={() => {}}
		>
			<h1>
				<FormattedMessage
					id="home.description"
					defaultMessage="Local transport shell"
				/>
			</h1>
			<output data-testid="locale">{locale}</output>
			<output data-testid="formatting-locale">{formattingLocale}</output>
			<output data-testid="formatted-list">
				{formatters.list(["a", "b", "c"])}
			</output>
			<output data-testid="pending">{pendingLocale}</output>
			<output data-testid="error">{loadError?.message}</output>
			{controls ? (
				<>
					<button type="button" onClick={() => setLocale("ar")}>
						Arabic
					</button>
					<button type="button" onClick={() => setLocale("en-US")}>
						English
					</button>
					<button type="button" onClick={() => setFormattingLocale("en-GB")}>
						Format GB
					</button>
				</>
			) : null}
		</IntlProvider>
	);
}

async function renderAt(search: string) {
	window.history.replaceState({}, "", `/${search}`);
	const initialState = await loadInitialLocaleState();

	return render(
		<LocaleProvider initialState={initialState}>
			<Subject />
		</LocaleProvider>,
	);
}

function renderWithLoader(
	initialState: InitialLocaleState,
	loadBundle: (locale: LocaleBundle["locale"]) => Promise<LocaleBundle>,
) {
	return render(
		<LocaleProvider initialState={initialState} loadBundle={loadBundle}>
			<Subject controls />
		</LocaleProvider>,
	);
}

beforeEach(() => {
	Object.defineProperty(document, "cookie", {
		value: "",
		configurable: true,
		writable: true,
	});
	window.history.replaceState({}, "", "/");
});

afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
	document.documentElement.lang = "";
	document.documentElement.dir = "";
});

describe("LocaleProvider", () => {
	it("renders the negotiated catalog and direction on its first render", async () => {
		await renderAt("?lang=ar");

		expect(screen.getByRole("heading").textContent).toBe("واجهة النقل المحلية");
		expect(document.documentElement.lang).toBe("ar");
		expect(document.documentElement.dir).toBe("rtl");
	});

	it("renders regional english with the source catalog", async () => {
		await renderAt("?lang=en-GB");

		expect(screen.getByRole("heading").textContent).toBe(
			"Local transport shell",
		);
		expect(document.documentElement.lang).toBe("en-GB");
		expect(document.documentElement.dir).toBe("ltr");
	});

	it("supports both development pseudo-locales", async () => {
		await renderAt("?lang=en-XA");
		expect(screen.getByRole("heading").textContent).toMatch(/^\[.*]$/);
		expect(document.documentElement.dir).toBe("ltr");
		cleanup();

		await renderAt("?lang=en-XB");
		expect(document.documentElement.dir).toBe("rtl");
	});

	it("does not persist a locale selected only through the URL", async () => {
		await renderAt("?lang=ar");

		expect(document.cookie).not.toContain(LOCALE_COOKIE);
	});

	it("keeps locale and messages atomic while a new catalog loads", async () => {
		const initialState: InitialLocaleState = {
			bundle: await loadLocaleBundle("en-US"),
			formattingLocalePreference: null,
			loadError: null,
		};
		const arabicBundle = await loadLocaleBundle("ar");
		let resolveBundle: (bundle: LocaleBundle) => void = () => {};
		const loadBundle = vi.fn(
			() =>
				new Promise<LocaleBundle>((resolve) => {
					resolveBundle = resolve;
				}),
		);
		renderWithLoader(initialState, loadBundle);

		fireEvent.click(screen.getByRole("button", { name: "Arabic" }));

		expect(screen.getByTestId("locale").textContent).toBe("en-US");
		expect(screen.getByTestId("pending").textContent).toBe("ar");
		expect(screen.getByRole("heading").textContent).toBe(
			"Local transport shell",
		);

		await act(async () => {
			resolveBundle(arabicBundle);
			await Promise.resolve();
		});

		expect(screen.getByTestId("locale").textContent).toBe("ar");
		expect(screen.getByTestId("pending").textContent).toBe("");
		expect(screen.getByRole("heading").textContent).toBe("واجهة النقل المحلية");
	});

	it("keeps the active bundle on failure and allows retry", async () => {
		vi.spyOn(console, "error").mockImplementation(() => {});
		const initialState: InitialLocaleState = {
			bundle: await loadLocaleBundle("en-US"),
			formattingLocalePreference: null,
			loadError: null,
		};
		const arabicBundle = await loadLocaleBundle("ar");
		const loadBundle = vi
			.fn<(locale: LocaleBundle["locale"]) => Promise<LocaleBundle>>()
			.mockRejectedValueOnce(new Error("chunk unavailable"))
			.mockResolvedValueOnce(arabicBundle);
		renderWithLoader(initialState, loadBundle);

		fireEvent.click(screen.getByRole("button", { name: "Arabic" }));
		await waitFor(() => {
			expect(screen.getByTestId("error").textContent).toBe("chunk unavailable");
		});
		expect(screen.getByTestId("locale").textContent).toBe("en-US");
		expect(screen.getByRole("heading").textContent).toBe(
			"Local transport shell",
		);

		fireEvent.click(screen.getByRole("button", { name: "Arabic" }));
		await waitFor(() => {
			expect(screen.getByTestId("locale").textContent).toBe("ar");
		});
		expect(loadBundle).toHaveBeenCalledTimes(2);
		expect(screen.getByTestId("error").textContent).toBe("");
	});

	it("follows the interface locale for formatting until a preference is set", async () => {
		const initialState: InitialLocaleState = {
			bundle: await loadLocaleBundle("en-US"),
			formattingLocalePreference: null,
			loadError: null,
		};
		renderWithLoader(initialState, loadLocaleBundle);

		expect(screen.getByTestId("formatting-locale").textContent).toBe("en-US");

		fireEvent.click(screen.getByRole("button", { name: "Arabic" }));
		await waitFor(() => {
			expect(screen.getByTestId("locale").textContent).toBe("ar");
		});

		expect(screen.getByTestId("formatting-locale").textContent).toBe("ar");
		expect(document.cookie).not.toContain(FORMATTING_LOCALE_COOKIE);
	});

	it("keeps formatting locale independent from the interface language", async () => {
		const initialState: InitialLocaleState = {
			bundle: await loadLocaleBundle("ar"),
			formattingLocalePreference: "en-US",
			loadError: null,
		};
		renderWithLoader(initialState, loadLocaleBundle);

		expect(screen.getByRole("heading").textContent).toBe("واجهة النقل المحلية");
		expect(screen.getByTestId("locale").textContent).toBe("ar");
		expect(screen.getByTestId("formatting-locale").textContent).toBe("en-US");
		expect(screen.getByTestId("formatted-list").textContent).toBe(
			"a, b, and c",
		);

		fireEvent.click(screen.getByRole("button", { name: "Format GB" }));

		expect(screen.getByTestId("locale").textContent).toBe("ar");
		expect(screen.getByTestId("formatting-locale").textContent).toBe("en-GB");
		expect(screen.getByTestId("formatted-list").textContent).toBe("a, b and c");
		expect(document.cookie).toContain(`${FORMATTING_LOCALE_COOKIE}=en-GB`);
	});

	it("hydrates an explicit formatting locale preference from its cookie", async () => {
		Object.defineProperty(document, "cookie", {
			value: `${FORMATTING_LOCALE_COOKIE}=en-GB`,
			configurable: true,
			writable: true,
		});
		window.history.replaceState({}, "", "/?lang=ar");

		const initialState = await loadInitialLocaleState();
		render(
			<LocaleProvider initialState={initialState}>
				<Subject />
			</LocaleProvider>,
		);

		expect(screen.getByTestId("locale").textContent).toBe("ar");
		expect(screen.getByTestId("formatting-locale").textContent).toBe("en-GB");
	});
});
