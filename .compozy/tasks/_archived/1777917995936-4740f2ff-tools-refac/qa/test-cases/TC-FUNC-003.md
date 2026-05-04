# TC-FUNC-003: Read-Surface Coverage (Coordination, Session, Workspace, Memory, Observe, Bridges)

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the expanded read-only built-in families return the same data as their CLI/HTTP/UDS counterparts and preserve current scope, visibility, redaction, and not-found semantics. Confirm `agh__coordination` is extended (not renamed) and that descriptor registration stays in sync with native handlers for memory, observe, and bridges.

## Traceability

- Tasks: task_03 (Coordination/Session/Workspace), task_04 (Memory/Observe/Bridges).
- TechSpec: "Canonical Built-In Surface", "Implementation Steps".
- ADRs: ADR-001, ADR-002.
- Surfaces: `internal/tools/builtin/{network.go,sessions.go,workspace.go,memory.go,observe.go,bridges.go}`, `internal/tools/builtin/toolsets.go`, `internal/api/core/{network.go,session_workspace.go,memory.go,bridges.go}`, `internal/observe/{query.go,health.go}`.

## Preconditions

- Isolated `AGH_HOME` with at least: one workspace, two sessions in that workspace (one with parent lineage), one bridge configured, one memory entry written, one network channel.
- Deterministic seed data captured under `qa/logs/TC-FUNC-003/seed.txt`.

## Test Steps

1. **Coordination:**
   ```bash
   agh network status -o json | tee qa/logs/TC-FUNC-003/cli-network-status.json
   agh tool invoke agh__network_status -o json | tee qa/logs/TC-FUNC-003/tool-network-status.json
   diff <(jq -S . qa/logs/TC-FUNC-003/cli-network-status.json) \
        <(jq -S . qa/logs/TC-FUNC-003/tool-network-status.json)
   ```
   Repeat for `network_channels`, `network_inbox`, `network_peers`. `network_send` is exercised by TC-SEC-003.
   - **Expected:** Tool output matches CLI output for the same caller scope. The `agh__coordination` toolset includes the new read verbs alongside `network_peers` and `network_send`.

2. **Sessions:**
   ```bash
   agh session list -o json | tee qa/logs/TC-FUNC-003/cli-session-list.json
   agh tool invoke agh__session_list -o json | tee qa/logs/TC-FUNC-003/tool-session-list.json
   ```
   Compare tool vs CLI for `session_status`, `session_history`, `session_events`, `session_describe`.
   - **Expected:** Output matches; "not found" semantics yield the same deterministic error code; lineage filters apply identically.

3. **Workspace:**
   ```bash
   agh workspace list -o json | tee qa/logs/TC-FUNC-003/cli-workspace-list.json
   agh tool invoke agh__workspace_list -o json | tee qa/logs/TC-FUNC-003/tool-workspace-list.json
   ```
   Compare tool vs CLI for `workspace_info`, `workspace_describe`.
   - **Expected:** Output matches and respects existing visibility rules.

4. **Memory:**
   ```bash
   agh memory list -o json | tee qa/logs/TC-FUNC-003/cli-memory-list.json
   agh tool invoke agh__memory_list -o json | tee qa/logs/TC-FUNC-003/tool-memory-list.json
   ```
   Compare tool vs CLI for `memory_read`, `memory_search`. (Writes belong to the future write surface; this TC focuses on reads only.)
   - **Expected:** Same scope filtering; redaction matches CLI semantics. No raw secret material surfaces.

5. **Observe:**
   ```bash
   agh observe events -o json | tee qa/logs/TC-FUNC-003/cli-observe-events.json
   agh tool invoke agh__observe_events -o json | tee qa/logs/TC-FUNC-003/tool-observe-events.json
   agh observe metrics -o json | tee qa/logs/TC-FUNC-003/cli-observe-metrics.json
   agh tool invoke agh__observe_metrics -o json | tee qa/logs/TC-FUNC-003/tool-observe-metrics.json
   ```
   - **Expected:** Output matches; events expose `actor_kind`/`actor_id` but never `claim_token`; secret-bearing payloads remain redacted.

6. **Bridges:**
   ```bash
   agh bridge list -o json | tee qa/logs/TC-FUNC-003/cli-bridge-list.json
   agh tool invoke agh__bridges_list -o json | tee qa/logs/TC-FUNC-003/tool-bridge-list.json
   agh bridge status --name $BRIDGE -o json | tee qa/logs/TC-FUNC-003/cli-bridge-status.json
   agh tool invoke agh__bridges_status --input '{"name":"'$BRIDGE'"}' -o json | tee qa/logs/TC-FUNC-003/tool-bridge-status.json
   ```
   - **Expected:** Tool output omits provider-config and credential material identical to CLI behavior. Status only.

7. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestNetwork|TestSession|TestWorkspace|TestMemory|TestObserve|TestBridges" -count=1 | tee qa/logs/TC-FUNC-003/builtin-tests.log
   go test ./internal/api/core -count=1 | tee qa/logs/TC-FUNC-003/api-core-tests.log
   ```
   - **Expected:** All tests pass; descriptor + native handler tests stay in sync.

## Evidence To Capture

- All `qa/logs/TC-FUNC-003/cli-*.json` and `qa/logs/TC-FUNC-003/tool-*.json` files plus their diffs.
- Seed data record `qa/logs/TC-FUNC-003/seed.txt`.
- Test logs from Step 7.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Workspace with no sessions | Fresh workspace | `agh__session_list` returns empty array; CLI behaves identically |
| Memory entry visibility scoped to session lineage | Cross-lineage session | Tool returns filtered subset matching CLI |
| Observe events with secret-bearing tool result | Force a tool result with sensitive data | Tool/CLI output redacts identically |
| Bridge with FAILED health | Mock provider down | Status reports failure; no provider-config exposed |

## Channels Exercised

- CLI vs tool parity (HTTP/UDS implicit because tool invoke routes through the daemon native provider).
- Direct HTTP/UDS for `/api/network/*`, `/api/sessions/*`, `/api/workspaces/*`, `/api/memory/*`, `/api/bridges/*`, `/api/observe/*` (capture under `qa/logs/TC-FUNC-003/uds-*.txt` for any TC step where parity is questioned).

## Related Test Cases

- TC-INT-002 (transport parity for `list/search/info/invoke`).
- TC-INT-005 (denial-reason flow into operator projection).
- TC-SEC-001 (claim-token redaction in observe output).
