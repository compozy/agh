# TC-AUT-005: Concurrent Heartbeats Yield Single-Success Path

**Priority:** P0 (Critical)
**Type:** Concurrency
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that under cross-session contention only one heartbeat / complete / fail call against the same `run_id` succeeds. The other call returns a deterministic mismatch error (`AUTONOMY_FOREIGN_RUN` or `AUTONOMY_NO_ACTIVE_LEASE`).

## Traceability

- Task: task_09.
- TechSpec: "Test Strategy → Integration Tests" (concurrent heartbeats), "Safety Invariants".
- ADR: ADR-005.
- Surfaces: `internal/task/lease_manager.go`, `internal/tools/builtin/autonomy.go`, `internal/api/core/agent_tasks.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two sessions (`S_A`, `S_B`) in the same workspace.
- One queued task fixture.

## Test Steps

1. From `S_A`, claim:
   ```bash
   agh tool invoke agh__task_run_claim_next --session $S_A -o json | tee qa/logs/TC-AUT-005/claim-a.json
   RUN_ID=$(jq -r '.run.id' qa/logs/TC-AUT-005/claim-a.json)
   ```

2. Concurrent heartbeats from both sessions (using the test harness's parallel exec primitive):
   ```bash
   {
     agh tool invoke agh__task_run_heartbeat --session $S_A --input '{"run_id":"'$RUN_ID'"}' -o json \
       > qa/logs/TC-AUT-005/heartbeat-a.json &
     agh tool invoke agh__task_run_heartbeat --session $S_B --input '{"run_id":"'$RUN_ID'"}' -o json \
       > qa/logs/TC-AUT-005/heartbeat-b.json &
     wait
   }
   ```
   - **Expected:** `S_A` succeeds; `S_B` returns `AUTONOMY_FOREIGN_RUN`. Run only one set of writers.

3. **Concurrent complete attempts:**
   - Re-run claim with a new run ID, then race complete from `S_A` and complete from `S_B`.
   - **Expected:** First valid completion wins; the other returns `AUTONOMY_FOREIGN_RUN`.

4. **Concurrent heartbeat from same session (network retries):**
   - Race two heartbeats from `S_A` for the same `run_id` (simulate retried request).
   - **Expected:** Both succeed (idempotent semantics) OR one succeeds and one returns deterministic stale/conflict error per the writer contract. Document actual behavior; either outcome is acceptable as long as no token leak occurs and lease state remains consistent.

5. **Race against expiry:**
   - Use a 5-second lease; race a heartbeat at 4.9s with the reaper at 5.0s.
   - **Expected:** Either heartbeat succeeds (extending lease) or returns `AUTONOMY_LEASE_EXPIRED`. No silent corruption.

6. Inspect `task_runs` after each contention scenario to confirm there is at most one active lease per session.

7. Run focused Go tests:
   ```bash
   go test ./internal/task -run "TestConcurrentHeartbeat|TestConcurrentComplete" -race -count=1 \
     | tee qa/logs/TC-AUT-005/task-race-tests.log
   go test ./internal/tools/builtin -run "TestAutonomyConcurrent" -race -count=1 \
     | tee qa/logs/TC-AUT-005/builtin-race-tests.log
   ```

## Evidence To Capture

- Both concurrent payloads with timestamps.
- `task_runs` query output before/during/after contention.
- Test logs with `-race` enabled.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Two sessions both starve under wait-for-claim | both call `claim_next --wait` | Each returns deterministically (one task won by the first; the other returns `not_found` after timeout) |
| Approval required mid-heartbeat | tool with approval | Approval bridge integration runs once; concurrent re-issues remain consistent |
| Network partition between SQLite and the daemon | unlikely in lab | Test only documents behavior; not a P0 invariant |

## Channels Exercised

- Tool invoke, HTTP/UDS, hosted MCP for the same run ID.
- `task_runs` SQLite under `-race`.

## Related Test Cases

- TC-AUT-001 (happy path).
- TC-AUT-002 (foreign run).
- TC-AUT-006 (writer convergence).
