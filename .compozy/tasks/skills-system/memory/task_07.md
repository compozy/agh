# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `session.PromptProvider` and refactor `memory.Assembler` so memory prompt assembly is reusable as a standalone section while preserving current `Assemble()` output for transition compatibility.

## Important Decisions
- ADR-003 ordering is the guardrail: `PromptSection()` must return exactly the memory block that `Assemble()` currently prepends ahead of the base prompt.
- Scope stops at the session interface, memory assembler, and tests; `daemon.ComposedAssembler` is task 09.
- Keep `memory.Assembler.Assemble()` as the compatibility surface by delegating to `PromptSection()` and only appending the trimmed base agent prompt when the memory block is non-empty.

## Learnings
- `skills.CatalogProvider` already exposes `PromptSection(ctx, workspace)` but does not yet compile against a session-level interface because `internal/session/prompt_provider.go` is still missing.
- `memory.Assembler.Assemble()` currently loads both indexes, renders a memory context block, and concatenates `contextBlock + "\n\n" + strings.TrimSpace(agent.Prompt)`.
- `internal/memory` now exposes `PromptSection()` with the same rendered memory block, and `go test -race -cover ./internal/memory` reports 80.5% coverage after the refactor.

## Files / Surfaces
- `internal/session/prompt_provider.go`
- `internal/session/interfaces.go`
- `internal/memory/assembler.go`
- `internal/memory/assembler_test.go`
- `internal/skills/catalog.go`
- `.codex/CONTINUITY-prompt-provider-refactor.md`

## Errors / Corrections
- No blocking errors after implementation. Targeted package tests, `make lint`, and `make verify` all passed cleanly.
- Code changes were committed as `3358429` (`refactor: add memory prompt provider`). Tracking/memory files remain unstaged on purpose because the worktree already contained unrelated tracking edits.

## Ready for Next Run
- Unrelated worktree changes already exist in `.agents/skills/*`, `.gitignore`, and multiple skills-system tracking files; do not revert them while doing task 07.
- Tracking updates for `task_07.md` and the task 07 row in `_tasks.md` still need to stay scoped to task 07, and tracking-only files should remain out of the local commit unless explicitly required.
