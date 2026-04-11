# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build the shared dispatcher path that schedules, triggers, manual fires, and future extension activations will all use.
- Persist run lifecycle state changes plus retry attempts, enforce global concurrency, and enforce restart-safe fire limits from persisted runs.
- Finish with explicit automation unit/integration coverage, repo verification, task tracking updates, and one local commit.

## Important Decisions
- Added a pure model leaf package at `internal/automation/model` so config/store code can depend on automation types without blocking the runtime dispatcher from importing `session` and `acp`.
- The dispatcher creates automation sessions as `session.SessionTypeSystem`.
- Workspace-scoped dispatch uses `session.CreateOpts.Workspace = workspace_id`; global dispatch uses `session.CreateOpts.WorkspacePath = <AGH home dir>` so the existing session manager contract remains unchanged.
- Retry behavior is implemented as separate persisted run rows per attempt. Each attempt is recorded as `scheduled`, transitions to `running`, then lands in `completed`, `failed`, or `cancelled`.

## Learnings
- The existing package graph (`config -> automation` and `session/acp -> config`) makes a dispatcher in the root automation package impossible without extracting the pure model layer first.
- Fire-limit checks need an in-process critical section around `CountRuns` plus `CreateRun` to avoid same-process double-admission races before the persisted scheduled row exists.
- The task’s lifecycle requirement can be proven against the existing run table by observing the same row while a fake session creator is paused at create/prompt boundaries; no extra transition table was necessary.

## Files / Surfaces
- `internal/automation/doc.go`
- `internal/automation/types.go`
- `internal/automation/persistence.go`
- `internal/automation/validate.go`
- `internal/automation/template.go`
- `internal/automation/model/`
- `internal/automation/dispatch.go`
- `internal/automation/dispatch_test.go`
- `internal/automation/dispatch_integration_test.go`
- `internal/config/automation.go`
- `internal/config/config.go`
- `internal/store/globaldb/global_db_automation.go`

## Errors / Corrections
- Initial dispatcher implementation imported `internal/config` transitively through `session`/`acp`, causing an import cycle. Fixed by extracting the pure model layer into `internal/automation/model` and repointing config/store imports there.
- The first dispatcher version rendered trigger prompts after session creation. Corrected the execution order so prompt rendering failures terminate the scheduled run before any session is started.
- Initial package coverage was 66.3%. Added focused validation, trigger-rendering, cancellation, and helper tests to bring `internal/automation` coverage to 85.7%.

## Ready for Next Run
- Verification evidence is clean:
  - `go test ./internal/automation`
  - `go test ./internal/automation -cover` => `coverage: 85.7% of statements`
  - `go test -tags integration ./internal/automation`
  - `go test ./internal/config ./internal/store/globaldb`
  - `go test -tags integration ./internal/store/globaldb`
  - `make verify`
- Next steps are task tracking updates, final self-review confirmation, and the single local commit for task 03.
