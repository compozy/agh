# TC-INT-006: Cache Invalidation Across Runtime Mutation Triggers

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the projection cache invalidates on every key listed in the TechSpec: agent reload, lineage change, toolset/descriptor reload, extension/skill registry change, hook reload, MCP auth health change that affects availability, and config overlay changes that affect tool policy. Confirm the cache never becomes authority — dispatch revalidates regardless of cache state.

## Traceability

- Task: task_01.
- TechSpec: "Known Risks → cache invalidation".
- ADR: ADR-002.
- Surfaces: `internal/tools/policy.go`, `internal/daemon/native_tools.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A session bound to a default agent.
- Hosted MCP not strictly required, but sessions endpoint is.

## Test Steps

For each invalidation key below: capture the session projection before and after the mutation; assert the projection changed when the mutation should change effective policy.

1. **Agent reload:**
   - Edit the agent definition to deny one tool.
   - Trigger reload.
   - Capture session projection before/after.
   - **Expected:** After projection no longer contains the denied tool.

2. **Lineage change:**
   - Spawn a child session under a different parent (or rebind lineage).
   - Capture before/after projections.
   - **Expected:** Projection reflects the new lineage's narrowing rules.

3. **Toolset / descriptor reload:**
   - Reload an extension whose manifest adds/removes a descriptor.
   - **Expected:** Projection updates to include / exclude the affected tool.

4. **Extension / skill registry change:**
   - Install or remove a skill that modifies `agh__catalog` membership.
   - **Expected:** `agh__skill_*` set updates.

5. **Hook reload:**
   - Add a new hook that denies one tool.
   - **Expected:** Projection drops it; dispatch denies with the same reason code.

6. **MCP auth health change:**
   - Force `mcp-server-a` from `connected` to `expired`.
   - **Expected:** Tools whose source is `mcp-server-a` flip to `unavailable` in operator view; session projection drops them.

7. **Config overlay change affecting tool policy:**
   - Set `[tools.policy].approval_timeout_seconds` (or a similar key) and reload config.
   - **Expected:** Subsequent calls reflect the new policy; cache must not return stale approval timeouts.

8. **Negative test — cache as authority:**
   - Force a stale cache hit (e.g., wait for cache TTL window) by mutating runtime AND immediately re-issuing the projection on the cache key. Then call the tool.
   - **Expected:** Even if projection returns the stale entry, dispatch revalidates and rejects with current reason. (No cache-as-authority regression.)

9. Run focused Go tests:
   ```bash
   go test ./internal/tools -run "TestCache|TestInvalidation|TestPolicy" -count=1 \
     | tee qa/logs/TC-INT-006/tools-tests.log
   go test ./internal/daemon -run "TestNativeTools" -count=1 \
     | tee qa/logs/TC-INT-006/daemon-tests.log
   ```

## Evidence To Capture

- For each invalidation key, before/after projection JSON under `qa/logs/TC-INT-006/key-<n>-{before,after}.json`.
- Diff of each pair.
- Negative-test log proving dispatch revalidates.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Two invalidation keys mutated simultaneously | agent reload + hook reload | Single re-projection reflects both |
| Mutation that does not affect tool policy | unrelated config write | Projection returns same set; no spurious churn |
| Hosted MCP bound during invalidation | rebind required | Old hosted MCP `tools/list` becomes invalid; subsequent bind sees current state |

## Channels Exercised

- HTTP/UDS projection.
- Hosted MCP projection (when applicable).
- Daemon policy resolver and cache.

## Related Test Cases

- TC-FUNC-001 (default discovery overlay + per-call recompute).
- TC-INT-001 (operator vs session projection divergence).
- TC-INT-005 (hook / source-health denial reason codes).
