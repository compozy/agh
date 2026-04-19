# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is complete: daemon-owned harness context resolution now lives in `internal/daemon/harness_context.go`, and startup plus prompt seams consume the same policy vocabulary.
- Task 02 is complete: startup prompt assembly now uses daemon-owned section descriptors and a selector over resolved harness policy, and `agh-network` startup content is no longer injected through the session-start inline overlay path.
- Task 03 is complete: turn augmentation now runs through a daemon-owned composite that still injects via `session.PromptInputAugmenter`, with descriptor ordering, aggregate budget handling, and explicit criticality semantics.
- Task 04 is complete: daemon-owned synthetic prompt submission now goes through `session.Manager.PromptSynthetic`, persists as `synthetic_reentry` with structured `PromptSyntheticMeta`, and queues FIFO behind active turns without widening the normal user/network prompt APIs.
- Task 05 is complete: transcript replay renders `synthetic_reentry` as daemon-originated system input, hooks classify it with a dedicated synthetic input class, and extension-host prompt replay discovers turn boundaries from synthetic initiating events as well as `user_message`.
- Task 06 is complete: detached harness work now rides on the existing task runtime through a daemon-owned bridge in `internal/daemon/harness_detached_work.go`, with harness-specific metadata persisted on `task.metadata` and `task_runs.metadata_json` for idempotency, recovery, and later wake-targeted reentry.
- Task 07 is complete: daemon-owned reentry now bridges detached terminal task-run events into policy-driven synthetic wake-up or silent/drop outcomes in `internal/daemon/harness_reentry_bridge.go`, reuses `session.Manager.PromptSynthetic`, records `event_summaries` for completion plus emitted/dropped reentry, and recovers idempotently from persisted run metadata plus synthetic session events.
- Task 08 is complete: harness lifecycle observability now writes `harness.context_resolved`, `harness.section_selected`, `harness.augmenter_applied`, and `harness.augmenter_failed` onto the existing global `event_summaries` timeline, while task 07's detached/reentry summaries remain the completion path; startup summaries queue until session creation and current observe/query/http/uds readers all expose the same ordered list.

## Shared Decisions
- Resolve harness behavior from durable session type plus per-turn origin/runtime metadata; any profile-like label is diagnostic output only, not the source of truth.
- Keep policy ownership in `internal/daemon`; `internal/session` consumes daemon-owned startup and prompt seams instead of embedding harness policy logic.
- Treat synthetic turns as valid resolver vocabulary with explicit validation now, and defer dedicated synthetic submission/persistence work to later tasks.
- Keep `session.PromptProvider` unchanged and add startup-aware selection around the existing assembler seam instead of creating a second prompt builder stack.
- Model `agh-network` as a selected append startup section in daemon-owned assembly; daemon boot should not rely on `StartupPromptOverlay` for the default network path anymore.
- Keep the manager-facing prompt-input augmenter seam unchanged for phase one, but centralize warning-only continuation, ordering, and aggregate budget enforcement inside a daemon-owned composite; any augmenter error that reaches `internal/session` is terminal for dispatch.
- Keep ordinary prompt entrypoints (`Prompt`, `PromptNetwork`, `PromptWithOpts`) user/network-compatible and route daemon-owned synthetic reentry through a dedicated manager helper instead of widening the transport-facing prompt API.
- Downstream consumers must treat `synthetic_reentry` as a prompt-turn boundary without reclassifying it as human input: transcript role is `system`, hook input class is `synthetic_reentry`, and extension-host prompt replay must scan both `user_message` and `synthetic_reentry`.
- Mixed-turn transcript tool lifecycle identity must be scoped by turn id plus tool call id; pairing by bare `tool_call_id` is not safe once daemon-inserted turns share a transcript window.
- Detached harness/background runtime ownership stays in `internal/daemon`, but durability stays on the generic task substrate: later tasks must extend `task` / `task_run` metadata and runtime flows instead of introducing a separate background-run store or state machine.
- Detached reentry should stay push-driven from durable task event records: the task service is the authoritative completion source, while the daemon bridge owns policy resolution, observability summaries, queueing, and recovery.
- Harness lifecycle observability must stay on the existing `event_summaries` read model; add new harness-visible decisions by extending the daemon-owned recorder/tests instead of introducing a second observe store or transport-specific event shape.

## Shared Learnings
- Daemon boot tests that exercise real HTTP/UDS server construction need registry doubles that satisfy the full `taskStore` surface, including `ReserveQueuedRun`; otherwise `bootTasks` skips task runtime setup and transport boot fails on a missing task service.
- Any change to the task API contract should be followed by `make codegen` before running the integration gate; `internal/extension` enforces `codegen-check`, so stale `openapi/agh.json` or `web/src/generated/agh-openapi.d.ts` will fail `make test-integration`.
- `event_summaries.session_id` is backed by the global `sessions` index; daemon tests that use fake session managers must still seed matching session rows (and a workspace row for global sessions) before asserting harness observability writes, and startup-time harness summaries need to queue until `OnSessionCreated` because startup section selection runs before that row exists.
- The task/skill QA workflow still references `scripts/discover-project-contract.py`, but this worktree does not contain that script; harness QA currently relies on the repo-defined verification contract (`make verify`, `make test-integration`, and targeted Go integration bundles) until the discovery entrypoint is restored.
- Detached completion scenarios that resolve to a silent/drop outcome should be proven via `harness.detached_run_completed` plus `harness.synthetic_reentry_dropped`; when policy rejects a non-system wake target, no synthetic session event or extra synthetic-context summary is emitted.

## Open Risks
- Follow-on tasks can regress the architecture if they reintroduce session-local policy branches or a foundational `HarnessProfile` enum instead of extending the shared resolver.
- Later startup or augmentation work can regress the new model if it bypasses the section selector and writes policy-conditioned prompt content directly in `internal/session`.
- Later runtime augmenters must register descriptors in the daemon composite whenever the resolver can enable them; an enabled-but-unregistered augmenter now surfaces as an explicit dispatch error instead of being silently skipped.

## Handoffs
- Task 02, task 03, and task 06 should extend the resolver-backed startup, prompt, and detached-runtime seams instead of adding parallel policy checks.
- Task 03 should build ordered turn augmentation with the same pattern used here: daemon-owned selection/composition over an existing session seam, not a parallel runtime path.
- Future prompt augmentation tasks should add new descriptor entries and tests in `internal/daemon/prompt_input_composite.go` rather than branching inside `internal/session` or adding one-off wrappers in harness context code.
- Task 06 and task 07 should reuse the task-04/task-05 synthetic contract instead of inventing another initiating event shape: downstream turn discovery now recognizes `synthetic_reentry`, so later detached-runtime work should emit and consume that boundary consistently.
- Task 07 should consume detached completion events from normal task/run persistence by reading the daemon-owned detached metadata on `task.metadata` and `task_runs.metadata_json`; do not add a separate detached-run catalog.
- Task 08 should build on the bridge-owned `harness.detached_run_completed`, `harness.synthetic_reentry_emitted`, and `harness.synthetic_reentry_dropped` summaries instead of introducing a second completion audit path.
