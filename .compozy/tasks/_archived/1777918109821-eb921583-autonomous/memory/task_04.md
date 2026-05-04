# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build the bounded agent situation context for Task 04 through existing startup prompt and live prompt augmentation seams.
- Final state: implementation, targeted tests, full `make verify`, self-review, tracking updates, and local code commit completed.

## Important Decisions
- Implement the context assembly in a small `internal/situation` package and wire it from `internal/daemon`; this keeps renderers testable while preserving daemon as the composition root.
- Use `session.StartupPromptContext` and `session.PromptInputAugmenter` instead of introducing a new session prompt pipeline.
- Reuse Task 02 `contract.AgentContextPayload` without changing DTOs or generated web artifacts.
- Expose the assembler through `RuntimeDeps.AgentContext` so Task 06 can wire `/agent/context` and CLI verbs without duplicating service lookup logic.

## Learnings
- `session.StartupPromptContext` already carries `SessionID`, so startup situation context can include the durable session identity before the driver starts.
- Task 02 already added the `/agent/context` contract DTOs in `internal/api/contract/agents.go`; Task 04 should assemble those shapes without changing the public DTO unless tests expose a gap.
- Task 03 stores coordination channel correlation in run metadata under `coordination_channel_id`, falling back to `Run.NetworkChannel` when absent.
- The prompt assembler can support startup-aware providers through a narrow optional interface while preserving `session.PromptProvider` for existing providers.
- `make verify` passed after lint fixes for function length, large parameter, cyclomatic complexity, line length, and nil-context test coverage.

## Files / Surfaces
- Implemented: `internal/situation/*` for context assembly/rendering/tests.
- Updated: `internal/daemon/boot.go`, `daemon.go`, `composed_assembler.go`, `prompt_sections.go`, `harness_context.go`, `prompt_input_composite.go`, and related tests/integration tests.
- Updated: `internal/session/manager_start.go` and `internal/session/prompt_overlay.go` so startup prompt context carries provider and timestamps.

## Errors / Corrections
- Initial full verification failed on lint only. Fixed root causes in production/test code and reran full `make verify` successfully.

## Ready for Next Run
- Local code commit: `5d87b3ed feat: add situation surface providers`.
- Task 06 should consume `RuntimeDeps.AgentContext` for endpoint/CLI exposure and keep raw claim tokens out of context output.
