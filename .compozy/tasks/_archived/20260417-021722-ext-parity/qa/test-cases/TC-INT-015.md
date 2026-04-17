# TC-INT-015: Operator write creates automation through reconcile

**Priority:** P0
**Type:** Integration
**Package:** internal/api/udsapi, internal/automation
**Related Tasks:** 10

## Objective

Validate that a PUT request for `automation.job` via the UDS API creates a resource record that triggers reconciliation, which in turn rebuilds the automation runtime from resource records rather than legacy storage. This proves the automation subsystem is fully driven by the resource store.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- UDS API server running on a test socket
- Resource store initialized with automation projector wired
- Automation runtime initialized and listening for projected state changes
- No legacy automation storage configured

## Test Steps

1. Send a PUT request to the UDS API to create an `automation.job` resource with `id=auto-job-1` and a valid job configuration in the data payload (e.g., trigger, actions, schedule).
   **Expected:** 200/201 response. Resource record created in the store.

2. Verify the resource record exists in `resource_records` with `kind=automation.job`, `id=auto-job-1`, `source=operator`.
   **Expected:** Record present with correct data payload.

3. Wait for reconciliation to complete (projector Apply triggered).
   **Expected:** Automation projector processes the new record.

4. Query the automation runtime for active jobs.
   **Expected:** `auto-job-1` is registered and active in the automation runtime with the correct configuration.

5. Send a PUT request to update `auto-job-1` with a modified schedule.
   **Expected:** 200 response. Resource record updated.

6. Wait for reconciliation.
   **Expected:** Automation runtime reflects the updated schedule for `auto-job-1`.

7. Send a DELETE request for `auto-job-1`.
   **Expected:** 200 response. Resource record removed.

8. Wait for reconciliation.
   **Expected:** `auto-job-1` is no longer active in the automation runtime.

## Edge Cases

- PUT with invalid job configuration — rejected at API level with 400, no record created
- PUT with the same ID twice — updates existing record, does not create duplicate
- DELETE for non-existent job — 404 or no-op, no error in reconciliation
- Rapid PUT/DELETE cycle — final state is consistent (deleted if DELETE was last)
- Automation projector failure after PUT — record persists, runtime retries on next reconcile
- Multiple automation jobs created in rapid succession — all appear in runtime after reconcile
