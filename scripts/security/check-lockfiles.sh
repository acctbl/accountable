#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

fail() {
	echo "lockfile-check: $*" >&2
	exit 1
}

for path in go.sum pnpm-lock.yaml mise.lock; do
	[[ -f "$path" ]] || fail "missing required lockfile: $path"
done

[[ -f go.mod ]] || fail "missing go.mod"
grep -q '^go ' go.mod || fail "go.mod has no go version directive"

latest_hits="$(
	find . \
		\( -path ./node_modules -o -path ./.git -o -path ./.pnpm-store -o -path ./dist -o -path ./testdata \) -prune -o \
		-type f \( -name 'Dockerfile*' -o -name 'docker-compose*.yml' -o -name 'docker-compose*.yaml' -o -name '*.tf' -o -path './.github/workflows/*' \) -print0 |
		xargs -0 grep -nE ':latest(["'\''[:space:]]|$)|[[:space:]]latest[[:space:]]*$' 2>/dev/null || true
)"

if [[ -n "$latest_hits" ]]; then
	echo "$latest_hits" >&2
	fail "floating ':latest' image or tag reference found"
fi

if [[ -n "${CI:-}" ]]; then
	pnpm install --frozen-lockfile --ignore-scripts >/dev/null
	mise install --locked >/dev/null
fi

echo "lockfile-check: ok"
