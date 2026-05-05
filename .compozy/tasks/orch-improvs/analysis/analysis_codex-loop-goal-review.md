# Analysis: Codex Loop Goal Review and Codex Review Threads

## Purpose

This note captures the research used to add the review-gate child spec to `orch-improvs`. It should be treated as background evidence, not as an implementation contract. The normative contracts are:

- [`../_techspec.md`](../_techspec.md)
- [`../_techspec_review_gate.md`](../_techspec_review_gate.md)
- [`../adrs/adr-007-review-gate-post-terminal-continuation-loop.md`](../adrs/adr-007-review-gate-post-terminal-continuation-loop.md)
- [`../adrs/adr-008-review-routing-uses-channels-without-channel-authority.md`](../adrs/adr-008-review-routing-uses-channels-without-channel-authority.md)
- [`../adrs/adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md`](../adrs/adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md)

## Codex Loop Goal Mechanism

Relevant files:

- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/activation.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/hooks.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/goal_confirm.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/store.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/config.go`

Observed behavior:

- `goal` is an activation mode alongside time/round modes.
- On Stop, the hook runs a goal check instead of blindly ending the loop.
- The reviewer receives the goal text, task prompt, latest assistant message, loop state, and continuation history.
- The reviewer produces prose, then a structured interpreter maps the prose into a strict verdict schema.
- The strict verdict includes `completed`, `confidence`, `reason`, `missing_work`, and `next_round_guidance`.
- Outcomes include completed, incomplete, error, timeout, and invalid output.
- Only completed ends the loop. Timeout, invalid output, and reviewer errors continue conservatively.
- Loop state is persisted in local JSON/JSONL files and merged into the next continuation prompt.
- Rapid stop guardrails detect shallow repeated stops and cut short after configured thresholds.

AGH translation:

- Copy the typed verdict shape, missing-work guidance, conservative failure handling, persistence, and rapid-terminal guardrails.
- Do not copy shell-configured reviewer commands, local JSON loop store, prompt header activation, Codex Stop-hook authority, or unbounded loops.
- AGH review state belongs in `task.Service` and global DB migrations.
- AGH continuation guidance belongs in `TaskContextBundle`, not only in prompt text.

## Codex App-Server Review/Goal Types

Relevant files:

- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ThreadGoal.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ThreadGoalStatus.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ThreadGoalUpdatedNotification.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ReviewStartParams.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ReviewTarget.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ReviewDelivery.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ReviewStartResponse.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ThreadSourceKind.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/NonSteerableTurnKind.ts`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ApprovalsReviewer.ts`

Observed shape:

- Thread goals have objective, status, optional token budget, tokens used, time used, and timestamps.
- Goal status is active/paused/budget-limited/complete.
- Review starts against a thread and target, with inline or detached delivery.
- Thread source kinds include `subAgentReview`.
- Non-steerable turn kinds include `review`.
- Approval reviewers include user, auto review, and guardian subagent.

AGH translation:

- Treat review as a distinct runtime mode/session role, not as normal worker execution.
- Make review request and review verdict typed protocol surfaces.
- Keep reviewer session routing separate from verdict authority.
- Do not make Codex thread goal state a direct AGH task model.

## Existing AGH Hooks

Relevant AGH observations:

- `internal/task/lease.go` already includes `review_request` in default coordination message kinds.
- Existing task statuses and run statuses do not include `pending_review`.
- `task.Service` already owns run terminalization and task status projection.
- `task_events.event_seq` is the durable replay primitive for task stream history.

AGH implication:

- Channels can already carry review coordination vocabulary.
- Review gate v1 should avoid adding `pending_review`.
- Review request/verdict state should be task-owned and persisted separately.
- Task events should expose review lifecycle after persistence.
