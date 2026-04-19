# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed task 01: added the daemon-owned harness resolver, startup overlay seam, prompt augmenter seam, synthetic turn vocabulary/validation, deterministic matrix tests, integration coverage, and a fresh green `make verify`.

## Important Decisions
- Centralized harness resolution in `internal/daemon/harness_context.go` and wired it from `bootPromptProviders`.
- Kept `internal/session` policy-agnostic by injecting daemon-owned `StartupPromptOverlay` and `PromptInputAugmenter` seams.
- Allowed synthetic turn origin in the resolver vocabulary while rejecting normal synthetic prompt submission until a dedicated path exists.

## Learnings
- The daemon test `recordingRegistry` fake had drifted behind `taskStore`; adding `ReserveQueuedRun` plus a compile-time assertion fixed real-server boot coverage.
- Repo-wide verification also required a semantic HTML fix in `web/src/components/ui/empty.tsx` so existing UI tests recognized the empty-state title and description correctly.

## Files / Surfaces
- `internal/daemon/harness_context.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/harness_context_test.go`
- `internal/daemon/harness_context_integration_test.go`
- `internal/daemon/daemon_test.go`
- `internal/session/interfaces.go`
- `internal/session/manager.go`
- `internal/session/manager_prompt.go`
- `internal/session/manager_start.go`
- `internal/session/manager_test.go`
- `internal/session/prompt_overlay.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/observe/tasks.go`
- `web/src/components/ui/empty.tsx`

## Errors / Corrections
- Initial final-gate failure: `TestBootRemovesStaleSocketAndCleansOrphans` reached real HTTP server construction and failed with `httpapi: task service is required`.
- Root cause was the stale `recordingRegistry` test double missing `ReserveQueuedRun`, which caused `bootTasks` to skip task runtime setup; fixing the double restored the intended boot path.

## Ready for Next Run
- Task 01 is complete and the code changes are committed locally as `8cdc6f8e` (`feat: add harness context resolver foundations`).
- Tracking-only files were updated locally and intentionally left out of the commit.
