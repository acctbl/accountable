export type RuntimeConfig = {
	schema_version: 1;
	api_base_url: string;
	architecture_probe: boolean;
	configuration_revision: string;
};

const keys = [
	"api_base_url",
	"architecture_probe",
	"configuration_revision",
	"schema_version",
] as const;

export function parseRuntimeConfig(value: unknown): RuntimeConfig {
	if (typeof value !== "object" || value === null || Array.isArray(value)) {
		throw new Error("Runtime configuration must be an object");
	}
	const record = value as Record<string, unknown>;
	if (Object.keys(record).sort().join("|") !== [...keys].sort().join("|")) {
		throw new Error("Runtime configuration has unknown or missing fields");
	}
	if (record.schema_version !== 1) {
		throw new Error("Unsupported runtime configuration schema");
	}
	if (typeof record.architecture_probe !== "boolean") {
		throw new Error("Invalid architecture probe flag");
	}
	if (
		typeof record.configuration_revision !== "string" ||
		!/^[-A-Za-z0-9._]{1,128}$/.test(record.configuration_revision)
	) {
		throw new Error("Invalid runtime configuration revision");
	}
	if (typeof record.api_base_url !== "string") {
		throw new Error("Invalid API base URL");
	}
	let apiUrl: URL;
	try {
		apiUrl = new URL(record.api_base_url);
	} catch {
		throw new Error("Invalid API base URL");
	}
	if (
		!(["http:", "https:"] as const).includes(
			apiUrl.protocol as "http:" | "https:",
		) ||
		apiUrl.username !== "" ||
		apiUrl.password !== "" ||
		apiUrl.search !== "" ||
		apiUrl.hash !== ""
	) {
		throw new Error("Unsafe API base URL");
	}
	return {
		schema_version: 1,
		api_base_url: apiUrl.toString().replace(/\/$/, ""),
		architecture_probe: record.architecture_probe,
		configuration_revision: record.configuration_revision,
	};
}

export async function loadRuntimeConfig(
	fetcher: typeof fetch = fetch,
): Promise<RuntimeConfig> {
	const response = await fetcher("/_runtime/config.json", {
		cache: "no-store",
		credentials: "same-origin",
		headers: { Accept: "application/json" },
	});
	if (!response.ok) {
		throw new Error("Runtime configuration is unavailable");
	}
	return parseRuntimeConfig(await response.json());
}
