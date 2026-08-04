#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
: "${ACCOUNTABLE_TEST_POSTGRES_HOST:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_PORT:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_DATABASE:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_USER:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_PASSWORD:?run through scripts/with-test-postgres.sh}"

STACK_TLS_DIR="$(mktemp -d)"
openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 1 \
	-subj "/CN=localhost" \
	-addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
	-keyout "$STACK_TLS_DIR/key.pem" \
	-out "$STACK_TLS_DIR/cert.pem" >/dev/null 2>&1

pnpm --filter @accountable/web build

API_CONFIG="$STACK_TLS_DIR/api.toml"
MIGRATE_CONFIG="$STACK_TLS_DIR/migrate.toml"
SECRETS_DIR="$STACK_TLS_DIR/secrets"
STORAGE_DIR="$STACK_TLS_DIR/storage"
mkdir -p "$SECRETS_DIR" "$STORAGE_DIR"
printf '%s' "$ACCOUNTABLE_TEST_POSTGRES_PASSWORD" >"$SECRETS_DIR/database.password"
openssl rand -base64 32 >"$SECRETS_DIR/crypto.primary_key"
chmod 0600 "$SECRETS_DIR/database.password" "$SECRETS_DIR/crypto.primary_key"
cat >"$API_CONFIG" <<EOF
environment = "development"
listen_address = "127.0.0.1:18080"
architecture_probe = true
allowed_origins = ["https://127.0.0.1:4173"]
trusted_proxy_cidrs = ["127.0.0.1/32", "::1/128"]
tls_certificate_file = "$STACK_TLS_DIR/cert.pem"
tls_private_key_file = "$STACK_TLS_DIR/key.pem"
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"
foundation_check_timeout = "30s"

[features]
provider = "noop"

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[database]
host = "$ACCOUNTABLE_TEST_POSTGRES_HOST"
port = $ACCOUNTABLE_TEST_POSTGRES_PORT
name = "$ACCOUNTABLE_TEST_POSTGRES_DATABASE"
user = "$ACCOUNTABLE_TEST_POSTGRES_USER"
role = "$ACCOUNTABLE_TEST_POSTGRES_USER"
password_ref = "database.password"
tls_mode = "disable"
connect_timeout = "5s"
statement_timeout = "10s"
health_check_interval = "250ms"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "$STORAGE_DIR"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"

[time]
provider = "system"
max_clock_error = "1s"
max_database_skew = "5s"
EOF
cat >"$MIGRATE_CONFIG" <<EOF
environment = "development"
foundation_check_timeout = "30s"

[features]
provider = "noop"

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[database]
host = "$ACCOUNTABLE_TEST_POSTGRES_HOST"
port = $ACCOUNTABLE_TEST_POSTGRES_PORT
name = "$ACCOUNTABLE_TEST_POSTGRES_DATABASE"
user = "$ACCOUNTABLE_TEST_POSTGRES_USER"
role = "$ACCOUNTABLE_TEST_POSTGRES_USER"
password_ref = "database.password"
tls_mode = "disable"
connect_timeout = "5s"
statement_timeout = "10s"
health_check_interval = "250ms"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "$STORAGE_DIR"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"

[time]
provider = "system"
max_clock_error = "1s"
max_database_skew = "5s"
EOF

mise exec -- go run ./cmd/migrate --config "$MIGRATE_CONFIG"
mise exec -- go run ./cmd/api --config "$API_CONFIG" &
API_PID=$!

export ACCOUNTABLE_RUNTIME_API_BASE_URL="https://127.0.0.1:18080"
export ACCOUNTABLE_RUNTIME_ARCHITECTURE_PROBE="true"
export ACCOUNTABLE_RUNTIME_CONFIGURATION_REVISION="playwright"
export ACCOUNTABLE_TLS_CERT_FILE="$STACK_TLS_DIR/cert.pem"
export ACCOUNTABLE_TLS_KEY_FILE="$STACK_TLS_DIR/key.pem"
export ACCOUNTABLE_WEB_HOST="127.0.0.1"
export ACCOUNTABLE_WEB_PORT="4173"
pnpm --filter @accountable/web exec vite preview --host 127.0.0.1 --port 4173 &
WEB_PID=$!

cleanup() {
	kill "$WEB_PID" 2>/dev/null || true
	kill "$API_PID" 2>/dev/null || true
	wait "$WEB_PID" 2>/dev/null || true
	wait "$API_PID" 2>/dev/null || true
	rm -f "$API_CONFIG" "$MIGRATE_CONFIG" "$STACK_TLS_DIR/cert.pem" "$STACK_TLS_DIR/key.pem"
	rm -f "$SECRETS_DIR/database.password" "$SECRETS_DIR/crypto.primary_key"
	rmdir "$SECRETS_DIR" "$STORAGE_DIR" 2>/dev/null || true
	rmdir "$STACK_TLS_DIR" 2>/dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 100); do
	if (echo >/dev/tcp/127.0.0.1/18080) >/dev/null 2>&1; then
		break
	fi
	sleep 0.1
done

wait "$WEB_PID"
