## TC-AUTO-015: Tasks UI Manual-First Labels And Coordinator Handoff

**Priority:** P1 (High)
**Type:** UI
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the existing Tasks UI accurately distinguishes saved intent from executable runs, labels
start/publish/approval as coordinator handoff boundaries, shows channel availability without status
authority, and preserves manual session workflows.

### Traceability

- Task: task_15, Tasks UI Manual First Labels And E2E.
- TechSpec: Web and Docs Tests, Manual Control Contract, Impact Analysis Operator Tasks UI.
- ADR: ADR-010, ADR-011, ADR-012.
- Resource lesson: Multica E2E reference uses test-owned fixtures, visible assertions, and cleanup for issue/task flows.
- Surfaces: `web/src/systems/tasks`, `web/e2e/tasks-coordinator-handoff.spec.ts`, generated API types, MSW/runtime fixtures.

### Preconditions

- Web dev server running against isolated daemon or runtime fixture.
- Test fixtures for draft user task, approval-needed agent task, queued/running run with channel, and manual session creation.
- Browser evidence path available under `qa/screenshots/TC-AUTO-015/`.

### Test Steps

1. Create a task in the UI and save it as draft/saved intent.
   - **Expected:** Detail lifecycle reads as saved intent, runs panel says no run is queued, and no channel chip is visible.

2. Inspect start/publish action labels and tooltips.
   - **Expected:** Copy describes publish/start/approval as making work executable and eligible for coordinator handoff.

3. Publish or start the task.
   - **Expected:** UI shows coordinator handoff or queued/running state and displays the channel chip only after run enqueue.

4. Approve an agent-created task from the inbox.
   - **Expected:** Approval is the handoff boundary; creation alone did not queue a run.

5. Start a manual session from the session UI.
   - **Expected:** Manual session start works and does not display task autonomy labels or trigger task orchestration.

6. Run relevant web gates.
   - **Expected:** Web lint, typecheck, unit tests, and Playwright task coordinator spec pass for changed surfaces.

### Evidence To Capture

- `qa/logs/TC-AUTO-015/web-lint.log`
- `qa/logs/TC-AUTO-015/web-typecheck.log`
- `qa/logs/TC-AUTO-015/web-test.log`
- `qa/logs/TC-AUTO-015/playwright-tasks-coordinator-handoff.log`
- `qa/screenshots/TC-AUTO-015/saved-intent.png`
- `qa/screenshots/TC-AUTO-015/coordinator-handoff-channel.png`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Draft with no runs | new task | Runs panel shows saved intent only |
| Queued run with channel | started task | Channel chip visible with non-authority tooltip |
| Agent approval | approval-needed task | Approval enqueues run |
| Manual session | session new | No task run created |

### Related Test Cases

- TC-AUTO-009: Runtime execution boundary.
- TC-AUTO-014: Channel non-authority.
