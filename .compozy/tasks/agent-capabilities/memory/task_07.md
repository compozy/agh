# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute task 07 as the QA quality gate for agent capabilities using the task 06 plan/cases, prove real loader + join + brief/rich discovery behavior, fix any regressions found, and publish fresh verification evidence under `.compozy/tasks/agent-capabilities/qa/`.

## Important Decisions
- Reuse the task 06 artifact root and execution ordering without renaming files or relocating evidence.
- Treat missing `.compozy/tasks/agent-capabilities/qa/verification-report.md` as the strongest pre-change signal that task 07 remains incomplete.
- Keep the discovered session integration regression strict: fix the stale fixture/import in `internal/session/manager_integration_test.go` instead of weakening the create/resume join assertion.

## Learnings
- Task 06 already split the QA matrix into smoke, targeted, and full lanes and mapped each required seam to a concrete TC ID, so task 07 should execute that matrix rather than redefining scope.
- The worktree is already dirty in unrelated skill/task tracking files; task 07 must avoid reverting or editing those surfaces unless this task explicitly requires it.
- `make verify` still omits the required capability-focused integration lanes, so task 07 must run fresh `-tags integration` session/network commands after the repo gate to prove loader/join/discovery behavior end to end.
- The session integration failure was test drift, not a runtime regression: `manager_integration_test.go` carried an unused `slices` import and expected rich capability fields that its fixture did not declare.
- Fresh coverage evidence on affected packages met the task threshold: `internal/config 82.2%`, `internal/session 80.9%`, `internal/network 81.6%`, `internal/api/core 80.0%`.

## Files / Surfaces
- `.compozy/tasks/agent-capabilities/task_07.md`
- `.compozy/tasks/agent-capabilities/_techspec.md`
- `.compozy/tasks/agent-capabilities/_tasks.md`
- `.compozy/tasks/agent-capabilities/adrs/adr-001.md`
- `.compozy/tasks/agent-capabilities/adrs/adr-002.md`
- `.compozy/tasks/agent-capabilities/adrs/adr-003.md`
- `.compozy/tasks/agent-capabilities/qa/test-plans/agent-capabilities-test-plan.md`
- `.compozy/tasks/agent-capabilities/qa/test-plans/agent-capabilities-regression.md`
- `.compozy/tasks/agent-capabilities/qa/test-cases/TC-INT-001.md` through `TC-INT-013.md`
- `.compozy/tasks/agent-capabilities/qa/test-cases/TC-FUNC-014.md`
- `.compozy/tasks/agent-capabilities/qa/issues/BUG-001.md`
- `.compozy/tasks/agent-capabilities/qa/verification-report.md`
- `internal/session/manager_integration_test.go`

## Errors / Corrections
- Smoke lane failure 1: `go test -tags integration ./internal/session ...` failed to compile due to unused `slices` import in `internal/session/manager_integration_test.go`; corrected by removing the dead import.
- Smoke lane failure 2: the same integration test expected `ContextNeeded` and `ArtifactsExpected` to survive create/resume joins, but its `AgentDef` fixture only declared `id/summary/outcome`; corrected by enriching the fixture and recording `BUG-001`.

## Ready for Next Run
- Task 07 is complete. Local commit: `ba21497c` (`test: validate agent capability flows`).
- Tracking/memory files remain intentionally unstaged in the worktree: `task_07.md`, `_tasks.md`, and `.compozy/tasks/agent-capabilities/memory/`.
