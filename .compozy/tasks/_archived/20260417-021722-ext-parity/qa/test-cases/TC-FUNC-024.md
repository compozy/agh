# TC-FUNC-024: Automation runs survive definition cutover

**Priority:** P1
**Type:** Functional
**Package:** internal/automation
**Related Tasks:** 10

## Objective

Validate that after migrating automation definitions from the legacy catalog to resource-backed records, existing automation run history (past executions, timestamps, exit statuses) and active locks (in-progress job exclusion locks) remain readable and functional. The cutover must not orphan historical data or break lock semantics for jobs that were running during the migration.

## Preconditions

- Legacy automation system has historical run records for job "nightly-backup" (at least 5 past runs with varied statuses: success, failure, skipped).
- An exclusion lock exists for a currently running instance of "nightly-backup" (simulated or real).
- The migration from legacy automation definitions to resource-backed records has been executed.
- The resource store contains the new automation.job record for "nightly-backup".

## Test Steps

1. Query the automation run history for "nightly-backup" after the migration.
   **Expected:** All historical runs are returned with correct timestamps, exit statuses, durations, and any associated metadata. No runs are missing compared to pre-migration state.

2. Verify the most recent run entry has the correct status and timestamp.
   **Expected:** The latest run matches the pre-migration state exactly. No timestamp drift or status corruption.

3. Query the active lock for "nightly-backup".
   **Expected:** The exclusion lock is still present and valid. Lock holder identity, acquisition timestamp, and TTL (if applicable) are preserved.

4. Attempt to start a second concurrent instance of "nightly-backup".
   **Expected:** The lock prevents concurrent execution. The system returns a "job already running" or equivalent rejection.

5. Release the existing lock (simulate the running job completing).
   **Expected:** Lock is released. Subsequent run attempts are no longer blocked.

6. Trigger a new run of "nightly-backup" through the resource-backed automation system.
   **Expected:** The job executes successfully. A new run history entry is appended with correct metadata. The run is attributed to the resource-backed definition, not the legacy one.

7. Query run history again.
   **Expected:** Both legacy historical runs and the new resource-backed run appear in a single, unified timeline. Ordering is correct by timestamp.

## Edge Cases

- Job that existed in legacy but was NOT migrated to a resource record: run history is still queryable even without an active resource definition (orphaned history is preserved, not deleted).
- Lock that expires during migration: the lock TTL logic still applies post-migration; expired locks are cleaned up normally.
- Run history entries with legacy-specific fields not present in the resource schema: these fields are preserved in a metadata sidecar or gracefully omitted without data loss for standard fields.
- Concurrent migration and job execution: a job that starts before migration and ends after migration writes its completion status correctly.
- Querying runs by time range that spans pre-migration and post-migration: returns a unified result set.
