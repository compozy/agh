# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 06 inbound delivery workers and turn-end handoff: bounded per-session inbound queues, immediate-vs-deferred delivery, safe wrapper rendering, and required unit/integration tests.

## Important Decisions
- Build delivery as standalone `internal/network` coordination that future `manager.go` can own, instead of expanding task 06 into full daemon boot/runtime wiring.
- Add only minimal session-facing seams required by the delivery layer: direct turn-end notification and busy-state inspection.
- Fire the session turn-end notifier from `pumpPrompt()` after prompt cleanup, not from `dispatchTurnEnd()`, so the delivery worker does not observe a stale busy state.

## Learnings
- `router.Receive()` already returns `[]Delivery`, so the delivery layer can consume routed messages without changing router ownership.
- `session.Manager.PromptNetwork()` already exists and carries `TurnSourceNetwork`; task 06 mainly needs lifecycle/queue orchestration around it.
- Current session runtime has no exported turn-end callback registration or busy-state query, so those surfaces must be added for the delivery layer to behave correctly.
- A queue-first trigger model is simpler than a special-case fast path: enqueue every inbound delivery, then let the per-session worker decide whether to deliver immediately or wait for the next turn-end wakeup.
- The safe wrapper can preserve machine-readable fidelity by base64-encoding canonical JSON for the full body while exposing only a narrow XML-escaped preview string to the model.

## Files / Surfaces
- `.codex/ledger/2026-04-11-MEMORY-inbound-delivery.md`
- `internal/session/interfaces.go`
- `internal/session/manager.go`
- `internal/session/manager_prompt.go`
- `internal/session/session.go`
- `internal/network/delivery_integration_test.go`
- `internal/network/router.go`
- `internal/network/delivery.go`
- `internal/network/delivery_test.go`

## Errors / Corrections
- Initial turn-end callback placement in `dispatchTurnEnd()` was corrected because it fired before `currentTurnSource` cleanup; workers now wake from `pumpPrompt()` after teardown.
- Integration helper naming was corrected to `waitForDeliveryCondition()` to avoid colliding with an existing `waitForCondition()` helper under `-tags integration`.
- Test coverage guardrails surfaced `staticcheck` `SA1012` on literal nil contexts; tests now use a `nilContext()` helper to keep the nil-input branches covered without lint failures.

## Ready for Next Run
- Delivery worker code and tests are complete and verified. The next task should compose the delivery coordinator into `internal/network/manager.go` / daemon boot wiring and keep routing, queue semantics, and turn-end wakeups delegated to the existing task 06 surfaces.
