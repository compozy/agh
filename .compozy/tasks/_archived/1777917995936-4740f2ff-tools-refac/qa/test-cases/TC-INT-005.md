# TC-INT-005: Hook Denial And Source-Health Denial Reason Codes

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that hook-denied and source-health-denied tools surface deterministic `reason_codes` in operator projection and that the projection layer's reasons match what dispatch will return when forced.

## Traceability

- Tasks: 01, 04, 06.
- TechSpec: "Test Strategy → Unit Tests", "Monitoring and Observability".
- ADR: ADR-002.
- Surfaces: `internal/tools/policy.go`, `internal/hooks/permission.go`, `internal/api/core/tools.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A hook fixture that denies `agh__memory_search` for the configured agent.
- A source (extension or MCP server) marked FAILED so an associated tool becomes unavailable.

## Test Steps

1. Capture operator projection and identify the affected entries:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/tools" \
     | tee qa/logs/TC-INT-005/operator.json
   jq '[.[] | select(.id=="agh__memory_search")][0]' qa/logs/TC-INT-005/operator.json \
     | tee qa/logs/TC-INT-005/memory-search.json
   ```
   - **Expected:** `reason_codes` includes a `hook_*` (or canonical) deny code.

2. Identify a source-FAILED tool and confirm reason:
   ```bash
   jq '[.[] | select(.availability=="unavailable" or .reason_codes | index("source_health"))]' \
     qa/logs/TC-INT-005/operator.json | tee qa/logs/TC-INT-005/source-health.json
   ```
   - **Expected:** Operator view lists the tool with `reason_codes` mentioning source health.

3. Force dispatch and confirm reason parity:
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/tools/agh__memory_search/invoke" -d '{}' \
     | tee qa/logs/TC-INT-005/dispatch-hook.json
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/tools/$SOURCE_FAILED_TOOL/invoke" -d '{}' \
     | tee qa/logs/TC-INT-005/dispatch-source.json
   ```
   - **Expected:** Both responses cite the same reason codes the projection used.

4. Repair source health (mock back to OK) and re-issue Step 3 for that tool:
   - **Expected:** Call succeeds. Operator projection updated.

5. Reload hook to remove the deny and re-issue Step 3 for `agh__memory_search`:
   - **Expected:** Call succeeds.

6. Run focused Go tests:
   ```bash
   go test ./internal/tools -run "TestPolicy|TestProjection" -count=1 | tee qa/logs/TC-INT-005/tools-tests.log
   go test ./internal/hooks -count=1 | tee qa/logs/TC-INT-005/hooks-tests.log
   ```

## Evidence To Capture

- All projection and dispatch payloads.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Multiple deny sources stacked | hook deny + source FAILED for same tool | Operator view shows both reason codes in deterministic order |
| Hook narrows but does not deny | hook reload allows but limits scope | Tool still callable; reason codes empty or annotated as `hook_narrowed` |
| Source health flapping | FAILED → OK → FAILED in tight loop | Operator view + dispatch must converge on the latest health state, not a stale cache |

## Channels Exercised

- Operator projection (`/api/tools`).
- Direct dispatch (`/api/tools/{id}/invoke`).
- Hooks subsystem.

## Related Test Cases

- TC-INT-001 (operator vs session projection divergence).
- TC-INT-006 (cache invalidation matrix).
- TC-FUNC-005 (hook tool family).
