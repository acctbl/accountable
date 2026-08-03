import { SystemService } from "@accountable/proto/accountable/system/v1/system_pb";
import {
	type Client,
	createClient,
	type Interceptor,
	type Transport,
} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import * as React from "react";
import type { RuntimeConfig } from "./runtime-config";

export type ApiClients = {
	system: Client<typeof SystemService>;
};

const ApiContext = React.createContext<ApiClients | null>(null);

function correlationAndLocaleInterceptor(): Interceptor {
	return (next) => async (request) => {
		const activeLocale = document.documentElement.lang;
		if (activeLocale) {
			request.header.set("Accept-Language", activeLocale);
		}
		if (!request.header.has("X-Trace-ID")) {
			request.header.set("X-Trace-ID", crypto.randomUUID());
		}
		return next(request);
	};
}

export function createApiTransport(config: RuntimeConfig): Transport {
	const includeCredentials: typeof fetch = (input, init) =>
		fetch(input, { ...init, credentials: "include" });
	return createConnectTransport({
		baseUrl: config.api_base_url,
		useBinaryFormat: true,
		fetch: includeCredentials,
		interceptors: [correlationAndLocaleInterceptor()],
	});
}

export function createApiClients(config: RuntimeConfig): ApiClients {
	const transport = createApiTransport(config);
	return {
		system: createClient(SystemService, transport),
	};
}

export function ApiProvider({
	children,
	clients,
}: {
	children: React.ReactNode;
	clients: ApiClients;
}) {
	return <ApiContext.Provider value={clients}>{children}</ApiContext.Provider>;
}

export function useApi(): ApiClients {
	const clients = React.useContext(ApiContext);
	if (!clients) {
		throw new Error("useApi must be used within ApiProvider");
	}
	return clients;
}
