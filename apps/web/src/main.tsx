import { getSerwist } from "virtual:serwist";
import { DirectionProvider } from "@base-ui/react/direction-provider";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import * as React from "react";
import ReactDOM from "react-dom/client";
import { IntlProvider } from "react-intl";
import { AppErrorBoundary } from "@/components/app-error-boundary";
import { ThemeProvider } from "@/components/theme-provider";
import { getDirection, SOURCE_LOCALE } from "@/i18n/config";
import {
	applyDocumentLocale,
	LocaleProvider,
	loadInitialLocaleState,
	useLocale,
} from "@/i18n/locale-provider";
import { RegionalSettingsProvider } from "@/i18n/regional-settings";
import { type ApiClients, ApiProvider, createApiClients } from "@/lib/api";
import { loadRuntimeConfig } from "@/lib/runtime-config";
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

function App({
	clients,
	queryClient,
}: {
	clients: ApiClients;
	queryClient: QueryClient;
}) {
	const { locale, messages } = useLocale();
	const regionalSettings = React.useMemo(
		() => ({
			timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
			currency: null,
			jurisdiction: null,
		}),
		[],
	);
	React.useEffect(() => {
		performance.mark("accountable:application-ready");
	}, []);

	return (
		<IntlProvider
			locale={locale}
			defaultLocale={SOURCE_LOCALE}
			messages={messages}
		>
			<DirectionProvider direction={getDirection(locale)}>
				<ThemeProvider defaultTheme="system" storageKey="acctbl-theme">
					<RegionalSettingsProvider value={regionalSettings}>
						<AppErrorBoundary>
							<ApiProvider clients={clients}>
								<QueryClientProvider client={queryClient}>
									<RouterProvider router={router} />
								</QueryClientProvider>
							</ApiProvider>
						</AppErrorBoundary>
					</RegionalSettingsProvider>
				</ThemeProvider>
			</DirectionProvider>
		</IntlProvider>
	);
}

async function registerSerwist() {
	if (
		import.meta.env.DEV ||
		location.hostname === "localhost" ||
		location.hostname === "127.0.0.1" ||
		!("serviceWorker" in navigator)
	) {
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

	const [initialState, runtimeConfig] = await Promise.all([
		loadInitialLocaleState(),
		loadRuntimeConfig(),
	]);
	if (initialState.loadError) {
		console.error(
			"Failed to load initial locale catalog",
			initialState.loadError,
		);
	}

	applyDocumentLocale(initialState.bundle.locale);
	const clients = createApiClients(runtimeConfig);
	if (runtimeConfig.architecture_probe) {
		const { installArchitectureProbeBridge } = await import(
			"./lib/architecture-probe-bridge"
		);
		installArchitectureProbeBridge(runtimeConfig);
	}
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false, staleTime: 30_000 } },
	});

	ReactDOM.createRoot(rootElement).render(
		<LocaleProvider initialState={initialState}>
			<App clients={clients} queryClient={queryClient} />
		</LocaleProvider>,
	);
}

void mountApp().catch(() => {
	const rootElement = document.getElementById("root");
	if (!rootElement) return;
	rootElement.innerHTML = `
		<main id="main-content" class="shell-main">
			<h1>Configuration unavailable</h1>
			<p role="alert">The shell could not load its public runtime configuration.</p>
			<button type="button" onclick="location.reload()">Reload</button>
		</main>`;
});
