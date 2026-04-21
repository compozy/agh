# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace the daemon's current one-slot harness prompt augmenter wrapper with an ordered composite that still injects through `session.PromptInputAugmenter`.
- Preserve stored original input while dispatching augmented input for user and network turns, and make critical vs warning-only augmenter failure behavior explicit.

## Important Decisions
- Keep the manager seam unchanged and move the composition logic into a new daemon-owned file instead of extending `session.Manager` right now.
- Treat any augmenter error returned to the manager as fatal; warning-only continuation must be handled inside the daemon composite.
- Model augmentation policy with ordered daemon descriptors that carry name, order, aggregate-budget contribution, budget behavior, and criticality so later augmenters can join the same path without another seam change.

## Learnings
- The current implementation still lives in `internal/daemon/harness_context.go` as `newHarnessPromptInputAugmenter(...)` and only knows about durable memory recall.
- `internal/session/manager_prompt.go` records the prompt input and dispatches `turn.start` before invoking `m.inputAugmenter`, which is the behavior that preserves stored-input semantics today.
- The composite can enforce the task's aggregate budget requirement without redesigning prompt dispatch by trimming or omitting only the added contribution and falling back to the last valid message when an augmenter returns blank output or an over-budget rewrite.

## Files / Surfaces
- `internal/daemon/prompt_input_composite.go`
- `internal/daemon/prompt_input_composite_test.go`
- `internal/daemon/prompt_input_composite_integration_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/harness_context.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/harness_context_test.go`
- `internal/daemon/harness_context_integration_test.go`
- `internal/session/manager_prompt.go`
- `internal/session/manager_test.go`
- `internal/session/manager_hooks_test.go`
- `internal/memory/recall.go`

## Errors / Corrections
- Full `make verify` passed after the composite landed; no follow-up code corrections were needed after the final verification pass.

## Ready for Next Run
- Task complete. Later augmentation work should extend the daemon descriptor set and resolver policy instead of adding another manager hook or special-casing durable memory.
