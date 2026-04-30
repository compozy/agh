# TC-AUT-003: `AUTONOMY_LEASE_ALREADY_HELD` On Second `run_claim_next`

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the single-active-lease-per-session invariant: a session that already holds an active lease cannot claim a new run. The deterministic error is `AUTONOMY_LEASE_ALREADY_HELD`.

## Traceability

- Task: task_09.
- TechSpec: "Session-Bound Autonomy Lookup → run_claim_next invariants", "Safety Invariants".
- ADR: ADR-005.
- Surfaces: `internal/tools/builtin/autonomy.go`, `internal/task/lease_manager.go`.

## Preconditions

- Isolated `AGH_HOME`.
- One session bound to an agent satisfying capabilities for two queued tasks.

## Test Steps

1. First claim:
   ```bash
   agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/TC-AUT-003/claim-1.json
   ```

2. Second claim while the first lease is still active:
   ```bash
   agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/TC-AUT-003/claim-2.json
   ```
   - **Expected:** `error.code=AUTONOMY_LEASE_ALREADY_HELD`.

3. Release the first claim:
   ```bash
   RUN1=$(jq -r '.run.id' qa/logs/TC-AUT-003/claim-1.json)
   agh tool invoke agh__task_run_release --input '{"run_id":"'$RUN1'","reason":"qa-test"}' -o json \
     | tee qa/logs/TC-AUT-003/release-1.json
   ```

4. Re-attempt the second claim:
   ```bash
   agh tool invoke agh__task_run_claim_next -o json | tee qa/logs/TC-AUT-003/claim-3.json
   ```
   - **Expected:** Success — a new lease is acquired.

5. **CLI parity:**
   ```bash
   agh task next -o json | tee qa/logs/TC-AUT-003/cli-next-while-held.json
   ```
   - **Expected:** Same `AUTONOMY_LEASE_ALREADY_HELD` if a lease is currently held by the same session context.

6. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestAutonomyLeaseAlreadyHeld|TestAutonomyClaim" -count=1 \
     | tee qa/logs/TC-AUT-003/builtin-tests.log
   go test ./internal/task -run "TestLeaseAlreadyHeld" -count=1 | tee qa/logs/TC-AUT-003/task-tests.log
   ```

## Evidence To Capture

- All claim / release payloads.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Session held lease but the lease is expired | call after lease TTL passes | `AUTONOMY_LEASE_EXPIRED` (covered by TC-AUT-004) |
| Idempotent claim with same key | repeat with `idempotency_key` | Second call returns the same `run_id` (NOT `AUTONOMY_LEASE_ALREADY_HELD`) |
| Different session sharing same workspace | different session ID | New session can claim independently |

## Channels Exercised

- Tool, CLI.

## Related Test Cases

- TC-AUT-001 (happy path).
- TC-AUT-004 (no active lease / expired lease).
