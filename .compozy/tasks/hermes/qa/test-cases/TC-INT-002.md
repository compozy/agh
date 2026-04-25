## TC-INT-002: Observability Retention And Health Payloads

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that `observability.retention_days` drives deterministic retention sweeps and that observe health exposes typed persistence, retention, failure, probe, automation, and memory fields without deleting active debugging evidence.

### Traceability

- Task: task_02, Observability Retention and Health Base.
- TechSpec: issue 17; Testing Approach retention sweep by timestamp and disabled configuration.
- ADR: ADR-001 health foundation for later lifecycle and memory tracks.
- Surfaces: `internal/observe`, `internal/store/globaldb`, `internal/api/contract`, `internal/api/core`, `agh observe health`, site observe-health docs.

### Preconditions

- Isolated global DB with seeded event summaries, token stats, and permission logs.
- Config variants for positive retention, zero retention, disabled observability, and invalid negative retention.
- CLI/API checks run against isolated daemon or handler fixture.

### Test Steps

1. Seed observability rows older and newer than the configured cutoff.
   - **Expected:** Test data contains rows on both sides of the cutoff for each retained table.

2. Run the retention sweep with `retention_days > 0`.
   - **Expected:** Only rows older than cutoff are deleted from `event_summaries`, `token_stats`, and `permission_log`; active session catalog data and per-session event DBs remain intact.

3. Run health conversion after the sweep.
   - **Expected:** `health.retention` reports enabled state, retention days, last sweep status, cutoff timestamp, and deleted row counts.

4. Repeat with `retention_days = 0`.
   - **Expected:** Sweep is a no-op and health reports disabled or keep-history retention without deleting rows.

5. Validate negative retention configuration.
   - **Expected:** Config validation fails clearly and does not start the daemon with invalid retention.

6. Query `agh observe health -o json` or equivalent HTTP handler.
   - **Expected:** JSON includes typed `health.persistence`, `health.retention`, `health.failures`, `health.agent_probes`, and automation scheduler sections when present.

### Evidence To Capture

- `qa/logs/TC-INT-002/go-test-observe-health.log`
- `qa/logs/TC-INT-002/observe-health.json`
- Row-count evidence before and after sweep

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| `retention_days = 0` | Keep-history config | No rows deleted |
| Negative retention | `-1` | Validation error |
| Disabled observability | `enabled = false` | Retention no-op |
| Empty tables | No rows | Sweep succeeds with zero deleted counts |

### Related Test Cases

- TC-INT-003: Lifecycle failure health builds on observe health.
- TC-FUNC-001: Memory health/history must coexist with observe health.
