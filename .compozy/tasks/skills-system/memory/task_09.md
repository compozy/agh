# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 09 `ComposedAssembler` in `internal/daemon` so prompt providers can be chained as `prepend -> agent prompt -> append`, with regression coverage proving memory-only output stays byte-identical to `memory.Assembler.Assemble()`.

## Important Decisions
- Treat the task spec and `_techspec.md` as authoritative for ordering: memory/prepend providers come before the base agent prompt and append providers come after it.
- ADR-003 contains one conflicting sentence about ordering, but the rest of ADR-003, the current `memory.Assembler` behavior, and the task spec all support `memory -> agent -> skills`; implementation will follow that preserved ordering.
- Implement the deliverables under `internal/daemon/` because that is the actual composition-root package in this repository.
- Use `NewComposedAssembler(...)` with `WithPrependPromptProviders(...)` and `WithAppendPromptProviders(...)` option helpers so the struct keeps explicit prepend/append slots while callers can supply variadic `session.PromptProvider` instances.

## Learnings
- `internal/daemon` package coverage missed the task gate on the first run (`79.6%`); adding real edge-case coverage for nil receiver behavior and empty/nil option handling raised the package to `80.1%`.
- The regression guarantee is best verified against the real `memory.Assembler` with actual `MEMORY.md` index files, not a mock provider.

## Files / Surfaces
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/composed_assembler.go`
- `internal/daemon/composed_assembler_test.go`
- `internal/memory/assembler.go`
- `internal/memory/assembler_test.go`
- `internal/skills/catalog.go`
- `internal/session/prompt_provider.go`

## Errors / Corrections
- Accidentally included Markdown files in a `gofmt` command; reran formatting on Go files only.
- Added edge-case tests after the first focused coverage run fell short of the 80% package target.

## Ready for Next Run
- Task 09 is complete. Code was committed as `c6e094b` (`feat: add composed prompt assembler`).
- Task tracking and workflow memory updates are present in the worktree but intentionally left unstaged, following the task rule to keep tracking-only files out of the automatic commit.
