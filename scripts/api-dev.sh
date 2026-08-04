#!/usr/bin/env bash
set -euo pipefail

: "${ACCOUNTABLE_TEST_POSTGRES_HOST:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_PORT:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_DATABASE:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_USER:?run through scripts/with-test-postgres.sh}"
: "${ACCOUNTABLE_TEST_POSTGRES_PASSWORD:?run through scripts/with-test-postgres.sh}"

DEV_DIR="$(mktemp -d "${TMPDIR:-/tmp}/accountable-api-dev.XXXXXX")"
SECRETS_DIR="$DEV_DIR/secrets"
STORAGE_DIR="$DEV_DIR/storage"
API_CONFIG="$DEV_DIR/api.toml"
MIGRATE_CONFIG="$DEV_DIR/migrate.toml"
mkdir -p "$SECRETS_DIR" "$STORAGE_DIR"

cleanup() {
	rm -f "$SECRETS_DIR/database.password" "$SECRETS_DIR/crypto.primary_key" "$API_CONFIG" "$MIGRATE_CONFIG"
	rmdir "$SECRETS_DIR" "$STORAGE_DIR" "$DEV_DIR" 2>/dev/null || true
}
trap cleanup EXIT

printf '%s' "$ACCOUNTABLE_TEST_POSTGRES_PASSWORD" >"$SECRETS_DIR/database.password"
openssl rand -base64 32 >"$SECRETS_DIR/crypto.primary_key"
chmod 0600 "$SECRETS_DIR/database.password" "$SECRETS_DIR/crypto.primary_key"

write_foundation() {
	local role="$1"
	cat <<EOF
schema_version = 1
revision = "development"
environment = "development"
cell_id = "local"
aws_region = "eu-west-2"
runtime_role = "$role"
foundation_check_timeout = "30s"
readiness_probe_interval = "2s"

[capabilities]
architecture_probe = $([ "$role" = "api" ] && printf true || printf false)
postgres = true
secrets = true
kms = true
object_storage = true
telemetry = false
redpanda = false
EOF
}

{
	write_foundation api
	cat <<EOF

[server]
listen_address = "127.0.0.1:8080"
allowed_origins = ["http://localhost:3000", "http://127.0.0.1:3000"]
trusted_proxy_cidrs = []
unary_rpc_timeout = "10s"
stream_rpc_timeout = "25s"
EOF
} >"$API_CONFIG"

write_foundation migrate >"$MIGRATE_CONFIG"

for config in "$API_CONFIG" "$MIGRATE_CONFIG"; do
	cat >>"$config" <<EOF

[secrets]
provider = "file"
directory = "$SECRETS_DIR"

[postgres]
host = "$ACCOUNTABLE_TEST_POSTGRES_HOST"
port = $ACCOUNTABLE_TEST_POSTGRES_PORT
name = "$ACCOUNTABLE_TEST_POSTGRES_DATABASE"
user = "$ACCOUNTABLE_TEST_POSTGRES_USER"
role = "$ACCOUNTABLE_TEST_POSTGRES_USER"
password_ref = "database.password"
tls_mode = "disable"
connect_timeout = "5s"
statement_timeout = "10s"
health_check_interval = "2s"
max_connections = 16
timezone = "UTC"

[object_storage]
provider = "filesystem"
root = "$STORAGE_DIR"
access_purpose = "foundation-proof"

[kms]
provider = "local"
key_ref = "crypto.primary_key"
encryption_context_prefix = "accountable.foundation"

[time]
provider = "system"
max_clock_error = "1s"
max_database_skew = "5s"
EOF
done

go run ./cmd/migrate --config "$MIGRATE_CONFIG"
go run ./cmd/preflight --config "$API_CONFIG"
go run ./cmd/api --config "$API_CONFIG"
