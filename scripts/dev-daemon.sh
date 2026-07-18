#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

binary=${1:-./.tmp/air/agh}
if [[ ! -x "$binary" ]]; then
  echo "dev: daemon binary is not executable: $binary" >&2
  exit 1
fi

stop_output=
if stop_output=$("$binary" daemon stop 2>&1); then
  echo "dev: stopped the existing daemon for development takeover"
elif [[ "$stop_output" != *"cli: daemon is not running"* ]]; then
  echo "dev: could not take over the existing daemon" >&2
  printf '%s\n' "$stop_output" >&2
  exit 1
fi

echo "dev: starting the daemon in the foreground"
exec "$binary" daemon start --foreground
