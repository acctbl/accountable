import { getSerwist } from "virtual:serwist";
import { DirectionProvider } from "@base-ui/react/direction-provider";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import ReactDOM from "react-dom/client";
import { IntlProvider } from "react-intl";
import { ThemeProvider } from "@/components/theme-provider";
import { getDirection, SOURCE_LOCALE } from "@/i18n/config";
import {
	applyDocumentLocale,
	LocaleProvider,
	loadInitialLocaleState,
	useLocale,
} from "@/i18n/locale-provider";
import { routeTree } from "./routeTree.gen";
import "@/styles.css";

const router = createRouter({
	routeTree,
	defaultPreload: "intent",
	scrollRestoration: true,
});

declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}

function App() {
	const { locale, messages } = useLocale();

	return (
		<IntlProvider
			locale={locale}
			defaultLocale={SOURCE_LOCALE}
			messages={messages}
		>
			<DirectionProvider direction={getDirection(locale)}>
				<ThemeProvider defaultTheme="system" storageKey="acctbl-theme">
					<RouterProvider router={router} />
				</ThemeProvider>
			</DirectionProvider>
		</IntlProvider>
	);
}

async function registerSerwist() {
	if (!("serviceWorker" in navigator)) {
		return;
	}

	const serwist = await getSerwist();
	await serwist?.register();
}

void registerSerwist();

async function mountApp() {
	const rootElement = document.getElementById("root");
	if (!rootElement || rootElement.innerHTML) {
		return;
	}

	const initialState = await loadInitialLocaleState();
	if (initialState.loadError) {
		console.error(
			"Failed to load initial locale catalog",
			initialState.loadError,
		);
	}

	applyDocumentLocale(initialState.bundle.locale);

	ReactDOM.createRoot(rootElement).render(
		<LocaleProvider initialState={initialState}>
			<App />
		</LocaleProvider>,
	);
}

void mountApp();
