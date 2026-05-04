# TC-AUT-001: Session-Bound Autonomy Claim → Heartbeat → Complete Flow

**Priority:** P0 (Critical)
**Type:** Functional / Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the canonical autonomy round-trip works through the new `agh__autonomy` tool family and that the public contract is keyed by `run_id` only. Confirm `task.Service.ClaimNextRun`, `HeartbeatRunLease`, `CompleteRunLease`, `FailRunLease`, and `ReleaseRunLease` are the sole writers reached.

## Traceability

- Task: task_09.
- TechSpec: "Session-Bound Autonomy Lookup", "Bootstrap Task Tools", "Implementation Steps".
- ADRs: ADR-003, ADR-005.
- Surfaces: `internal/tools/builtin/autonomy.go`, `internal/task/{lease.go,lease_manager.go}`, `internal/api/core/agent_tasks.go`, `internal/cli/task.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A workspace + one session bound to an agent that satisfies the queued task's `required_capabilities`.
- A queued task fixture eligible for claim.

## Test Steps

1. **Claim:**
   ```bash
   agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/TC-AUT-001/claim.json
   RUN_ID=$(jq -r '.run.id' qa/logs/TC-AUT-001/claim.json)
   ```
   - **Expected:** Response contains `run.id`, `run.lease_until`, `run.claim_token_hash` (observability metadata), and zero `claim_token` field. The `task_runs` row transitions to `claimed`.

2. **Heartbeat:**
   ```bash
   agh tool invoke agh__task_run_heartbeat --input '{"run_id":"'$RUN_ID'","lease_seconds":60}' -o json \
     | tee qa/logs/TC-AUT-001/heartbeat.json
   ```
   - **Expected:** `lease_until` advances; row remains `claimed` / `running`.

3. **Complete:**
   ```bash
   agh tool invoke agh__task_run_complete --input '{"run_id":"'$RUN_ID'","result":{"ok":true}}' -o json \
     | tee qa/logs/TC-AUT-001/complete.json
   ```
   - **Expected:** Row transitions to `completed`; result persisted.

4. **Confirm writer convergence:**
   - Inspect daemon log for the call sequence; confirm `task.Service` lease writers are invoked exactly once per step.
   - Confirm no parallel writer (e.g., direct SQL update) was used.

5. **Negative — fail path:**
   - Re-run Steps 1-2 against a new claim, then call `agh__task_run_fail`:
   ```bash
   agh tool invoke agh__task_run_fail --input '{"run_id":"'$RUN_ID2'","error":"qa-fail"}' -o json \
     | tee qa/logs/TC-AUT-001/fail.json
   ```
   - **Expected:** Row transitions to `failed`; failure metadata stored.

6. **Negative — release path:**
   - Re-run Steps 1-2 against a new claim, then call `agh__task_run_release`:
   ```bash
   agh tool invoke agh__task_run_release --input '{"run_id":"'$RUN_ID3'","reason":"qa-release"}' -o json \
     | tee qa/logs/TC-AUT-001/release.json
   ```
   - **Expected:** Row returns to queue, lease released. Re-claim from same session is allowed.

7. **CLI parity:**
   ```bash
   agh task next -o json | tee qa/logs/TC-AUT-001/cli-next.json
   ```
   - **Expected:** CLI returns the run summary without raw token fields. CLI heartbeat/complete/fail/release accept `run_id` and not `--claim-token`.

8. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/agents/tasks/runs/next" -d '{}' | tee qa/logs/TC-AUT-001/uds-next.json
   ```
   - **Expected:** Same behavior; no `claim_token` field anywhere.

9. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestAutonomy" -count=1 | tee qa/logs/TC-AUT-001/builtin-tests.log
   go test ./internal/task -count=1 | tee qa/logs/TC-AUT-001/task-tests.log
   go test ./internal/api/core -run "TestAgentTask" -count=1 | tee qa/logs/TC-AUT-001/api-core-tests.log
   ```

## Evidence To Capture

- All `qa/logs/TC-AUT-001/*.json` payloads.
- `task_runs` table query log showing state transitions per step.
- Daemon log filtered for autonomy events.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Wait-for-claim with empty queue | `{"wait":true}` | Tool blocks for the configured timeout; returns deterministic timeout reason |
| Idempotent claim with `idempotency_key` | repeat claim with same key | Second call returns the same `run_id` without minting a new lease |
| Heartbeat with no `lease_seconds` | `{"run_id":"..."}` | Default lease_seconds applied per writer |
| Complete with malformed result | `{"run_id":"...","result":"not-an-object"}` | Validator rejects with deterministic error |

## Channels Exercised

- Tool, CLI, HTTP/UDS.
- `task_runs` SQLite.
- Daemon log.

## Related Test Cases

- TC-AUT-002..006 (negative paths and contention).
- TC-SEC-001 (claim_token redaction sweep).
- TC-AUT-006 (writer convergence proof).
