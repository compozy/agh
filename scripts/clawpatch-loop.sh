#!/usr/bin/env bash
#
# clawpatch-loop.sh — sequentially next -> fix -> revalidate until no findings remain.

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

while :; do
  id=$(clawpatch next --json 2>/dev/null | jq -r '(.id // (.[0].id) // empty)')
  [[ -z "$id" ]] && { echo "no open findings remaining"; break; }

  echo ">>> fix $id"
  clawpatch fix --finding "$id"

  echo ">>> revalidate $id"
  clawpatch revalidate --finding "$id"
done
