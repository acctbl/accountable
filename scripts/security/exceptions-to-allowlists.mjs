#!/usr/bin/env node
import { mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const EXCEPTIONS_DIR = join(ROOT, "docs", "exceptions");
const OUT_DIR = join(ROOT, ".security", "allowlists");
const TODAY = new Date().toISOString().slice(0, 10);

const ALLOWED_TOOLS = new Set(["gitleaks", "govulncheck", "trivy", "trivy-iac"]);
const ALLOWED_KINDS = new Set(["regex", "path", "cve", "vuln_id", "misconfig_id"]);
const REQUIRED = ["tool", "finding", "kind", "expires", "owner", "reason", "residual_risk"];

function parseFrontmatter(text, path) {
	if (!text.startsWith("---\n")) {
		throw new Error(`${path}: missing YAML frontmatter`);
	}
	const end = text.indexOf("\n---\n", 4);
	if (end === -1) {
		throw new Error(`${path}: unterminated YAML frontmatter`);
	}
	const data = {};
	for (const line of text.slice(4, end).split("\n")) {
		if (!line.trim() || line.trimStart().startsWith("#")) {
			continue;
		}
		const idx = line.indexOf(":");
		if (idx === -1) {
			throw new Error(`${path}: invalid frontmatter line: ${line}`);
		}
		const key = line.slice(0, idx).trim();
		let value = line.slice(idx + 1).trim();
		if (
			(value.startsWith('"') && value.endsWith('"')) ||
			(value.startsWith("'") && value.endsWith("'"))
		) {
			value = value.slice(1, -1);
		}
		const comment = value.indexOf(" #");
		if (comment !== -1) {
			value = value.slice(0, comment).trim();
		}
		data[key] = value;
	}
	return data;
}

function validate(data, path) {
	for (const key of REQUIRED) {
		if (!data[key]) {
			throw new Error(`${path}: missing required field '${key}'`);
		}
	}
	if (!ALLOWED_TOOLS.has(data.tool)) {
		throw new Error(`${path}: unsupported tool '${data.tool}'`);
	}
	if (!ALLOWED_KINDS.has(data.kind)) {
		throw new Error(`${path}: unsupported kind '${data.kind}'`);
	}
	if (!/^\d{4}-\d{2}-\d{2}$/.test(data.expires)) {
		throw new Error(`${path}: expires must be YYYY-MM-DD`);
	}
	if (data.expires < TODAY) {
		throw new Error(`${path}: exception expired on ${data.expires}`);
	}
}

function tomlString(value) {
	return `'''${String(value).replace(/'''/g, "''\\'")}'''`;
}

function main() {
	rmSync(OUT_DIR, { recursive: true, force: true });
	mkdirSync(OUT_DIR, { recursive: true });

	const files = readdirSync(EXCEPTIONS_DIR)
		.filter((name) => name.endsWith(".md") && name !== "README.md")
		.sort();

	const gitleaksRegexes = [];
	const gitleaksPaths = ["(?:^|/)testdata/negative(?:/.*)?$"];
	const trivyLines = [];
	const govulnIds = [];
	let active = 0;

	for (const name of files) {
		const path = join(EXCEPTIONS_DIR, name);
		const data = parseFrontmatter(readFileSync(path, "utf8"), path);
		validate(data, path);
		active += 1;

		switch (data.tool) {
			case "gitleaks": {
				if (data.kind === "regex") {
					gitleaksRegexes.push(data.finding);
				} else if (data.kind === "path") {
					gitleaksPaths.push(data.finding);
				} else {
					throw new Error(`${path}: gitleaks kind must be regex or path`);
				}
				break;
			}
			case "govulncheck": {
				if (data.kind !== "vuln_id") {
					throw new Error(`${path}: govulncheck kind must be vuln_id`);
				}
				govulnIds.push(data.finding);
				break;
			}
			case "trivy":
			case "trivy-iac": {
				if (data.kind !== "cve" && data.kind !== "misconfig_id") {
					throw new Error(`${path}: trivy kind must be cve or misconfig_id`);
				}
				trivyLines.push(`${data.finding} exp:${data.expires}`);
				break;
			}
			default:
				throw new Error(`${path}: unhandled tool`);
		}
	}

	const gitleaks = [
		"title = 'accountable'",
		"",
		"[extend]",
		"useDefault = true",
		"",
		"[[allowlists]]",
		"description = 'Negative-proof fixtures and dated docs/exceptions'",
		"paths = [",
		...gitleaksPaths.map((value) => `\t${tomlString(value)},`),
		"]",
		...(gitleaksRegexes.length
			? ["regexes = [", ...gitleaksRegexes.map((value) => `\t${tomlString(value)},`), "]"]
			: []),
		"",
	].join("\n");
	writeFileSync(join(OUT_DIR, "gitleaks.toml"), gitleaks);

	writeFileSync(join(OUT_DIR, "trivyignore"), `${trivyLines.join("\n")}${trivyLines.length ? "\n" : ""}`);

	writeFileSync(
		join(OUT_DIR, "govulncheck-exclude.txt"),
		govulnIds.length ? `${govulnIds.join("\n")}\n` : "",
	);

	writeFileSync(
		join(OUT_DIR, "manifest.json"),
		`${JSON.stringify({ generated_at: new Date().toISOString(), active_exceptions: active, files }, null, 2)}\n`,
	);

	console.log(`Wrote ${active} active exception(s) to ${OUT_DIR}`);
}

try {
	main();
} catch (error) {
	console.error(error instanceof Error ? error.message : error);
	process.exit(1);
}
