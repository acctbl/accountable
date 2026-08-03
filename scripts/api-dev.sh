#!/usr/bin/env bash
set -euo pipefail

: "${ACCOUNTABLE_TEST_POSTGRES_DSN:?run through scripts/with-test-postgres.sh}"

DEV_DIR="$(mktemp -d "${TMPDIR:-/tmp}/accountable-api-dev.XXXXXX")"
SECRETS_DIR="$DEV_DIR/secrets"
STORAGE_DIR="$DEV_DIR/storage"
API_CONFIG="$DEV_DIR/api.toml"
MIGRATE_CONFIG="$DEV_DIR/migrate.toml"
mkdir -p "$SECRETS_DIR" "$STORAGE_DIR"

cleanup() {
	rm -f "$SECRETS_DIR/database.api_dsn" "$SECRETS_DIR/crypto.primary_key" "$API_CONFIG" "$MIGRATE_CONFIG"
	rmdir "$SECRETS_DIR" "$STORAGE_DIR" "$DEV_DIR" 2>/dev/null || true
}
trap cleanup EXIT

printf '%s' "$ACCOUNTABLE_TEST_POSTGRES_DSN" >"$SECRETS_DIR/database.api_dsn"
openssl rand -base64 32 >"$SECRETS_DIR/crypto.primary_key"
chmod 0600 "$SECRETS_DIR/database.api_dsn" "$SECRETS_DIR/crypto.primary_key"

cat >"$API_CONFIG" <<EOF
environment = "development"
listen_address = "127.0.0.1:8080"
architecture_probe = true
allowed_origins = ["http://localhost:3000", "http://127.0.0.1:3000"]
trusted_proxy_cidrs = []
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[database]
dsn_ref = "database.api_dsn"
connect_timeout = "5s"
health_check_interval = "2s"
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
health_check_interval = "2s"
max_connections = 16
timezone = "UTC"

[storage]
provider = "filesystem"
root = "$STORAGE_DIR"

[crypto]
provider = "local"
key_ref = "crypto.primary_key"
EOF

go run ./cmd/migrate --config "$MIGRATE_CONFIG"
go run ./cmd/api --config "$API_CONFIG"
