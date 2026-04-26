## TC-AUTO-017: End-To-End Coordinated Run From Manual Task To Token-Fenced Completion

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 75 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the primary autonomy MVP workflow through public surfaces: a user creates durable task
intent, explicitly starts execution, the run binds a channel, coordinator bootstrap occurs, an
eligible worker claims through `agh task next`, communicates through the channel, heartbeats, and
completes through token-fenced task APIs.

### Traceability

- Task: tasks 01-14 end-to-end, with task_15/task_16 for UI/docs corroboration where used.
- TechSpec: Data Flow, Manual Control Contract, Scheduler and Claim Authority, Coordinator Trigger, Task-Channel Coordination Contract.
- ADR: ADR-003, ADR-004, ADR-005, ADR-006, ADR-010, ADR-012.
- Resource lesson: Hermes trajectory references require durable run/action evidence; Paperclip orchestration plan requires at most one active owner.
- Surfaces: daemon, SQLite, CLI/UDS, task service, network channel, scheduler, coordinator, session manager.

### Preconditions

- Isolated daemon with coordinator enabled for a workspace.
- At least one agent/session fixture capable of claiming the run.
- Agent command env is configured with `AGH_SESSION_ID` and `AGH_AGENT`.
- Evidence directories exist under `qa/logs/TC-AUTO-017/`.

### Test Steps

1. Create a user task through CLI/API.
   - **Expected:** Task exists as intent only, with no run, no claimable work, and no coordinator.

2. Start or publish the task.
   - **Expected:** Exactly one workspace-scoped run is enqueued, a stable coordination channel is bound, and coordinator bootstrap conditions are evaluated.

3. Verify coordinator session state.
   - **Expected:** One coordinator session is active or reused for the workspace with restricted permissions and situation context.

4. Have an eligible worker run `agh task next --wait -o json`.
   - **Expected:** Worker claims the run through the task service and receives raw token plus channel metadata.

5. Worker sends `agh ch send --kind status` and `--kind result` messages.
   - **Expected:** Messages are persisted as conversation only; run remains active until task API completion.

6. Worker heartbeats and completes with `agh task heartbeat` and `agh task complete`.
   - **Expected:** Lease extends and terminal state changes only through current raw token.

7. Attempt stale duplicate complete after terminal state.
   - **Expected:** Duplicate/stale mutation fails and final result remains unchanged.

### Evidence To Capture

- `qa/logs/TC-AUTO-017/task-create.json`
- `qa/logs/TC-AUTO-017/task-start.json`
- `qa/logs/TC-AUTO-017/coordinator-session.json`
- `qa/logs/TC-AUTO-017/worker-claim.json`
- `qa/logs/TC-AUTO-017/channel-messages.jsonl`
- `qa/logs/TC-AUTO-017/task-complete.json`
- `qa/logs/TC-AUTO-017/final-run-state.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Coordinator disabled | same run | Run can still be claimed manually, no coordinator bootstrap |
| Worker unavailable | no eligible idle session | Scheduler records no-match, no claim |
| Restart before claim | daemon restart after enqueue | Run/channel persist and coordinator recovery occurs if configured |
| Restart during lease | daemon restart after claim expires | Lease recovery makes run claimable without stale completion |

### Related Test Cases

- TC-AUTO-009: Execution boundary.
- TC-AUTO-013: Coordinator bootstrap.
- TC-AUTO-014: Channel non-authority.
