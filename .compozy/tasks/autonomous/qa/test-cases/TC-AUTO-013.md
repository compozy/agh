## TC-AUTO-013: Coordinator Bootstrap And Restricted Orchestration

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 65 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the coordinator is a configurable managed session that starts only for executable
workspace-scoped channel-bound runs, remains singleton per workspace, uses restricted orchestration
permissions, recovers when pending work remains, and relies on public task/channel/spawn APIs.

### Traceability

- Task: task_14, Coordinator Bootstrap And Restricted Orchestration.
- TechSpec: Coordinator Agent, Coordinator Trigger, Scheduler and Claim Authority, Task-Channel Coordination Contract.
- ADR: ADR-004, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, ADR-012.
- Resource lesson: Hermes runner/trajectory references require auditable orchestration evidence; Paperclip orchestration plan requires centralized ownership control.
- Surfaces: `internal/coordinator`, `internal/daemon`, `internal/session`, `internal/task`, `internal/network`, coordinator hooks.

### Preconditions

- Coordinator auto-start enabled for a workspace.
- One user-created task that has not been started.
- One workspace-scoped run that can be started with a stable channel.
- One global-scope run for negative coverage.

### Test Steps

1. Create a task without starting/publishing/approving it.
   - **Expected:** No coordinator session is created and no run/channel exists.

2. Start or publish the task to enqueue a workspace-scoped coordinated run.
   - **Expected:** Run has stable `coordination_channel_id`, and exactly one coordinator session starts or is reused for the workspace.

3. Enqueue multiple runs concurrently in the same workspace.
   - **Expected:** Existing healthy coordinator is reused; no duplicate active coordinator appears.

4. Inspect coordinator permissions and spawn behavior.
   - **Expected:** Coordinator has restricted orchestration-safe permissions and cannot spawn another coordinator.

5. Stop or crash the coordinator while executable work remains pending.
   - **Expected:** Daemon recovery may start a replacement coordinator under the same config/cap/policy conditions.

6. Exercise coordinator/worker operational communication through the run channel and task API.
   - **Expected:** Coordinator/worker exchange `status`, `blocker`, or `result` messages through the channel, but claim/heartbeat/complete still use token-fenced task APIs.

7. Enqueue a global-scope run.
   - **Expected:** No automatic coordinator bootstrap occurs in the MVP.

### Evidence To Capture

- `qa/logs/TC-AUTO-013/task-created-no-coordinator.json`
- `qa/logs/TC-AUTO-013/coordinator-bootstrap.json`
- `qa/logs/TC-AUTO-013/coordinator-singleton.log`
- `qa/logs/TC-AUTO-013/coordinator-permissions.json`
- `qa/logs/TC-AUTO-013/coordinator-recovery.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Disabled config | coordinator enabled false | No auto-start |
| Missing channel | run lacks channel ID | Bootstrap rejects/delays with clear reason |
| Concurrent enqueue | two starts same workspace | One active coordinator |
| Global run | scope global | No auto-spawn |

### Related Test Cases

- TC-AUTO-009: Execution boundary creates the channel-bound run.
- TC-AUTO-017: End-to-end coordinated execution.
