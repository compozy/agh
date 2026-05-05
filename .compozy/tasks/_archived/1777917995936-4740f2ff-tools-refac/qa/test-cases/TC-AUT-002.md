# TC-AUT-002: `AUTONOMY_FOREIGN_RUN` Cross-Session Denial

**Priority:** P0 (Critical)
**Type:** Functional / Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that a session cannot heartbeat / complete / fail / release a run claimed by a different session. The daemon must return `AUTONOMY_FOREIGN_RUN` deterministically across tool, CLI, HTTP, and UDS surfaces.

## Traceability

- Task: task_09.
- TechSpec: "Session-Bound Autonomy Lookup".
- ADR: ADR-005.
- Surfaces: `internal/tools/builtin/autonomy.go`, `internal/task/lease_manager.go`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two sessions in the same workspace: `S_A`, `S_B`. Both bound to compatible agents.
- One queued task fixture.

## Test Steps

1. From `S_A`, claim:
   ```bash
   agh tool invoke agh__task_run_claim_next --session $S_A -o json | tee qa/logs/TC-AUT-002/claim-a.json
   RUN_ID=$(jq -r '.run.id' qa/logs/TC-AUT-002/claim-a.json)
   ```

2. From `S_B`, attempt heartbeat on the same `run_id`:
   ```bash
   agh tool invoke agh__task_run_heartbeat --session $S_B --input '{"run_id":"'$RUN_ID'"}' -o json \
     | tee qa/logs/TC-AUT-002/heartbeat-b.json
   ```
   - **Expected:** `error.code=AUTONOMY_FOREIGN_RUN`. No `task_runs` mutation.

3. From `S_B`, attempt complete:
   ```bash
   agh tool invoke agh__task_run_complete --session $S_B --input '{"run_id":"'$RUN_ID'","result":{}}' -o json \
     | tee qa/logs/TC-AUT-002/complete-b.json
   ```
   - **Expected:** Same `AUTONOMY_FOREIGN_RUN`.

4. From `S_B`, attempt fail and release. Same expected code.

5. **CLI parity (run from a shell whose session context is `S_B`):**
   ```bash
   AGH_SESSION=$S_B agh task heartbeat $RUN_ID -o json | tee qa/logs/TC-AUT-002/cli-heartbeat-b.json
   ```
   - **Expected:** Same `AUTONOMY_FOREIGN_RUN`.

6. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     -H "X-AGH-Session: $S_B" \
     "http://localhost/api/agents/tasks/runs/$RUN_ID/heartbeat" -d '{}' \
     | tee qa/logs/TC-AUT-002/uds-heartbeat-b.json
   ```
   - **Expected:** Same `AUTONOMY_FOREIGN_RUN`.

7. **Hosted MCP parity:** call the autonomy tool via hosted MCP bound to `S_B`.
   - **Expected:** Same code.

8. Confirm `task_runs` lease state never mutated by the foreign session — query SQLite and compare to the snapshot taken before Step 2.

9. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestAutonomyForeign" -count=1 | tee qa/logs/TC-AUT-002/builtin-tests.log
   go test ./internal/task -run "TestLease.*Foreign" -count=1 | tee qa/logs/TC-AUT-002/task-tests.log
   ```

## Evidence To Capture

- All foreign-attempt payloads per surface.
- `task_runs` snapshot before/after.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Foreign session is in a different workspace | cross-workspace attempt | `AUTONOMY_FOREIGN_RUN` (or `AUTONOMY_NO_ACTIVE_LEASE`, depending on lookup order) |
| Empty session scope | call with no session | `AUTONOMY_SESSION_REQUIRED` |
| Foreign attempt during heartbeat race window | heartbeat ↔ complete contention | Daemon returns deterministic `AUTONOMY_FOREIGN_RUN` |

## Channels Exercised

- Tool, CLI, HTTP/UDS, hosted MCP.

## Related Test Cases

- TC-AUT-001 (happy path).
- TC-AUT-005 (concurrent heartbeats single-success).
- TC-AUT-006 (writer convergence).
