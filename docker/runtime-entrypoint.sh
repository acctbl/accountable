#!/bin/sh
set -eu

mode="${1:-}"
case "$mode" in
	api|migrate|preflight)
		: "${ACCOUNTABLE_CONFIG_BASE64:?ACCOUNTABLE_CONFIG_BASE64 is required}"
		umask 077
		printf '%s' "$ACCOUNTABLE_CONFIG_BASE64" | base64 -d > /run/accountable/config.toml
		unset ACCOUNTABLE_CONFIG_BASE64
		exec "/usr/local/bin/accountable-$mode" --config /run/accountable/config.toml
		;;
	bootstrap)
		: "${ACCOUNTABLE_CONFIG_BASE64:?ACCOUNTABLE_CONFIG_BASE64 is required}"
		: "${ACCOUNTABLE_DATABASE_MASTER_PASSWORD:?ACCOUNTABLE_DATABASE_MASTER_PASSWORD is required}"
		umask 077
		mkdir -p /run/accountable/secrets
		printf '%s' "$ACCOUNTABLE_CONFIG_BASE64" | base64 -d > /run/accountable/config.toml
		printf '%s' "$ACCOUNTABLE_DATABASE_MASTER_PASSWORD" > /run/accountable/secrets/database-master-password
		unset ACCOUNTABLE_CONFIG_BASE64 ACCOUNTABLE_DATABASE_MASTER_PASSWORD
		exec /usr/local/bin/accountable-bootstrap --config /run/accountable/config.toml
		;;
	*)
		echo "runtime mode must be api, bootstrap, migrate, or preflight" >&2
		exit 2
		;;
esac
