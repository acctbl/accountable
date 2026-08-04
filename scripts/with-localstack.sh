#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
	echo "usage: with-localstack.sh <command> [args...]" >&2
	exit 2
fi
if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	echo "LocalStack contract tests require Docker" >&2
	exit 1
fi
: "${LOCALSTACK_TEST_IMAGE:?LOCALSTACK_TEST_IMAGE is required}"

LOCALSTACK_CONTAINER="accountable-localstack-test-$$"

cleanup() {
	docker rm --force "$LOCALSTACK_CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run --detach \
	--name "$LOCALSTACK_CONTAINER" \
	--env DISABLE_EVENTS=1 \
	--env SERVICES=s3,kms \
	--publish 127.0.0.1::4566 \
	"$LOCALSTACK_TEST_IMAGE" >/dev/null

LOCALSTACK_PORT=""
for _ in $(seq 1 50); do
	LOCALSTACK_BINDING="$(docker port "$LOCALSTACK_CONTAINER" 4566/tcp 2>/dev/null || true)"
	if [ -n "$LOCALSTACK_BINDING" ]; then
		LOCALSTACK_PORT="${LOCALSTACK_BINDING##*:}"
		break
	fi
	sleep 0.1
done
if [ -z "$LOCALSTACK_PORT" ]; then
	echo "LocalStack did not publish its gateway port" >&2
	docker logs "$LOCALSTACK_CONTAINER" >&2 || true
	exit 1
fi
LOCALSTACK_READY=false
for _ in $(seq 1 100); do
	if curl --fail --silent --max-time 0.5 "http://127.0.0.1:${LOCALSTACK_PORT}/_localstack/health" >/dev/null 2>&1; then
		LOCALSTACK_READY=true
		break
	fi
	sleep 0.1
done
if [ "$LOCALSTACK_READY" != true ]; then
	echo "LocalStack did not become ready" >&2
	docker logs "$LOCALSTACK_CONTAINER" >&2 || true
	exit 1
fi

export ACCOUNTABLE_TEST_AWS_ENDPOINT="http://127.0.0.1:${LOCALSTACK_PORT}"
export ACCOUNTABLE_REQUIRE_LOCALSTACK=1
export AWS_ACCESS_KEY_ID="accountable-test"
export AWS_SECRET_ACCESS_KEY="accountable-test"
export AWS_REGION="us-east-1"
set +e
"$@"
STATUS=$?
set -e
if [ "$STATUS" -ne 0 ]; then
	docker logs "$LOCALSTACK_CONTAINER" >&2 || true
fi
exit "$STATUS"
