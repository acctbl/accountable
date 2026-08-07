#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
	echo "usage: with-test-postgres.sh <command> [args...]" >&2
	exit 2
fi

TEST_POSTGRES_PASSWORD="accountable-local-test-only"
TEST_POSTGRES_DATABASE="accountable"
POSTGRES_CONTAINER=""
POSTGRES_DATA=""
POSTGRES_BINDIR=""
POSTGRES_PORT=""

cleanup() {
	if [ -n "$POSTGRES_CONTAINER" ]; then
		docker rm --force "$POSTGRES_CONTAINER" >/dev/null 2>&1 || true
	fi
	if [ -n "$POSTGRES_DATA" ] && [ -n "$POSTGRES_BINDIR" ]; then
		"$POSTGRES_BINDIR/pg_ctl" -D "$POSTGRES_DATA" -m immediate stop >/dev/null 2>&1 || true
		rm -rf "$POSTGRES_DATA"
	fi
}
trap cleanup EXIT

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	: "${POSTGRES_TEST_IMAGE:?POSTGRES_TEST_IMAGE is required}"
	POSTGRES_CONTAINER="accountable-postgres-test-$$"
	docker run --detach --rm \
		--name "$POSTGRES_CONTAINER" \
		--env POSTGRES_PASSWORD="$TEST_POSTGRES_PASSWORD" \
		--env POSTGRES_DB="$TEST_POSTGRES_DATABASE" \
		--publish 127.0.0.1::5432 \
		"$POSTGRES_TEST_IMAGE" >/dev/null
	POSTGRES_PORT="$(docker inspect --format '{{(index (index .NetworkSettings.Ports "5432/tcp") 0).HostPort}}' "$POSTGRES_CONTAINER")"
	POSTGRES_READY="false"
	for _ in $(seq 1 300); do
		if docker exec "$POSTGRES_CONTAINER" pg_isready --host 127.0.0.1 --username postgres --dbname "$TEST_POSTGRES_DATABASE" >/dev/null 2>&1; then
			POSTGRES_READY="true"
			break
		fi
		sleep 0.1
	done
	if [ "$POSTGRES_READY" != "true" ]; then
		echo "disposable Postgres did not become ready" >&2
		exit 1
	fi
	export ACCOUNTABLE_TEST_POSTGRES_DSN="postgres://postgres:${TEST_POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/${TEST_POSTGRES_DATABASE}?sslmode=disable"
elif command -v pg_config >/dev/null 2>&1; then
	POSTGRES_BINDIR="$(pg_config --bindir)"
	POSTGRES_DATA="$(mktemp -d "${TMPDIR:-/tmp}/accountable-postgres.XXXXXX")"
	POSTGRES_PORT="$((55432 + ($$ % 1000)))"
	"$POSTGRES_BINDIR/initdb" -D "$POSTGRES_DATA" -A trust -U postgres >/dev/null
	if ! "$POSTGRES_BINDIR/pg_ctl" -D "$POSTGRES_DATA" \
		-o "-h 127.0.0.1 -p $POSTGRES_PORT -k $POSTGRES_DATA" -w start; then
		echo "local Postgres failed to start" >&2
		if [ -d "$POSTGRES_DATA/log" ]; then
			cat "$POSTGRES_DATA"/log/* >&2 || true
		fi
		exit 1
	fi
	"$POSTGRES_BINDIR/createdb" -h 127.0.0.1 -p "$POSTGRES_PORT" -U postgres "$TEST_POSTGRES_DATABASE"
	export ACCOUNTABLE_TEST_POSTGRES_DSN="postgres://postgres@127.0.0.1:${POSTGRES_PORT}/${TEST_POSTGRES_DATABASE}?sslmode=disable"
else
	echo "real Postgres proof requires Docker or local PostgreSQL binaries" >&2
	exit 1
fi

export ACCOUNTABLE_REQUIRE_POSTGRES=1
export ACCOUNTABLE_TEST_POSTGRES_HOST="127.0.0.1"
export ACCOUNTABLE_TEST_POSTGRES_PORT="$POSTGRES_PORT"
export ACCOUNTABLE_TEST_POSTGRES_DATABASE="$TEST_POSTGRES_DATABASE"
export ACCOUNTABLE_TEST_POSTGRES_USER="postgres"
export ACCOUNTABLE_TEST_POSTGRES_PASSWORD="$TEST_POSTGRES_PASSWORD"
"$@"
