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
			localeBootstrap(includePseudoLocales),
			tailwindcss(),
			tanstackRouter({ target: "react", autoCodeSplitting: true }),
			viteReact(),
			serwist({
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
