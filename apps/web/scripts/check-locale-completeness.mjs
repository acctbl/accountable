import { readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const messagesDir = join(dirname(fileURLToPath(import.meta.url)), "../src/i18n/messages");
const en = JSON.parse(readFileSync(join(messagesDir, "en.json"), "utf8"));
const enKeys = new Set(Object.keys(en));

let failed = false;

for (const file of readdirSync(messagesDir).sort()) {
	if (!file.endsWith(".json") || file === "en.json" || file.includes("-")) {
		continue;
	}

	const catalog = JSON.parse(readFileSync(join(messagesDir, file), "utf8"));
	const keys = new Set(Object.keys(catalog));
	const missing = [...enKeys].filter((key) => !keys.has(key));
	const extra = [...keys].filter((key) => !enKeys.has(key));

	if (missing.length > 0 || extra.length > 0) {
		failed = true;
		console.error(`${file}: translation catalog does not match en.json`);
		if (missing.length > 0) {
			console.error(`  missing: ${missing.join(", ")}`);
		}
		if (extra.length > 0) {
			console.error(`  extra: ${extra.join(", ")}`);
		}
	}
}

if (failed) {
	process.exit(1);
}

console.log("Translation catalogs match en.json");
