# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 13 safe spawn: agent-facing spawn API/CLI, daemon/session spawn service, permission narrowing, TTL/depth/child caps, workspace bounds, parent-stop/orphan/TTL reaper, lease release, hooks, tests, verification, tracking updates, and one local commit.
- Baseline gap: contract/spec already define `/api/agent/spawn` DTO/path scaffolding, but there is no `SpawnOpts`, UDS route, CLI command, daemon/session spawn path, or reaper behavior.
- Current state: implementation, tests, task tracking, final verification, and local code/test commit are complete.

## Important Decisions
- Build on Task 12 lineage (`session.CreateOpts.Lineage`, `store.SessionLineage`, globaldb lineage filters) instead of adding a parallel metadata path.
- Enforce spawn safety in daemon/session code after hook patches; prompts or hook behavior are not trusted for permission safety.
- Reaper must release active child task-run leases through task service APIs before stopping reaped child sessions.
- API/CLI spawn requests inherit the caller parent workspace/channel; cross-workspace spawn is rejected at the session boundary instead of exposed as a request override.
- `auto_stop_on_parent` should default to true at ingress, while explicit false remains representable for future bounded policies.

## Learnings
- Shared memory says Task 12 completed typed lineage columns/read models but deliberately left spawn API/reaper behavior for Task 13.
- ADR-006 and TechSpec require default max spawn depth 1, default max children per parent 5, mandatory TTL, permission subset checks over known atom categories, and fail-closed unknown child atoms.
- ADR-009 allows spawn hooks but forbids hooks from bypassing TTL, lineage, caps, or permission narrowing.
- ADR-010 requires manual operator session creation to remain first-class and not subject to child-only caps.
- Existing contract/spec already include `AgentSpawnRequest`, `AgentSpawnResponse`, and `/api/agent/spawn`; the task is runtime/transport wiring and validation, not a fresh DTO design.
- `session.Create` rejects requests that provide both workspace ID and workspace path; the spawn path now inherits the parent workspace by passing a single workspace ID when available, falling back to path only if needed.

## Files / Surfaces
- Draft implementation surfaces now include `internal/session/spawn.go`, session manager create/start hook support, daemon spawn reaper wiring, task service session-lease release API, `/api/agent/spawn` UDS route, UDS client method, and top-level `agh spawn` CLI command.
- Draft tests now cover comparator exact/subset/superset/unknown atom behavior, spawn validation/caps/hook revalidation, strict handler validation, CLI request mapping, reaper TTL/parent/orphan cleanup with release-before-stop ordering, and structural task lease release.
- Final touched implementation surfaces: `internal/session`, `internal/task`, `internal/daemon`, `internal/api/core`, `internal/api/udsapi`, and `internal/cli`.
- Verification evidence:
  - `go test ./internal/session -run TestMessageDeltaAsyncHooksDoNotBlockPromptStreaming -count=20` passed after adding failure-safe cleanup for that pre-existing async-hook test.
  - `go test ./internal/session ./internal/task ./internal/daemon ./internal/api/core ./internal/api/udsapi ./internal/cli -count=1` passed.
  - `make verify` passed with Go lint `0 issues.`, Go runner `DONE 6257 tests in 62.040s`, and package boundaries OK.
  - `go test -cover ./internal/session ./internal/task ./internal/daemon ./internal/api/core ./internal/api/udsapi ./internal/cli -count=1` passed; Task 13's spawn policy home package `internal/session` reported 80.1% and `internal/task` reported 80.1%.
- Commit: `b28cc047 feat: add safe spawn API and reaper` (code/tests only; tracking/memory files intentionally left unstaged).

## Errors / Corrections
- Corrected spawn creation to pass one inherited workspace reference into `Create`; passing both workspace ID and path triggered existing `session.pre_create` validation.
- Corrected lint fallout from the new CLI/reaper files and added failure-safe release cleanup in `TestMessageDeltaAsyncHooksDoNotBlockPromptStreaming` after the focused package run exposed a cleanup hang on assertion failure.

## Ready for Next Run
- Task 13 is complete; remaining unstaged `.compozy/tasks/autonomous/*` changes are tracking/docs/memory work or pre-existing workflow edits, not code left for this task.
