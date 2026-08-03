#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
: "${ACCOUNTABLE_TEST_POSTGRES_DSN:?run through scripts/with-test-postgres.sh}"

STACK_TLS_DIR="$(mktemp -d)"
openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 1 \
	-subj "/CN=localhost" \
	-addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
	-keyout "$STACK_TLS_DIR/key.pem" \
	-out "$STACK_TLS_DIR/cert.pem" >/dev/null 2>&1

pnpm --filter @accountable/web build

API_CONFIG="$STACK_TLS_DIR/api.toml"
MIGRATE_CONFIG="$STACK_TLS_DIR/migrate.toml"
WEB_CONFIG="$STACK_TLS_DIR/web.toml"
SECRETS_DIR="$STACK_TLS_DIR/secrets"
STORAGE_DIR="$STACK_TLS_DIR/storage"
mkdir -p "$SECRETS_DIR" "$STORAGE_DIR"
printf '%s' "$ACCOUNTABLE_TEST_POSTGRES_DSN" >"$SECRETS_DIR/database.api_dsn"
openssl rand -base64 32 >"$SECRETS_DIR/crypto.primary_key"
chmod 0600 "$SECRETS_DIR/database.api_dsn" "$SECRETS_DIR/crypto.primary_key"
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

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[database]
dsn_ref = "database.api_dsn"
connect_timeout = "5s"
health_check_interval = "250ms"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "$STORAGE_DIR"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"
EOF
cat >"$MIGRATE_CONFIG" <<EOF
environment = "development"

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[database]
dsn_ref = "database.api_dsn"
connect_timeout = "5s"
health_check_interval = "250ms"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "$STORAGE_DIR"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"
EOF
cat >"$WEB_CONFIG" <<EOF
environment = "development"
listen_address = "127.0.0.1:4173"
dist_dir = "$ROOT/apps/web/dist"
api_base_url = "https://127.0.0.1:18080"
architecture_probe = true
configuration_revision = "playwright"
tls_certificate_file = "$STACK_TLS_DIR/cert.pem"
tls_private_key_file = "$STACK_TLS_DIR/key.pem"
EOF

mise exec -- go run ./cmd/migrate --config "$MIGRATE_CONFIG"
mise exec -- go run ./cmd/api --config "$API_CONFIG" &
API_PID=$!
mise exec -- go run ./cmd/web --config "$WEB_CONFIG" &
WEB_PID=$!

cleanup() {
	kill "$WEB_PID" 2>/dev/null || true
	kill "$API_PID" 2>/dev/null || true
	wait "$WEB_PID" 2>/dev/null || true
	wait "$API_PID" 2>/dev/null || true
	rm -f "$API_CONFIG" "$MIGRATE_CONFIG" "$WEB_CONFIG" "$STACK_TLS_DIR/cert.pem" "$STACK_TLS_DIR/key.pem"
	rm -f "$SECRETS_DIR/database.api_dsn" "$SECRETS_DIR/crypto.primary_key"
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
