## TC-INT-001: Persistence And Retry Foundations

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that global and session SQLite stores use the shared migration runner durably and that the shared retry helper is context-aware, bounded, and deterministic enough for later Hermes tracks.

### Traceability

- Task: task_01, Persistence and Retry Foundations.
- TechSpec: issues 10, 11, and 17; Testing Approach migration ordering, idempotence, failed rollback, retry backoff, jitter, cap, and cancellation.
- ADR: ADR-001 shared foundation sequencing.
- Surfaces: `internal/store`, `internal/store/globaldb`, `internal/store/sessiondb`, `internal/retry`, current task plan's backend P0 foundation.

### Preconditions

- Repository dependencies are installed.
- Use only temp directories for global and session database files.
- No live daemon state or user AGH home is used.

### Test Steps

1. Run focused store and retry tests for migration and retry behavior.
   - **Expected:** Tests pass and include ordered migration, repeated boot, rollback, integrity mismatch, context cancellation, jitter, and cap assertions.

2. Create a fresh global DB and inspect `schema_migrations`.
   - **Expected:** `schema_migrations` exists, migration rows are ordered by version, names/checksums are non-empty, and expected global tables exist.

3. Reopen the same global DB without deleting files.
   - **Expected:** Existing migration rows remain unchanged and no migration reapplies.

4. Create and reopen a fresh session DB.
   - **Expected:** Session DB also records migration rows and repeated boot is idempotent.

5. Exercise a failing migration in an isolated test DB.
   - **Expected:** The operation returns a wrapped failure, data changes from the failed migration are rolled back, and no success row is inserted.

6. Exercise retry cancellation and non-retryable error paths.
   - **Expected:** Retry returns immediately on canceled context, honors max attempts, and does not retry permanent errors.

### Evidence To Capture

- `qa/logs/TC-INT-001/go-test-store-retry.log`
- DB query output or test assertion log showing `schema_migrations`
- Any failure details in `qa/issues/BUG-*.md`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Duplicate migration version | Two migrations with same version | Validation failure before applying |
| Tampered checksum | Existing row checksum changed | Integrity mismatch failure |
| Context canceled before retry | Canceled context | No delay wait and cancellation returned |
| Session DB reopen | Same temp session DB path | No duplicate migration rows |

### Related Test Cases

- TC-INT-002: Observability retention depends on global store durability.
- TC-INT-004: Automation scheduler state depends on migration guarantees.
