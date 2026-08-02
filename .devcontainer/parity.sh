#!/usr/bin/env bash
set -euo pipefail

mise trust mise.toml
mise install --locked
exec mise exec -- task ci
