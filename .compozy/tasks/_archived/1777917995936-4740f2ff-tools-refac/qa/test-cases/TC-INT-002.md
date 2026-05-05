# TC-INT-002: Tool / CLI / HTTP / UDS / Hosted MCP Parity For Built-in Catalog

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `list`, `search`, `get`, and `invoke` agree across tool, CLI, HTTP, UDS, and hosted MCP for the same caller scope. The set of callable tools, the deny/unavailable reasons, and the call results must be identical.

## Traceability

- Tasks: 01, 03, 04, 05, 06, 07, 08, 09, 10.
- TechSpec: "API Endpoints", "Agent Manageability Plan", "E2E parity checks".
- ADRs: ADR-001, ADR-002.
- Surfaces: `internal/api/core/tools.go`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli/tool.go`, `internal/daemon/hosted_mcp.go`, `internal/mcp/hosted.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A session bound to a default agent with the canonical built-in catalog enabled.
- Hosted MCP bind achievable via `agh tool mcp --session $SID --bind-nonce $NONCE`.

## Test Steps

1. **List parity:**
   ```bash
   agh tool list -o json | jq -S '[.[].id]' > qa/logs/TC-INT-002/cli-tool-list-ids.json
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/tools" \
     | jq -S '[.[].id]' > qa/logs/TC-INT-002/uds-tool-list-ids.json
   curl -s "http://127.0.0.1:$AGH_HTTP_PORT/api/tools" \
     | jq -S '[.[].id]' > qa/logs/TC-INT-002/http-tool-list-ids.json
   diff qa/logs/TC-INT-002/cli-tool-list-ids.json qa/logs/TC-INT-002/uds-tool-list-ids.json
   diff qa/logs/TC-INT-002/cli-tool-list-ids.json qa/logs/TC-INT-002/http-tool-list-ids.json
   ```
   - **Expected:** All three files identical (operator scope).

2. **Session-scope list parity:**
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/sessions/$SID/tools" \
     | jq -S '[.[].id]' > qa/logs/TC-INT-002/uds-session-tools.json
   ```
   Bind hosted MCP and capture `tools/list`:
   ```bash
   # The bind harness should record JSON-RPC frames; extract tool IDs into:
   jq -S '[.result.tools[].name]' qa/logs/TC-INT-002/hosted-mcp-tools-list.json \
     > qa/logs/TC-INT-002/hosted-mcp-tools-ids.json
   diff qa/logs/TC-INT-002/uds-session-tools.json qa/logs/TC-INT-002/hosted-mcp-tools-ids.json
   ```
   - **Expected:** Set + ordering identical between session HTTP/UDS view and hosted MCP `tools/list` (TC-INT-003 covers this in depth, here we keep parity as a smoke check).

3. **Search parity:**
   ```bash
   for surface in cli uds http; do …; done
   # Against query "memory" and verify the same set of IDs is returned.
   ```
   - **Expected:** Search results agree across surfaces.

4. **Get parity:**
   For one well-known tool ID per family (`agh__memory_list`, `agh__sessions_list`, `agh__workspace_list`, `agh__config_show`, `agh__hooks_list`, `agh__automation_jobs_list`, `agh__extensions_list`, `agh__mcp_auth_status`, `agh__observe_events`, `agh__bridges_list`, `agh__network_status`, `agh__task_run_list`):
   ```bash
   agh tool info <id> -o json > qa/logs/TC-INT-002/cli-info-$id.json
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/tools/$id" \
     > qa/logs/TC-INT-002/uds-info-$id.json
   ```
   - **Expected:** Same descriptor (input/output schema, reason codes, source ref).

5. **Invoke parity (read-only):**
   For the same well-known IDs, invoke the tool with empty/minimal input via tool, CLI, UDS, HTTP, and hosted MCP `tools/call`. Capture all responses under `qa/logs/TC-INT-002/invoke-<id>-<surface>.json`.
   - **Expected:** Identical successful payloads (or identical deterministic errors). For tools that depend on session lineage, the tool/UDS/HTTP-with-session-header variants align with hosted MCP under the same session.

6. **Mutate parity (mutable families):**
   For one allowed mutation per family (config, hooks, automation, extension), exercise tool, CLI, UDS in sequence (not in parallel — see CLAUDE.md sequential write rule) and confirm equivalent result.

7. Run focused Go tests:
   ```bash
   go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1 \
     | tee qa/logs/TC-INT-002/api-tests.log
   go test ./internal/cli -run "TestTool|TestToolsets" -count=1 \
     | tee qa/logs/TC-INT-002/cli-tests.log
   go test ./internal/tools/builtin -count=1 | tee qa/logs/TC-INT-002/builtin-tests.log
   ```

## Evidence To Capture

- All ID lists per surface plus diffs.
- All info / invoke payloads per ID per surface.
- Hosted MCP JSON-RPC frame logs.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Surface not configured (e.g., HTTP disabled) | `[http].enabled=false` | The HTTP surface is skipped from the parity matrix; CLI/UDS/MCP must still agree |
| Hosted MCP not bound | session without nonce bind | Hosted MCP comparisons are skipped for that lab; CLI/HTTP/UDS still agree |
| Operator-only tool requested via session scope | `agh__mcp_auth_login` (intentionally not present) | All surfaces deny consistently |
| Search filters deny narrows | denied tool searched by name | Operator view returns with reason; session view empty |

## Channels Exercised

- Tool, CLI, HTTP, UDS, hosted MCP.

## Related Test Cases

- TC-FUNC-001 (default discovery overlay).
- TC-INT-003 (hosted MCP equality deep dive).
- TC-FUNC-003..008 (per-family parity drilldowns).
