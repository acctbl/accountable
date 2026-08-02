#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

node scripts/security/exceptions-to-allowlists.mjs
CONFIG=".security/allowlists/gitleaks.toml"

gitleaks git \
	--config "$CONFIG" \
	--redact \
	--verbose \
	--exit-code 1
