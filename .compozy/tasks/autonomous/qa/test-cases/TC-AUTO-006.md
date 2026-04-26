## TC-AUTO-006: Task Run Lease Schema, Capability Rows, And Redacted Reads

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify SQLite task-run schema and read models preserve lease metadata, coordination channel
association, exact-match capability rows, indexes, restart reads, and raw-token redaction.

### Traceability

- Task: task_07, Task Claim Lease Schema.
- TechSpec: Data Models, Task capability fields, Lease invariants, Task-Channel Coordination Contract.
- ADR: ADR-003, ADR-009, ADR-010, ADR-011, ADR-012.
- Resource lesson: Multica issue/store references favor indexed durable state for issue/run filtering.
- Surfaces: `internal/task`, `internal/store/globaldb`, `internal/api/contract/tasks.go`.

### Preconditions

- Isolated SQLite global database.
- Fixtures for queued, claimed, channel-bound, capability-restricted, and terminal task runs.
- Public read endpoints or conversion helpers are available for run list/detail checks.

### Test Steps

1. Create or migrate the global DB schema.
   - **Expected:** `task_runs` includes claim-token hash, lease timestamps, heartbeat timestamp, and `coordination_channel_id`; capability side tables and indexes exist.

2. Persist a task run with required/preferred capabilities and a coordination channel.
   - **Expected:** Rows round-trip deterministically, and channel lookup returns the run by `coordination_channel_id`.

3. Persist an active lease with raw token hash data and reopen the DB.
   - **Expected:** Lease fields survive reopen/restart and raw token material is not returned by normal read helpers.

4. Convert/list/get task runs through public DTOs.
   - **Expected:** DTOs expose safe lease state or `claim_token_hash` only; no raw `claim_token` appears.

5. Query candidate runs by capability requirements.
   - **Expected:** Exact-match side tables filter without JSON parsing and preserve empty capability sets.

### Evidence To Capture

- `qa/logs/TC-AUTO-006/schema-inspection.log`
- `qa/logs/TC-AUTO-006/capability-rows.log`
- `qa/logs/TC-AUTO-006/restart-read.log`
- `qa/logs/TC-AUTO-006/read-model-redaction.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Empty capabilities | no required/preferred rows | Run remains claimable by capability-neutral criteria |
| Multiple capabilities | repeated rows | Rows sort/dedupe deterministically |
| Missing channel | non-coordinated run | Channel field omitted or empty without fake value |
| Raw token in metadata | metadata contains `claim_token` | Validation rejects or redacts before public read |

### Related Test Cases

- TC-AUTO-007: Claim service uses this schema.
- TC-AUTO-002: Contract redaction parity.
