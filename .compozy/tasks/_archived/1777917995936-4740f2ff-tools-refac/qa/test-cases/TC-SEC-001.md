# TC-SEC-001: Raw `claim_token` Redaction Across Every AGH-Owned Surface

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that no AGH-owned surface — tool, CLI, HTTP, UDS, hosted MCP, SSE, daemon log, observe events, memory entries, web fixtures, generated OpenAPI/TS types, or site docs — accepts or emits raw `claim_token` after the autonomy hard cut. Confirm `claim_token_hash` survives only as observability metadata.

## Traceability

- Task: task_09 (and the docs/codegen alignment in task_11).
- TechSpec: "Safety Invariants", "Post-Implementation Residual Checks", "Monitoring and Observability".
- ADR: ADR-005.
- Surfaces: `internal/api/contract/agents.go`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go`, `internal/api/spec/spec.go`, `internal/network/*`, `internal/observe/*`, generated artifacts.

## Preconditions

- Isolated `AGH_HOME`.
- One workspace + two sessions in the same workspace.
- A queued task fixture eligible for claim (recorded via `agh task create`).

## Test Steps

Follow the canonical sweep procedure in `tools-refac-redaction-suite.md`. Capture each channel's output under `qa/logs/TC-SEC-001/` and run the consolidated grep at the end.

1. **Drive an autonomy round-trip via tools:**
   ```bash
   agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/TC-SEC-001/tool-claim.json
   agh tool invoke agh__task_run_heartbeat --input '{"run_id":"'$RUN_ID'"}' -o json \
     | tee qa/logs/TC-SEC-001/tool-heartbeat.json
   agh tool invoke agh__task_run_complete --input '{"run_id":"'$RUN_ID'","result":{"ok":true}}' -o json \
     | tee qa/logs/TC-SEC-001/tool-complete.json
   ```

2. **Drive an autonomy round-trip via CLI:**
   ```bash
   agh task next -o json | tee qa/logs/TC-SEC-001/cli-next.json
   agh task heartbeat $RUN_ID -o json | tee qa/logs/TC-SEC-001/cli-heartbeat.json
   agh task complete $RUN_ID --result '{"ok":true}' -o json | tee qa/logs/TC-SEC-001/cli-complete.json
   ```
   - **Expected:** No `--claim-token` flag exists. Output never contains a raw token. `claim_token_hash` may appear.

3. **Drive an autonomy round-trip via HTTP/UDS:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/agents/tasks/runs/next" -d '{}' | tee qa/logs/TC-SEC-001/uds-next.json
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/agents/tasks/runs/$RUN_ID/heartbeat" -d '{}' | tee qa/logs/TC-SEC-001/uds-heartbeat.json
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/agents/tasks/runs/$RUN_ID/complete" -d '{"result":{"ok":true}}' \
     | tee qa/logs/TC-SEC-001/uds-complete.json
   ```

4. **Drive an autonomy round-trip via hosted MCP:**
   - Bind hosted MCP for the session and call `tools/call` for `agh__task_run_claim_next`, `..._heartbeat`, `..._complete`.
   - Capture frames in `qa/logs/TC-SEC-001/mcp-frames.jsonl`.

5. **Capture parallel SSE / log / observe / memory output:**
   ```bash
   curl -N --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/sessions/$SID/events" \
     | tee qa/logs/TC-SEC-001/sse-events.txt &
   sleep 1
   # … drive a heartbeat or complete to push events …
   cat $AGH_HOME/logs/daemon.log | tee qa/logs/TC-SEC-001/daemon.log
   agh observe events -o json | tee qa/logs/TC-SEC-001/observe-events.json
   agh memory list -o json | tee qa/logs/TC-SEC-001/memory.json
   ```

6. **Inspect generated artifacts:**
   ```bash
   grep -nE "claim_token" openapi/agh.json | tee qa/logs/TC-SEC-001/openapi-grep.txt
   grep -nE "claim_token" web/src/generated/agh-openapi.d.ts | tee qa/logs/TC-SEC-001/openapi-d-ts-grep.txt
   grep -RIn "claim_token" web/src/systems/tasks | tee qa/logs/TC-SEC-001/web-tasks-grep.txt
   ```
   - **Expected:** Zero matches except `claim_token_hash`.

7. **Inspect site docs:**
   ```bash
   grep -RIn "claim_token" packages/site/content/runtime | tee qa/logs/TC-SEC-001/site-grep.txt
   ```
   - **Expected:** Zero matches except `claim_token_hash` references in observability docs.

8. **Run the consolidated cross-channel grep:**
   ```bash
   grep -RIn -E "claim_token[^_]" qa/logs/TC-SEC-001 | tee qa/logs/TC-SEC-001/cross-channel-grep.txt
   ```
   - **Expected:** Zero matches.

9. Run focused Go tests:
   ```bash
   go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/contract ./internal/api/spec \
     -run "TestAgentTask|TestAutonomy|TestRedaction" -count=1 | tee qa/logs/TC-SEC-001/api-tests.log
   go test ./internal/cli -run "TestTask" -count=1 | tee qa/logs/TC-SEC-001/cli-tests.log
   go test ./internal/tools/builtin -run "TestAutonomy" -count=1 | tee qa/logs/TC-SEC-001/builtin-autonomy-tests.log
   go test ./internal/observe ./internal/memory -count=1 | tee qa/logs/TC-SEC-001/observe-memory-tests.log
   ```

## Evidence To Capture

- All channel logs listed above.
- Cross-channel grep result.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Hosted MCP `tools/call` argument that includes a raw token | `{"claim_token":"...", "run_id":"..."}` | Bridge ignores extra fields; lookup uses session-bound resolver; response never echoes the token |
| CLI invocation with `--claim-token` flag | `agh task heartbeat $RUN_ID --claim-token X` | Flag does not exist; argparse rejects it |
| Observe event payload containing tool output that included `claim_token` mistakenly | injected via test seam | Redactor strips it before persist; observe output only shows `claim_token_hash` |
| Memory entry with raw token text | injected via test seam | Memory write rejects or redacts before persist |

## Channels Exercised

- Tool invoke, CLI, HTTP, UDS, hosted MCP, SSE, daemon log, observe events, memory entries, generated OpenAPI/TS, site docs.

## Related Test Cases

- TC-AUT-001 (autonomy flow via tools).
- TC-AUT-006 (writer convergence).
- TC-SEC-003 (network send raw-token rejection).
- TC-REG-001 (codegen artifact grep).
- TC-REG-004 (web tasks system grep).
