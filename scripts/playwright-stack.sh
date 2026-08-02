#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mise exec -- go run ./cmd/api &
API_PID=$!

cleanup() {
	kill "$API_PID" 2>/dev/null || true
	wait "$API_PID" 2>/dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 100); do
	if (echo >/dev/tcp/127.0.0.1/8080) >/dev/null 2>&1; then
		break
	fi
	sleep 0.1
done

pnpm --filter @accountable/web i18n:compile
exec pnpm --filter @accountable/web exec vite --host 127.0.0.1 --port 4173 --strictPort
