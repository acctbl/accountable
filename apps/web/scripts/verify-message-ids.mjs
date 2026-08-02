import { readFileSync } from "node:fs";

const CATALOG = "src/i18n/messages/en.json";
const NAMESPACED_ID = /^[a-z][a-zA-Z0-9]*(\.[a-zA-Z0-9]+)+$/;

const catalog = JSON.parse(readFileSync(CATALOG, "utf8"));
const generatedIds = Object.keys(catalog).filter((id) => !NAMESPACED_ID.test(id));

if (generatedIds.length > 0) {
	console.error(
		`${CATALOG}: ${generatedIds.length} message(s) have no explicit id.\n\n` +
			`formatjs falls back to a content hash when a message omits "id". The id is\n` +
			`the shared key across web, mobile and server, so a hash means editing the\n` +
			`English copy silently orphans that message's translations on every client.\n\n` +
			`Give each of these an explicit id of the form "screen.thing":\n` +
			generatedIds.map((id) => `  ${id}`).join("\n"),
	);
	process.exit(1);
}

console.log(`${CATALOG}: ${Object.keys(catalog).length} message id(s) OK`);
