#!/usr/bin/env bash
set -euo pipefail

# The parity runner maps its caller's UID onto the bind-mounted workspace, so
# keep all mutable tool state under /tmp instead of the image user's home.
mkdir -p "$HOME" "$MISE_DATA_DIR" "$MISE_CONFIG_DIR"

mise trust mise.toml
mise install --locked
exec mise exec -- task ci
