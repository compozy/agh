## TC-AUTO-008: Agent Task Lease API And CLI Lifecycle

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 55 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify agents can claim and maintain task-run leases through first-class UDS endpoints and CLI
verbs, with validated identity, stable JSON output, coordination channel metadata, and token
redaction after the claim response.

### Traceability

- Task: task_09, Agent Task Lease API And CLI Verbs.
- TechSpec: Agent Kernel CLI, API Endpoints, Task Claim and Lease.
- ADR: ADR-002, ADR-003, ADR-010, ADR-011, ADR-012.
- Resource lesson: Paperclip heartbeat CLI shows task_18 should capture live status/log output and terminal state, not just command exit.
- Surfaces: `internal/api/udsapi`, `internal/cli/task.go`, `internal/task`, daemon UDS client.

### Preconditions

- Active managed session with valid agent identity env.
- One queued workspace run with coordination channel metadata.
- One queued non-coordinated run for omission checks.

### Test Steps

1. Run `agh task next --wait --lease-seconds 300 -o json`.
   - **Expected:** Command claims one eligible run, returns raw `claim_token` exactly once, and includes channel metadata for channel-bound runs.

2. Run `agh task heartbeat <run-id> --claim-token "$CLAIM_TOKEN" -o json`.
   - **Expected:** Lease extends, response omits raw token, and read models show safe lease state only.

3. Send `agh ch send --kind status` using the claim response channel.
   - **Expected:** Message succeeds and task run remains active until a task API terminal command runs.

4. Run `agh task complete <run-id> --claim-token "$CLAIM_TOKEN" --result ... -o json`.
   - **Expected:** Run becomes terminal through token-fenced API and cannot be completed again with the same token.

5. Repeat negative CLI/UDS calls: invalid caller env, stale token, malformed JSON, permission denial, and no work found.
   - **Expected:** Structured errors/exit codes are stable for agents and do not echo raw token values.

### Evidence To Capture

- `qa/logs/TC-AUTO-008/task-next.json`
- `qa/logs/TC-AUTO-008/task-heartbeat.json`
- `qa/logs/TC-AUTO-008/task-complete.json`
- `qa/logs/TC-AUTO-008/stale-token-error.json`
- `qa/logs/TC-AUTO-008/uds-handler-negative.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Non-coordinated run | no channel ID | Claim response omits channel cleanly |
| `--wait` no work | no eligible run | Documented no-work response/exit code |
| Agent-created run | approved agent task | Same API lifecycle as manual run |
| Invalid caller | missing `AGH_SESSION_ID` | Identity-required error |

### Related Test Cases

- TC-AUTO-007: Domain lease invariants.
- TC-AUTO-014: Channel messages remain non-authoritative.
