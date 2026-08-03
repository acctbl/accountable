import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";

const catalog = "src/i18n/messages/en.json";
const before = readFileSync(catalog);
const extraction = spawnSync("pnpm", ["i18n:extract"], {
	stdio: "inherit",
	shell: process.platform === "win32",
});

if (extraction.error) {
	throw extraction.error;
}
if (extraction.status !== 0) {
	process.exit(extraction.status ?? 1);
}

const after = readFileSync(catalog);
if (!before.equals(after)) {
	console.error(`${catalog} was not current; run pnpm i18n:extract`);
	process.exit(1);
}

console.log(`${catalog} is current`);
