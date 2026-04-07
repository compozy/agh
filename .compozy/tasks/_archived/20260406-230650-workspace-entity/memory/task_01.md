# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the initial `internal/workspace/` package surface for task_01: persisted and resolved workspace models, sentinel errors, and store/resolver interfaces only.

## Important Decisions
- Keep the task limited to `workspace.go` and `store.go` with no resolver implementation, matching ADR-004's package split for the thin API surface required here.
- Do not add JSON tags yet because the TechSpec does not define wire tags for these models; tests will lock the current zero-value struct shape instead.
- Use declarations only in the production package; avoid adding placeholder constructors or behavior solely to manufacture statement coverage.

## Learnings
- `internal/store/global_db.go` currently implements only `SessionRegistry`; there is no existing workspace API to adapt or preserve.
- `internal/session/manager.go` still accepts raw workspace strings, so the pre-change signal for this task is simply the absence of a `workspace` domain package.
- `.compozy/tasks/workspace-entity/` is currently untracked in git; tracking and memory updates should not be included in the auto-commit unless later repository rules require it.
- `go test ./internal/workspace -cover` reports `coverage: [no statements]` because task_01 adds declarations only; error and API-shape tests still protect the exported surface.

## Files / Surfaces
- `internal/store/global_db.go`
- `internal/session/manager.go`
- `internal/config/agent.go`
- `internal/config/config.go`
- `internal/config/home.go`
- `internal/workspace/` (new)
- `internal/workspace/workspace_test.go`

## Errors / Corrections
- Initial skill lookup used the wrong base path; the installed task skills are under `.agents/skills/` inside the repository.

## Ready for Next Run
- Task_01 finished with clean verification: `go test ./internal/workspace -cover` and `make verify` both passed after the new package and tests landed.
