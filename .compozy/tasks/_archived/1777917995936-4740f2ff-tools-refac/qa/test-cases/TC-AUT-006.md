# TC-AUT-006: Tool / CLI / HTTP / UDS Converge On The Same `task.Service` Lease Writers

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the autonomy bridge does not create a parallel writer path. Tool, CLI, HTTP, and UDS calls that perform the same lifecycle action must hit the same `task.Service` lease writers and produce identical observable effects on `task_runs` rows, observe events, and `claim_token_hash` lifecycle metadata.

## Traceability

- Task: task_09.
- TechSpec: "Bootstrap Task Tools → required writer mapping", "Safety Invariants".
- ADR: ADR-003.
- Surfaces: `internal/tools/builtin/autonomy.go`, `internal/task/lease_manager.go`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A queued task fixture per surface, so each round-trip uses its own run.

## Test Steps

For each surface in {tool, CLI, HTTP, UDS}, run the same lifecycle:

1. claim → heartbeat → complete on a fresh task fixture.
2. claim → release on a fresh fixture.
3. claim → fail on a fresh fixture.

For each step, capture:

- The surface response payload.
- The `task_runs` row before and after.
- The corresponding observe event(s).
- The daemon log entries that show the writer (`task.Service.*`) being invoked.

Compare across the four surfaces:

```bash
diff <(jq -S 'del(.run.id, .run.lease_until)' qa/logs/TC-AUT-006/tool-claim.json) \
     <(jq -S 'del(.run.id, .run.lease_until)' qa/logs/TC-AUT-006/cli-claim.json)
diff <(jq -S 'del(.run.id, .run.lease_until)' qa/logs/TC-AUT-006/uds-claim.json) \
     <(jq -S 'del(.run.id, .run.lease_until)' qa/logs/TC-AUT-006/http-claim.json)
```

- **Expected:** Modulo run-id and lease-until, response shapes are identical. `claim_token_hash` is consistent. `task_runs` row state transitions match. Observe events emit the same `actor_kind`/`actor_id` shape across surfaces.

Confirm via daemon log that each request invokes exactly one of `ClaimNextRun`, `HeartbeatRunLease`, `CompleteRunLease`, `FailRunLease`, `ReleaseRunLease` per call. There must be no parallel lease writer.

Run focused Go tests:

```bash
go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/tools/builtin \
  -run "TestAutonomy|TestAgentTask" -count=1 | tee qa/logs/TC-AUT-006/parity-tests.log
go test ./internal/task -count=1 | tee qa/logs/TC-AUT-006/task-tests.log
```

## Evidence To Capture

- 12 round-trip payloads (3 lifecycles × 4 surfaces).
- 12 `task_runs` snapshots before/after.
- Aggregated observe-event log filtered for autonomy event types.
- Daemon log filtered for `task.Service.*` invocations.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Hosted MCP variant | route the same lifecycle via hosted MCP `tools/call` | Equivalent `task.Service` invocation; payloads consistent with tool/UDS responses |
| Operator-triggered lease release via `agh task release` | operator runs explicit release | Surfaced through the same writer; emits identical observe events |
| Variant with idempotency_key on claim | repeat call with same key | Single `ClaimNextRun` invocation; second call returns same `run_id` without minting a new lease |

## Channels Exercised

- Tool / CLI / HTTP / UDS / hosted MCP (optional but recommended).

## Related Test Cases

- TC-AUT-001 (happy path).
- TC-AUT-002, TC-AUT-005 (mismatch and contention proof).
- TC-INT-002 (transport parity for the rest of the catalog).
