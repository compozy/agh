## TC-AUTO-014: Coordination Channels Are Conversation Only

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify task-run coordination channels support operational conversation and correlation metadata, but
cannot claim, heartbeat, release, complete, fail, or otherwise mutate task-run ownership/status.

### Traceability

- Task: task_06, task_08, task_10, task_14.
- TechSpec: Task-Channel Coordination Contract, Data Flow, Manual Control Contract.
- ADR: ADR-007 and ADR-012.
- Resource lesson: Multica inbox update references treat messages as notification/read-model changes, not source-of-truth ownership state.
- Surfaces: `internal/network`, `internal/task`, `agh ch`, `agh task`, task read models, channel read models.

### Preconditions

- One workspace-scoped coordinated run with stable `coordination_channel_id`.
- One claimed active run with current raw token held by a managed session.
- Ability to read run status/owner before and after channel messages.

### Test Steps

1. Capture run owner, status, lease deadline, and terminal fields before channel messages.
   - **Expected:** Baseline shows the current task service state.

2. Send `agh ch send --kind status` with task/run/channel/correlation metadata.
   - **Expected:** Message is accepted and persisted; run owner/status/lease/terminal fields remain unchanged.

3. Send `agh ch send --kind result` with plausible result body.
   - **Expected:** Message is accepted as conversation only; run is not completed and result is not written to task-run terminal state.

4. Attempt channel send/reply with raw `claim_token` in metadata extension and body.
   - **Expected:** Request is rejected or sanitized according to implementation policy; token is not persisted or logged.

5. Complete the run through `agh task complete` with the current token.
   - **Expected:** Only the task API changes terminal status; previous channel messages remain audit/context artifacts.

6. Inspect web/docs-visible read models if applicable.
   - **Expected:** UI/docs wording does not imply channel messages own task status.

### Evidence To Capture

- `qa/logs/TC-AUTO-014/run-before-channel.json`
- `qa/logs/TC-AUTO-014/channel-status-message.json`
- `qa/logs/TC-AUTO-014/run-after-status-message.json`
- `qa/logs/TC-AUTO-014/channel-token-rejection.log`
- `qa/logs/TC-AUTO-014/task-complete-authority.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| `review_request` | review message | No status mutation |
| `handoff` | handoff message | No ownership mutation |
| Missing correlation | no task/run metadata | Rejected or clearly non-task-bound |
| Message replay | read old message | Does not replay task transitions |

### Related Test Cases

- TC-AUTO-005: Channel command behavior.
- TC-AUTO-016: Docs explain channel/task boundary.
