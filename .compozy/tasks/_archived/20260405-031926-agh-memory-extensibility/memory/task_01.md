# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented the `internal/memory` store core for task 01: validated memory types/header metadata, frontmatter parsing, scope-aware CRUD, index loading/truncation, index cleanup on delete, staleness helpers, and the required unit tests.

## Important Decisions
- Added `Store.ForWorkspace(workspaceRoot)` to bind workspace scope without mutating the base global store. Future daemon/API code can create per-workspace clones safely.
- `Store.EnsureDirs()` succeeds for the base global-only store and also creates the workspace directory when the store is workspace-bound.
- Introduced exported sentinel `memory.ErrValidation` so later handlers can classify invalid scope/filename/frontmatter errors without parsing strings.

## Learnings
- There was no live frontmatter helper in the v2 tree, so `internal/memory/store.go` owns strict YAML frontmatter parsing with `github.com/goccy/go-yaml`.
- The task PRD directory is currently untracked in git; tracking files were updated locally and should remain unstaged unless the repo later decides to version them.
- Targeted verification passed with `go test ./internal/memory`, `go test -race -cover ./internal/memory`, and `make verify`; package coverage reached 81.0%.

## Files / Surfaces
- `internal/memory/types.go`
- `internal/memory/store.go`
- `internal/memory/staleness.go`
- `internal/memory/store_test.go`
- `.compozy/tasks/agh-memory-extensibility/task_01.md`
- `.compozy/tasks/agh-memory-extensibility/_tasks.md`

## Errors / Corrections
- Initial test draft shared one temp-backed store across parallel subtests in `TestStoreRejectsInvalidInputs`; fixed by giving each subtest its own isolated `testStoreEnv`.

## Ready for Next Run
- Task tracking was updated after clean verification. Next step in the broader PRD is task 02, which should add `config.MemoryDirName`/`HomePaths.MemoryDir` and wire the store-facing session/config seams around this package.
