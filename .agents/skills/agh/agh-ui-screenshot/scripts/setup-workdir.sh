#!/usr/bin/env bash
# Bootstrap helper — creates an isolated work dir and installs chrome-launcher + chrome-remote-interface via bun.
# Required: bun, Google Chrome installed at the default macOS / Linux location.
# Usage:
#   bash setup-workdir.sh [<workdir>]
# If <workdir> is omitted, defaults to /tmp/agh-ui-screenshot.
# Output:
#   stdout: the absolute path of the prepared workdir.
#   stderr: bun install errors.
#   exit 0 on success.

set -euo pipefail

WORKDIR="${1:-/tmp/agh-ui-screenshot}"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ ! -f package.json ]; then
  cat > package.json <<'EOF'
{
  "name": "agh-ui-screenshot-workdir",
  "private": true,
  "type": "module"
}
EOF
fi

if [ ! -d node_modules/chrome-launcher ] || [ ! -d node_modules/chrome-remote-interface ]; then
  bun add chrome-launcher chrome-remote-interface >&2
fi

echo "$WORKDIR"
