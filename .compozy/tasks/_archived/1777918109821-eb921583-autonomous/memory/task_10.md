# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the explicit execution boundary for task_10: task create remains intent-only; task publish/start/approval enqueue one idempotent run and bind workspace coordinated runs to a stable coordination channel.

## Important Decisions
- Root cause from audit: publish/approve are task-only today, there is no task-level start operation, and `/tasks/:id/runs` is the only enqueue surface. Fix should add a shared task execution boundary in the task service rather than an `orchestration_required` creation flag.
- Coordination channel binding belongs at the run reservation/enqueue path, not task creation. Workspace runs without an explicit network channel need a derived stable channel ID plus durable channel metadata.
- Publish/start/approval now use deterministic default idempotency keys (`task.<action>.<task_id>`) and return the existing run on same-origin retries before mutating task state.
- API contracts now expose a `TaskExecutionRequest` body and task-plus-run `TaskExecutionResponse` for publish/start/approval; web adapters send an empty execution request for existing UI publish/approve actions and still return the task record to callers.

## Learnings
- Pre-change signal: `rg` found no `Service.StartTask` or `/api/tasks/:id/start` route.
- Current `ReserveQueuedRun` sets `coordination_channel_id` only from an existing network channel, leaving workspace runs without a channel binding when no task/run channel is supplied.
- Task-run hook payload context currently includes actor fields but not origin fields; task_10 requires origin metadata at the enqueue/start hook boundary.
- A failed blocked publish must not leave a draft task ready after dependencies clear; the operator must retry publish to cross the execution boundary.
- `internal/task` coverage for lifecycle helpers is 80.1%; full `internal/store/globaldb` package coverage remains lower because unrelated scheduler/bridge/migration surfaces dominate that package.

## Files / Surfaces
- Touched: `internal/task`, `internal/store/globaldb`, `internal/hooks`, `internal/api/{contract,core,httpapi,udsapi,spec,testutil}`, `internal/cli`, generated OpenAPI/SDK/web contracts, and `web/src/systems/tasks/adapters`.

## Errors / Corrections
- Corrected legacy HTTP/UDS integration expectations that attempted a second explicit enqueue after publish; publish now returns the queued run.
- Corrected approval integration expectations from conflict-on-repeat to idempotent same-run replay for same-origin retries.
- Corrected web typecheck fallout by sending `{}` as the execution request body for generated publish/approve operations.
- Corrected final lint fallout by renaming task-package execution types to avoid exported-name stutter, passing the CLI execution bundle by pointer, and deduplicating the repeated agent identity fallback error string.

## Verification Evidence
- `make verify` passed after final lint fixes: oxlint reported 0 warnings/errors, golangci-lint reported 0 issues, Go test runner reported `DONE 6193 tests in 53.264s`, and package boundaries were respected.
- Task-specific integration slice passed after final changes: `go test -tags integration ./internal/task ./internal/api/httpapi ./internal/api/udsapi ./internal/cli -run 'TestTaskManager(StartBoundaryCreatesChannelAndClaimableRun|AgentCreatedTaskApprovesThenClaims|ApprovalGateAndAttemptExhaustion|PublishTask)|Test(HTTP|UDS)(FullRoundTripWithRealSessionManager|Task(PublishRunDetailAndLiveRoutesRoundTrip|DashboardInboxApprovalAndTriageRoutesRoundTrip))|TestHTTPTransportTaskSurfaceMatchesDocumentedSpecOperations|TestTask' -count=1`.
- Coverage target evidence passed after final changes: `go test ./internal/task -cover -count=1` reported `coverage: 80.2% of statements`.

## Ready for Next Run
- Implementation, verification, tracking updates, and the local code commit are complete.
- Local commit: `c615111b feat: add task execution boundary`.
- Remaining dirty files are workflow/task tracking documents that were intentionally left out of the code commit, plus pre-existing autonomous task document edits.
