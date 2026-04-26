## TC-AUTO-009: Execution Boundary And Coordination Channel Binding

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 50 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify task creation records intent only, while publish/start/approval is the idempotent execution
boundary that creates exactly one claimable run and binds workspace-scoped coordinated runs to a
stable coordination channel.

### Traceability

- Task: task_10, Operator Start Publish Approval Execution Boundary.
- TechSpec: Manual Control Contract, Coordinator Trigger, Task-Channel Coordination Contract.
- ADR: ADR-005, ADR-007, ADR-010, ADR-011, ADR-012.
- Resource lesson: Multica create-versus-start mutations and Paperclip execution locks require explicit execution boundaries.
- Surfaces: `internal/task`, `internal/network`, CLI/API task start/publish/approve, task hooks.

### Preconditions

- Isolated workspace with coordinator config available.
- API or CLI paths for task create, start, publish, and approve.
- Ability to inspect runs and channel IDs after each operation.

### Test Steps

1. Create a user task as draft/ready intent.
   - **Expected:** Task exists, no run exists, no `coordination_channel_id` exists, and no coordinator session starts.

2. Publish or start the user task with optional channel override.
   - **Expected:** Exactly one run is enqueued, the workspace run has one stable `coordination_channel_id`, and actor/origin metadata names the operator path.

3. Repeat the same start/publish request with the same idempotency key.
   - **Expected:** Existing run/channel is returned or conflict is documented; no duplicate run or duplicate channel is created.

4. Create an agent task requiring approval, then approve it.
   - **Expected:** Creation has no run; approval enqueues exactly one channel-bound run with agent/user actor provenance.

5. Create or start a global-scope run.
   - **Expected:** Global-scope run does not auto-bind workspace coordinator semantics and does not auto-spawn a coordinator in the MVP.

### Evidence To Capture

- `qa/logs/TC-AUTO-009/task-create.json`
- `qa/logs/TC-AUTO-009/runs-before-start.json`
- `qa/logs/TC-AUTO-009/task-start.json`
- `qa/logs/TC-AUTO-009/idempotent-start.json`
- `qa/logs/TC-AUTO-009/approval-enqueue.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Repeated start | same task and idempotency key | No duplicate run/channel |
| Channel override | `--channel coord-run-123` | Run stores stable requested/resolved channel |
| Agent-created task | approval needed | No run before approval |
| Manual session start | `agh session new` | No task run or coordinator trigger |

### Related Test Cases

- TC-AUTO-013: Coordinator observes this boundary.
- TC-AUTO-015: UI labels this boundary honestly.
