#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

: "${AIR_VERSION:?AIR_VERSION must be set by the development command}"

air_pid=
vite_pid=
dev_web_dir=
api_proxy_target=${AGH_WEB_API_PROXY_TARGET:-http://localhost:2123}
export AGH_WEB_API_PROXY_TARGET=$api_proxy_target

terminate_process() {
  local pid=$1
  local name=$2

  if [[ -z "$pid" ]] || ! kill -0 "$pid" 2>/dev/null; then
    return
  fi

  echo "dev: stopping $name"
  if ! kill -TERM "$pid" 2>/dev/null && kill -0 "$pid" 2>/dev/null; then
    echo "dev: failed to signal $name (pid=$pid)" >&2
  fi
}

wait_for_process() {
  local pid=$1
  local name=$2
  local attempts=0

  if [[ -z "$pid" ]]; then
    return
  fi

  while kill -0 "$pid" 2>/dev/null && ((attempts < 64)); do
    sleep 0.25
    attempts=$((attempts + 1))
  done

  if kill -0 "$pid" 2>/dev/null; then
    echo "dev: $name did not stop after 16s; forcing termination" >&2
    if ! kill -KILL "$pid" 2>/dev/null && kill -0 "$pid" 2>/dev/null; then
      echo "dev: failed to terminate $name (pid=$pid)" >&2
    fi
  fi

  local wait_status=0
  wait "$pid" || wait_status=$?
  if ((wait_status != 0 && wait_status != 130 && wait_status != 137 && wait_status != 143)); then
    echo "dev: $name exited with status $wait_status" >&2
  fi
}

remove_dev_web_redirect() {
  if [[ -z "$dev_web_dir" ]]; then
    return
  fi

  local redirect_file="$dev_web_dir/index.html"
  if [[ -e "$redirect_file" ]] && ! rm -f "$redirect_file"; then
    echo "dev: failed to remove $redirect_file" >&2
  fi
}

cleanup() {
  local exit_status=$?
  trap - EXIT INT TERM

  remove_dev_web_redirect
  terminate_process "$vite_pid" "Vite"
  terminate_process "$air_pid" "Air"
  wait_for_process "$vite_pid" "Vite"
  wait_for_process "$air_pid" "Air"

  exit "$exit_status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ -n ${AGH_WEB_PORT:-} ]]; then
  web_port=$(bun scripts/find-dev-port.mjs "$AGH_WEB_PORT" --strict)
else
  web_port=$(bun scripts/find-dev-port.mjs 3000)
fi
web_url="http://localhost:$web_port"

dev_web_dir="$repo_root/.tmp/dev-web-redirect"
mkdir -p "$dev_web_dir"
{
  printf '%s\n' '<!doctype html>' '<meta charset="utf-8">'
  printf '<meta http-equiv="refresh" content="0;url=%s/">\n' "$web_url"
  printf '%s\n' '<title>AGH development server</title>' '<script>'
  printf 'window.location.replace("%s" + window.location.pathname + window.location.search + window.location.hash);\n' "$web_url"
  printf '%s\n' '</script>'
  printf '<p>Open <a href="%s/">%s</a>.</p>\n' "$web_url" "$web_url"
} > "$dev_web_dir/index.html"
export AGH_WEB_DIST_DIR=$dev_web_dir

echo "dev: live web UI: $web_url"
echo "dev: daemon web routes will redirect to the live UI"
echo "dev: API traffic will be proxied to the daemon on $api_proxy_target"

bash scripts/run-air.sh "$AIR_VERSION" -c .air.toml &
air_pid=$!

bun run --cwd web dev:raw -- --port "$web_port" --strictPort &
vite_pid=$!

while :; do
  if ! kill -0 "$air_pid" 2>/dev/null; then
    air_status=0
    wait "$air_pid" || air_status=$?
    air_pid=
    echo "dev: Air exited with status $air_status" >&2
    exit "$air_status"
  fi
  if ! kill -0 "$vite_pid" 2>/dev/null; then
    vite_status=0
    wait "$vite_pid" || vite_status=$?
    vite_pid=
    echo "dev: Vite exited with status $vite_status" >&2
    exit "$vite_status"
  fi
  sleep 0.25
done
