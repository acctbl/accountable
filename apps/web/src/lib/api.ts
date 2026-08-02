import { SystemService } from "@accountable/proto/accountable/system/v1/system_pb";
import { createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const sendActiveLocale: Interceptor = (next) => async (request) => {
	const activeLocale = document.documentElement.lang;

	if (activeLocale) {
		request.header.set("Accept-Language", activeLocale);
	}

	return next(request);
};

const transport = createConnectTransport({
	baseUrl: "",
	useBinaryFormat: true,
	interceptors: [sendActiveLocale],
});

export const systemClient = createClient(SystemService, transport);
