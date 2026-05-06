# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Build the Slice 1 frozen snapshot and prompt assembly layer so session boot can capture stable memory context, recall/provider outputs share one prompt-packaging path, and sub-agents inherit parent memory read-only.

## Important Decisions

- Added `SnapshotService` and `FrozenSnapshot` in `internal/memory` as the service-level model for prompt-safe memory captured at session boot.
- `SnapshotService.InvalidateNextBoot` only increments a generation counter for future captures; it never mutates already captured `FrozenSnapshot.Section` content.
- Snapshot capture loads prompt blocks in deterministic precedence order: global, workspace, agent-global, agent-workspace. `_system` remains excluded because the service relies on existing prompt-index loaders.
- Provider snapshot support is a narrow `SnapshotProvider.SystemPromptBlock` seam. `ErrNotImplemented` falls back to the local store; other provider errors fail closed with context.
- `RenderRecallPromptSection` is now the shared deterministic renderer for `memcontract.Packaged` recall output; `NewRecallAugmenter` delegates to it instead of owning separate prompt formatting logic.
- `Assembler.PromptStartupSection` keeps daemon composition thin by accepting startup/session metadata and delegating capture/rendering to `SnapshotService`.
- Sub-agent inheritance uses `ParentSnapshot` cloning, sets `ControllerMode=read_only`, records `InheritedFrom`, and does not re-resolve private child memory.

## Learnings

- The snapshot freshness warning can be tested deterministically by creating real memory documents from index links and pinning file mtimes with an injected clock.
- Memory package coverage remains close to the floor; snapshot tests need to cover provider fallback, cap/freshness, scope order, inheritance, and reload boundaries to keep `internal/memory` above 80%.
- Prompt-section cap tests must assert behaviorally on removed long content and truncation markers, not on exact full rendered output.

## Files / Surfaces

- `internal/memory/snapshot.go`
- `internal/memory/assembler.go`
- `internal/memory/recall.go`
- `internal/memory/assembler_test.go`
- `internal/daemon/prompt_input_composite_test.go`
- `internal/situation/service.go` (validated consumer surface, no direct change)

## Errors / Corrections

- Initial cap-focused test expected too much of the long repeated body to survive trimming. The assertion was corrected to verify freshness warning, cap trimming, and truncation marker semantics without weakening the production cap.

## Ready for Next Run

- Task 08 passed focused memory, race, coverage, daemon consumer, situation, lint, `git diff --check`, and full `make verify` before tracking updates.
- Next loop iteration should execute `task_09` (Memory Observability and SSE Hygiene).
