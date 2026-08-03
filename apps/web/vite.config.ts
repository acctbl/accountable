import { serwist } from "@serwist/vite";
import tailwindcss from "@tailwindcss/vite";

import { tanstackRouter } from "@tanstack/router-plugin/vite";

import viteReact from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";
import { localeBootstrapScript } from "./src/i18n/bootstrap";

function localeBootstrap(includePseudoLocales: boolean): Plugin {
	return {
		name: "accountable:locale-bootstrap",
		transformIndexHtml() {
			return [
				{
					tag: "script",
					children: localeBootstrapScript(includePseudoLocales),
					injectTo: "head",
				},
			];
		},
	};
}

function developmentRuntimeConfig(): Plugin {
	const body = JSON.stringify({
		schema_version: 1,
		api_base_url: "http://localhost:8080",
		architecture_probe: false,
		configuration_revision: "local-development",
	});
	const serveConfig = (
		_request: unknown,
		response: import("node:http").ServerResponse,
	) => {
		response.statusCode = 200;
		response.setHeader("Cache-Control", "no-store");
		response.setHeader("Content-Type", "application/json; charset=utf-8");
		response.end(body);
	};
	return {
		name: "accountable:runtime-config",
		configureServer(server) {
			server.middlewares.use("/_runtime/config.json", serveConfig);
		},
	};
}

const config = defineConfig(({ command }) => {
	const includePseudoLocales = command === "serve";

	return {
		resolve: { tsconfigPaths: true },
		server: {
			proxy: {
				"/accountable.system.v1.SystemService": {
					target: "http://localhost:8080",
				},
			},
		},
		plugins: [
			developmentRuntimeConfig(),
			localeBootstrap(includePseudoLocales),
			tailwindcss(),
			tanstackRouter({ target: "react", autoCodeSplitting: true }),
			viteReact(),
			serwist({
				disable: command === "serve",
				swSrc: "src/sw.ts",
				swDest: "sw.js",
				globDirectory: "dist",
				injectionPoint: "self.__SW_MANIFEST",
				rollupFormat: "iife",
			}),
		],
	};
});

export default config;
