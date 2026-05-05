#!/usr/bin/env zsh
set -euo pipefail

RUN_DIR=".compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution"
QA_ROOT=".compozy/tasks/network-threads/qa"
COMMAND_LOG="$RUN_DIR/cli-api-command-log.txt"
AGH_BIN="${AGH_BIN:-./bin/agh}"
FIXTURE="internal/testutil/acpmock/testdata/network_collaboration_fixture.json"
DRIVER="$RUN_DIR/acpmock-driver"
THREAD_ID="${THREAD_ID:-thread_builders_qa2}"
WORK_ID="${WORK_ID:-work_patch_qa2}"
TRACE_ID="${TRACE_ID:-trace_ops_patch_qa2}"
MSG_SAY_ID="${MSG_SAY_ID:-msg_say_qa2}"
MSG_DIRECT_ID="${MSG_DIRECT_ID:-msg_direct_qa2}"
MSG_RECEIPT_ID="${MSG_RECEIPT_ID:-msg_receipt_qa2}"
MSG_TRACE_ID="${MSG_TRACE_ID:-msg_trace_qa2}"
MSG_SUMMARY_ID="${MSG_SUMMARY_ID:-msg_summary_qa2}"
MSG_INVALID_CROSS_WORK_ID="${MSG_INVALID_CROSS_WORK_ID:-msg_invalid_cross_work_qa2}"

source "$QA_ROOT/bootstrap.env"

: > "$COMMAND_LOG"

log() {
  print -r -- "$*" | tee -a "$COMMAND_LOG"
}

run_capture() {
  local name="$1"
  shift
  log "+ $*"
  if "$@" > "$RUN_DIR/$name.stdout" 2> "$RUN_DIR/$name.stderr"; then
    cat "$RUN_DIR/$name.stdout" >> "$COMMAND_LOG"
    return 0
  else
    local exit_code=$?
    cat "$RUN_DIR/$name.stderr" >> "$COMMAND_LOG"
    log "COMMAND FAILED ($exit_code): $name"
    exit "$exit_code"
  fi
}

ensure_daemon_started() {
  log "+ $AGH_BIN daemon start -o json"
  if "$AGH_BIN" daemon start -o json > "$RUN_DIR/daemon-start.stdout" 2> "$RUN_DIR/daemon-start.stderr"; then
    cat "$RUN_DIR/daemon-start.stdout" >> "$COMMAND_LOG"
    return 0
  fi
  local exit_code=$?
  if grep -q "daemon already running" "$RUN_DIR/daemon-start.stderr"; then
    log "daemon already running; recording current daemon status"
    "$AGH_BIN" daemon status -o json > "$RUN_DIR/daemon-start.stdout" 2>> "$RUN_DIR/daemon-start.stderr"
    cat "$RUN_DIR/daemon-start.stdout" >> "$COMMAND_LOG"
    return 0
  fi
  cat "$RUN_DIR/daemon-start.stderr" >> "$COMMAND_LOG"
  log "COMMAND FAILED ($exit_code): daemon-start"
  exit "$exit_code"
}

expect_fail() {
  local name="$1"
  shift
  log "+ expect-fail $*"
  set +e
  "$@" > "$RUN_DIR/$name.stdout" 2> "$RUN_DIR/$name.stderr"
  local exit_code=$?
  set -e
  print -r -- "$exit_code" > "$RUN_DIR/$name.exit"
  if [[ "$exit_code" -eq 0 ]]; then
    log "UNEXPECTED SUCCESS: $name"
    exit 1
  fi
  cat "$RUN_DIR/$name.stderr" >> "$COMMAND_LOG"
  return 0
}

json_value() {
  local file="$1"
  local expr="$2"
  python3 - "$file" "$expr" <<'PY'
import json
import sys

path, expr = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)
for part in expr.split("."):
    if part:
        data = data[part]
print(data)
PY
}

http_get() {
  local name="$1"
  local route_path="$2"
  log "+ curl GET $AGH_WEB_API_PROXY_TARGET$route_path"
  curl -sS -f "$AGH_WEB_API_PROXY_TARGET$route_path" \
    > "$RUN_DIR/$name.stdout" \
    2> "$RUN_DIR/$name.stderr"
  cat "$RUN_DIR/$name.stdout" >> "$COMMAND_LOG"
}

