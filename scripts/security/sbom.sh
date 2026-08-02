#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

mkdir -p dist
OUT="${SBOM_OUT:-dist/sbom.cdx.json}"
TARGET="${SBOM_TARGET:-.}"

trivy fs \
	--format cyclonedx \
	--include-dev-deps \
	--output "$OUT" \
	--skip-dirs node_modules,.git,.pnpm-store,dist,test-results,playwright-report,testdata \
	"$TARGET"

[[ -s "$OUT" ]] || {
	echo "sbom: empty output at $OUT" >&2
	exit 1
}

node --input-type=module - "$OUT" <<'EOF'
import { readFileSync } from "node:fs";

const out = process.argv[2];
const doc = JSON.parse(readFileSync(out, "utf8"));
const components = Array.isArray(doc.components) ? doc.components.length : 0;
if (components < 1) {
	console.error(`sbom: expected at least one component in ${out}`);
	process.exit(1);
}
console.log(`sbom: wrote ${out} (${components} components)`);
EOF
