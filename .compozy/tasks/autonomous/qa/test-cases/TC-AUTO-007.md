## TC-AUTO-007: ClaimNextRun And Token-Fenced Lease Mutations

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify `ClaimNextRun(criteria)` is the authoritative next-work primitive and that heartbeat,
release, complete, fail, and expired-lease recovery are fenced by the current raw claim token,
owning session, and SQLite transaction boundaries.

### Traceability

- Task: task_08, ClaimNextRun And Lease Fencing Service.
- TechSpec: Scheduler and Claim Authority, Lease invariants, Data Flow.
- ADR: ADR-003, ADR-004, ADR-009, ADR-010, ADR-012.
- Resource lesson: Paperclip issue-run orchestration requires centralized ownership locks; heartbeat CLI requires terminal-status and timeout evidence.
- Surfaces: `internal/task`, `internal/store/globaldb`, task hooks, lease recovery.

### Preconditions

- Two or more active sessions in the same workspace.
- Queued runs with mixed priority, capabilities, and channel-bound metadata.
- Deterministic clock or testable lease expiry controls.

### Test Steps

1. Run concurrent `ClaimNextRun` attempts against one queued run.
   - **Expected:** Exactly one session claims the run; losers receive no-work/lease conflict without partial ownership.

2. Claim a channel-bound run successfully.
   - **Expected:** Result includes raw `claim_token` once, lease deadline, owning session, task/run summaries, and safe channel metadata.

3. Heartbeat, complete, fail, and release using missing, stale, mismatched, and current tokens.
   - **Expected:** Only the current token from the owning session can mutate the active run.

4. Enforce one active lease per session.
   - **Expected:** A session with an active lease cannot claim a second run until it completes, fails, releases, or expires.

5. Expire or recover a lease, then retry stale heartbeat and late complete from the old holder.
   - **Expected:** Stale operations fail explicitly and cannot overwrite the recovered/new owner state.

6. Verify hook emission around claim and lease lifecycle.
   - **Expected:** Pre-claim dispatch occurs before transaction commit; post-claim/release/recovery hooks contain safe identifiers and no raw token.

### Evidence To Capture

- `qa/logs/TC-AUTO-007/concurrent-claim.log`
- `qa/logs/TC-AUTO-007/claim-response.json`
- `qa/logs/TC-AUTO-007/token-fencing.log`
- `qa/logs/TC-AUTO-007/expired-recovery.log`
- `qa/logs/TC-AUTO-007/lease-hooks.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Capability mismatch | run requires unavailable capability | Claim skips run |
| Unexpired active lease | another session owns run | Claim skips run |
| Recovered lease | old token after recovery | Heartbeat/complete fails explicitly |
| Release reason | `handoff`, `ttl_expired` | Reason recorded without token leakage |

### Related Test Cases

- TC-AUTO-008: Public CLI/UDS claim and lease flow.
- TC-AUTO-010: Scheduler recovery delegates to task service.
