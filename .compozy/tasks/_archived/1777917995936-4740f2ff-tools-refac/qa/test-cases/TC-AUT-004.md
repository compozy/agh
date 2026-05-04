# TC-AUT-004: `AUTONOMY_NO_ACTIVE_LEASE` And `AUTONOMY_LEASE_EXPIRED`

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__task_run_heartbeat`, `_complete`, `_fail`, and `_release` reject calls when:

- The calling session has no active lease (`AUTONOMY_NO_ACTIVE_LEASE`).
- The active lease is past its `lease_until` (`AUTONOMY_LEASE_EXPIRED`).

## Traceability

- Task: task_09.
- TechSpec: "Session-Bound Autonomy Lookup".
- ADR: ADR-005.
- Surfaces: `internal/tools/builtin/autonomy.go`, `internal/task/lease_manager.go`.

## Preconditions

- Isolated `AGH_HOME`.
- One session.
- A queued task fixture.

## Test Steps

1. **No active lease — heartbeat:**
   ```bash
   agh tool invoke agh__task_run_heartbeat --input '{"run_id":"non-existent"}' -o json \
     | tee qa/logs/TC-AUT-004/heartbeat-no-lease.json
   ```
   - **Expected:** `error.code=AUTONOMY_NO_ACTIVE_LEASE`.

2. **No active lease — complete / fail / release:** repeat with each verb. Same error code.

3. **Expired lease:**
   - Claim a run with a short `lease_seconds` (e.g., 5).
   - Wait until past expiry.
   - Heartbeat the lease.
   ```bash
   agh tool invoke agh__task_run_claim_next --input '{"lease_seconds":5}' -o json \
     | tee qa/logs/TC-AUT-004/claim-short.json
   sleep 10
   RUN_ID=$(jq -r '.run.id' qa/logs/TC-AUT-004/claim-short.json)
   agh tool invoke agh__task_run_heartbeat --input '{"run_id":"'$RUN_ID'"}' -o json \
     | tee qa/logs/TC-AUT-004/heartbeat-expired.json
   ```
   - **Expected:** `error.code=AUTONOMY_LEASE_EXPIRED`.

4. **Expired lease — complete:** same expired window, attempt complete.
   - **Expected:** Same `AUTONOMY_LEASE_EXPIRED`.

5. **Empty session scope:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/agents/tasks/runs/$RUN_ID/heartbeat" -d '{}' \
     | tee qa/logs/TC-AUT-004/uds-no-session.json
   ```
   - **Expected:** Without a session header / context, `AUTONOMY_SESSION_REQUIRED`.

6. **Recovery:** ensure that after the expiry the run is requeue-able and another claim can succeed.
   - **Expected:** New claim against the same task succeeds (lease released by reaper).

7. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestAutonomyExpired|TestAutonomyNoLease" -count=1 \
     | tee qa/logs/TC-AUT-004/builtin-tests.log
   go test ./internal/task -run "TestLeaseExpiry|TestLeaseLookup" -count=1 \
     | tee qa/logs/TC-AUT-004/task-tests.log
   ```

## Evidence To Capture

- All deny payloads.
- `task_runs` snapshots before/after expiry.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Heartbeat on a `released` run | release first, then heartbeat | `AUTONOMY_NO_ACTIVE_LEASE` (lease no longer active) |
| Complete after lease expired but before reaper runs | tight timing | `AUTONOMY_LEASE_EXPIRED` |
| Reaper races with heartbeat | concurrent reaper + heartbeat | One outcome only; deterministic; documented behavior |

## Channels Exercised

- Tool, CLI, HTTP/UDS.

## Related Test Cases

- TC-AUT-001 (happy path).
- TC-AUT-002 (foreign run).
- TC-AUT-003 (lease already held).
