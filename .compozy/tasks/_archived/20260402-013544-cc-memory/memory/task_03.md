# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add prompt-level `MemoryContext` support in `internal/prompt` only.
- Inject a memory section after `CONTEXT:` and before any additional sections.
- Preserve all existing prompt section ordering and zero-change behavior when memory inputs are empty.
- Deliver tests that cover inclusion, omission, partial population, instruction content, and ordering.

## Important Decisions

- Treat the task spec, techspec, and ADR-003/ADR-005 as the approved design baseline; no separate design loop is needed.
- Keep scope out of `internal/kernel/*`; task_04 will populate `MemoryContext`.
- Use one rendered memory block rather than separate top-level sections so ordering stays simple and additive.

## Learnings

- `internal/prompt` already sits at `87.9%` coverage before this change, so the main test need is behavior coverage rather than coverage rescue.
- Existing assembler tests already pin the section order up to `CONTEXT:`; new tests should extend that coverage without rewriting existing expectations.
- `go test ./internal/prompt -race -cover` now reports `89.2%` statement coverage after the memory section tests.
- `make verify` passed cleanly after the prompt changes, so the additive `MemoryContext` field did not break existing callers.

## Files / Surfaces

- `internal/prompt/assembler.go`
- `internal/prompt/context.go`
- `internal/prompt/assembler_test.go`
- `.compozy/tasks/cc-memory/task_03.md`
- `.compozy/tasks/cc-memory/_tasks.md`
- `.compozy/tasks/cc-memory/memory/MEMORY.md`

## Errors / Corrections

- None during implementation or verification.

## Ready for Next Run

- Task implementation is complete. Remaining action for a later task is task_04 wiring in `internal/kernel/session_manager.go` to populate `MemoryContext` from `memdir` indexes and blackboard `type="memory"` entries.