http_post_code() {
  local name="$1"
  local route_path="$2"
  local payload="$3"
  log "+ curl POST $AGH_WEB_API_PROXY_TARGET$route_path"
  curl -sS \
    -X POST \
    -H "content-type: application/json" \
    -d "$payload" \
    -o "$RUN_DIR/$name.stdout" \
    -w "%{http_code}" \
    "$AGH_WEB_API_PROXY_TARGET$route_path" \
    > "$RUN_DIR/$name.status" \
    2> "$RUN_DIR/$name.stderr"
  cat "$RUN_DIR/$name.stdout" >> "$COMMAND_LOG"
}

wait_for_http() {
  local route_path="$1"
  local target="$2"
  local attempt
  for attempt in {1..80}; do
    if curl -sS -f "$AGH_WEB_API_PROXY_TARGET$route_path" > "$target" 2> "$target.stderr"; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

log "SCENARIO=$SCENARIO_SLUG"
log "AGH_HOME=$AGH_HOME"
log "AGH_WEB_API_PROXY_TARGET=$AGH_WEB_API_PROXY_TARGET"
log "WORKSPACE_PATH=$WORKSPACE_PATH"
log "THREAD_ID=$THREAD_ID"
log "WORK_ID=$WORK_ID"

run_capture build-acpmock-driver go build -o "$DRIVER" ./internal/testutil/acpmock/cmd/acpmock-driver

python3 - "$AGH_HOME" "$DRIVER" "$FIXTURE" "$RUN_DIR" <<'PY'
import json
import pathlib
import shlex
import sys

agh_home = pathlib.Path(sys.argv[1])
driver = pathlib.Path(sys.argv[2]).resolve()
fixture = pathlib.Path(sys.argv[3]).resolve()
run_dir = pathlib.Path(sys.argv[4]).resolve()
agents_dir = agh_home / "agents"
agents_dir.mkdir(parents=True, exist_ok=True)

with fixture.open("r", encoding="utf-8") as handle:
    data = json.load(handle)
agents = {agent["name"]: agent for agent in data["agents"]}

def yaml_single(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"

for name in ["ops-coordinator", "patch-worker"]:
    agent = agents[name]
    diagnostics = run_dir / f"acpmock-{name}.jsonl"
    command = shlex.join([
        str(driver),
        "--fixture",
        str(fixture),
        "--agent",
        name,
        "--diagnostics",
        str(diagnostics),
    ])
    content = "\n".join([
        "---",
        f"name: {name}",
        f"provider: {agent['provider']}",
        f"command: {yaml_single(command)}",
        f"permissions: {agent['permissions']}",
        "---",
        "",
        agent["prompt"],
        "",
    ])
    target = agents_dir / name / "AGENT.md"
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")
PY

run_capture provider-auth-claude "$AGH_BIN" provider auth status claude --no-probe -o json
run_capture provider-auth-codex "$AGH_BIN" provider auth status codex --no-probe -o json

ensure_daemon_started
wait_for_http "/api/daemon/status" "$RUN_DIR/http-daemon-status.json"
run_capture daemon-status "$AGH_BIN" daemon status -o json
http_get api-network-status "/api/network/status"

run_capture session-ops "$AGH_BIN" session new --cwd "$WORKSPACE_PATH" --agent ops-coordinator --channel builders -o json
run_capture session-patch "$AGH_BIN" session new --cwd "$WORKSPACE_PATH" --agent patch-worker --channel builders -o json

OPS_SESSION_ID="$(json_value "$RUN_DIR/session-ops.stdout" "id")"
PATCH_SESSION_ID="$(json_value "$RUN_DIR/session-patch.stdout" "id")"
OPS_PEER_ID="ops-coordinator.$OPS_SESSION_ID"
PATCH_PEER_ID="patch-worker.$PATCH_SESSION_ID"
print -r -- "OPS_SESSION_ID=$OPS_SESSION_ID" > "$RUN_DIR/scenario.env"
print -r -- "PATCH_SESSION_ID=$PATCH_SESSION_ID" >> "$RUN_DIR/scenario.env"
print -r -- "OPS_PEER_ID=$OPS_PEER_ID" >> "$RUN_DIR/scenario.env"
print -r -- "PATCH_PEER_ID=$PATCH_PEER_ID" >> "$RUN_DIR/scenario.env"
print -r -- "THREAD_ID=$THREAD_ID" >> "$RUN_DIR/scenario.env"
print -r -- "WORK_ID=$WORK_ID" >> "$RUN_DIR/scenario.env"
print -r -- "TRACE_ID=$TRACE_ID" >> "$RUN_DIR/scenario.env"

run_capture network-peers "$AGH_BIN" network peers builders -o json

run_capture thread-send-msg-say-01 "$AGH_BIN" network send \
  --session "$OPS_SESSION_ID" \
  --channel builders \
  --surface thread \
  --thread "$THREAD_ID" \
  --kind say \
  --id "$MSG_SAY_ID" \
  --trace-id "$TRACE_ID" \
  --body '{"text":"Review the launch checklist.","intent":"review-request"}' \
  -o json

run_capture threads-list "$AGH_BIN" network threads list --channel builders -o json
run_capture thread-show "$AGH_BIN" network threads show --channel builders --thread "$THREAD_ID" -o json
run_capture thread-messages-initial "$AGH_BIN" network threads messages --channel builders --thread "$THREAD_ID" -o json
http_get api-thread-messages-initial "/api/network/channels/builders/threads/$THREAD_ID/messages"

run_capture direct-resolve "$AGH_BIN" network directs resolve \
  --session "$PATCH_SESSION_ID" \
  --channel builders \
  --peer "$OPS_PEER_ID" \
  -o json
DIRECT_ID="$(json_value "$RUN_DIR/direct-resolve.stdout" "direct.direct_id")"
print -r -- "DIRECT_ID=$DIRECT_ID" >> "$RUN_DIR/scenario.env"

run_capture direct-send-msg-direct-01 "$AGH_BIN" network send \
  --session "$PATCH_SESSION_ID" \
  --channel builders \
  --surface direct \
  --direct "$DIRECT_ID" \
  --kind say \
  --to "$OPS_PEER_ID" \
  --work "$WORK_ID" \
  --id "$MSG_DIRECT_ID" \
  --reply-to "$MSG_SAY_ID" \
  --trace-id "$TRACE_ID" \
  --causation-id "$MSG_SAY_ID" \
  --body '{"text":"Review the migration details in the direct room.","intent":"handoff"}' \
  -o json

run_capture direct-send-msg-receipt-01 "$AGH_BIN" network send \
  --session "$OPS_SESSION_ID" \
  --channel builders \
  --surface direct \
  --direct "$DIRECT_ID" \
  --kind receipt \
  --to "$PATCH_PEER_ID" \
  --work "$WORK_ID" \
  --id "$MSG_RECEIPT_ID" \
  --reply-to "$MSG_DIRECT_ID" \
  --trace-id "$TRACE_ID" \
  --causation-id "$MSG_DIRECT_ID" \
  --body "{\"for_id\":\"$MSG_DIRECT_ID\",\"status\":\"accepted\"}" \
  -o json

run_capture direct-send-msg-trace-02 "$AGH_BIN" network send \
  --session "$PATCH_SESSION_ID" \
  --channel builders \
  --surface direct \
  --direct "$DIRECT_ID" \
  --kind trace \
  --to "$OPS_PEER_ID" \
  --work "$WORK_ID" \
  --id "$MSG_TRACE_ID" \
  --reply-to "$MSG_RECEIPT_ID" \
  --trace-id "$TRACE_ID" \
  --causation-id "$MSG_RECEIPT_ID" \
  --body '{"message":"Trace update recorded.","state":"working"}' \
  -o json

run_capture directs-list "$AGH_BIN" network directs list --channel builders --peer "$OPS_PEER_ID" -o json
run_capture direct-show "$AGH_BIN" network directs show --channel builders --direct "$DIRECT_ID" -o json
run_capture direct-messages "$AGH_BIN" network directs messages --channel builders --direct "$DIRECT_ID" -o json
run_capture direct-work-lookup "$AGH_BIN" network work lookup --work "$WORK_ID" -o json
http_get api-direct-messages "/api/network/channels/builders/directs/$DIRECT_ID/messages"

run_capture thread-send-msg-summary-01 "$AGH_BIN" network send \
  --session "$PATCH_SESSION_ID" \
  --channel builders \
  --surface thread \
  --thread "$THREAD_ID" \
  --kind say \
  --id "$MSG_SUMMARY_ID" \
  --reply-to "$MSG_TRACE_ID" \
  --trace-id "$TRACE_ID" \
  --causation-id "$MSG_TRACE_ID" \
  --body '{"text":"Summary: migration review passed with one cleanup follow-up.","intent":"summarize-back"}' \
  -o json

run_capture thread-messages-final "$AGH_BIN" network threads messages --channel builders --thread "$THREAD_ID" -o json
http_get api-thread-messages-final "/api/network/channels/builders/threads/$THREAD_ID/messages"

expect_fail invalid-direct-send "$AGH_BIN" network send \
  --session "$OPS_SESSION_ID" \
  --channel builders \
  --surface direct \
  --direct direct_invalid \
  --kind say \
  --to "$PATCH_PEER_ID" \
  --body '{"text":"invalid direct room"}' \
  -o json

set +e
"$AGH_BIN" network directs resolve --session "$OPS_SESSION_ID" --channel builders --peer "$PATCH_PEER_ID" -o json > "$RUN_DIR/direct-race-1.stdout" 2> "$RUN_DIR/direct-race-1.stderr" &
race_one=$!
"$AGH_BIN" network directs resolve --session "$PATCH_SESSION_ID" --channel builders --peer "$OPS_PEER_ID" -o json > "$RUN_DIR/direct-race-2.stdout" 2> "$RUN_DIR/direct-race-2.stderr" &
race_two=$!
wait "$race_one"
race_one_status=$?
wait "$race_two"
race_two_status=$?
set -e
print -r -- "$race_one_status" > "$RUN_DIR/direct-race-1.exit"
print -r -- "$race_two_status" > "$RUN_DIR/direct-race-2.exit"
if [[ "$race_one_status" -ne 0 || "$race_two_status" -ne 0 ]]; then
  log "direct race command failed"
  exit 1
fi
python3 - "$RUN_DIR/direct-race-1.stdout" "$RUN_DIR/direct-race-2.stdout" "$DIRECT_ID" <<'PY'
import json
import sys

left = json.load(open(sys.argv[1], "r", encoding="utf-8"))["direct"]["direct_id"]
right = json.load(open(sys.argv[2], "r", encoding="utf-8"))["direct"]["direct_id"]
want = sys.argv[3]
if left != right or left != want:
    raise SystemExit(f"direct race mismatch: left={left} right={right} want={want}")
PY

http_post_code legacy-interaction-id "/api/network/send" '{"session_id":"'"$OPS_SESSION_ID"'","channel":"builders","kind":"say","interaction_id":"legacy","body":{"text":"legacy"}}'
legacy_status="$(cat "$RUN_DIR/legacy-interaction-id.status")"
if [[ "$legacy_status" != "400" ]]; then
  log "legacy interaction_id status=$legacy_status, want 400"
  exit 1
fi

expect_fail legacy-kind-direct "$AGH_BIN" network send \
  --session "$OPS_SESSION_ID" \
  --channel builders \
  --surface direct \
  --direct "$DIRECT_ID" \
  --kind direct \
  --to "$PATCH_PEER_ID" \
  --body '{"text":"legacy kind"}' \
  -o json

expect_fail stale-flag-interaction-id "$AGH_BIN" network send \
  --session "$OPS_SESSION_ID" \
  --channel builders \
  --kind say \
  --interaction-id old \
  --body '{"text":"stale flag"}' \
  -o json

expect_fail invalid-cross-container-work "$AGH_BIN" network send \
  --session "$PATCH_SESSION_ID" \
  --channel builders \
  --surface thread \
  --thread "$THREAD_ID" \
  --kind say \
  --work "$WORK_ID" \
  --id "$MSG_INVALID_CROSS_WORK_ID" \
  --body '{"text":"invalid work reuse"}' \
  -o json

run_capture network-status-final "$AGH_BIN" network status -o json
http_get api-network-status-final "/api/network/status"
run_capture session-list-final "$AGH_BIN" session list --all -o json

log "CLI/API scenario completed"
