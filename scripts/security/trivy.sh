#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

node scripts/security/exceptions-to-allowlists.mjs
IGNORE=".security/allowlists/trivyignore"

trivy fs \
	--scanners vuln \
	--include-dev-deps \
	--severity HIGH,CRITICAL \
	--exit-code 1 \
	--ignorefile "$IGNORE" \
	--skip-dirs node_modules,.git,.pnpm-store,dist,test-results,playwright-report,testdata \
	.

trivy config \
	--severity HIGH,CRITICAL \
	--exit-code 1 \
	--ignorefile "$IGNORE" \
	infra
