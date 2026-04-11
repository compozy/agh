# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the daemon-side delivery broker and session-output projection for channel adapters, including ordered per-route delivery, bounded/coalescing queues, ack tracking, resumable snapshots, and required unit/integration coverage.

## Important Decisions
- Treat the task PRD, techspec, and ADRs as the approved design for this implementation task instead of opening a separate design loop.
- Prefer an in-memory active-delivery projection keyed by the prompt/session/route context rather than expanding globaldb with a new route-by-session lookup for task 06.
- Keep delivery transport separate from hooks and observability by adding a narrow daemon->extension delivery caller instead of routing channel delivery through hook dispatch.
- Keep `internal/channels` ACP-agnostic and place the session-event projection seam in `internal/extension.ChannelDeliveryNotifier` to avoid a `channels -> acp` import cycle.
- Preserve fast prompt output by seeding prompt registration from persisted turn events immediately after `submitPrompt` returns the created `turn_id`.

## Learnings
- `internal/extension.Manager` already negotiates `channels/deliver` during initialize, but it has no runtime caller for that service yet.
- `internal/session.Manager` already emits normalized prompt events through `Notifier.OnAgentEvent`, and `HostAPIHandler.submitPrompt` can recover the created `turn_id` plus post-prompt persisted events via `sessions.Events`.
- The current channel persistence surface resolves routes by routing key and instance, not by `session_id`, so active delivery recovery is cheapest if the broker owns the live session/turn projection state in memory.
- Queue-pressure handling can safely coalesce or replace intermediate deltas, but terminal `final` and `error` events must remain deliverable even when they supersede an earlier queued delta.
- Adding the delivery request/ack/snapshot contract to the extension SDK codegen updates both `openapi/agh.json` and `sdk/typescript/src/generated/contracts.ts`; `make verify` will fail until codegen is refreshed.

## Files / Surfaces
- `internal/channels/`
- `internal/extension/manager.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_channels.go`
- `internal/session/interfaces.go`
- `internal/session/manager_prompt.go`
- `internal/store/globaldb/global_db_channel.go`
- `internal/extension/channel_delivery_notifier.go`
- `internal/extension/channel_delivery_integration_test.go`
- `internal/transcript/transcript.go`
- `sdk/typescript/src/generated/contracts.ts`
- `openapi/agh.json`

## Errors / Corrections
- `make verify` initially failed because the generated TypeScript contracts were stale after adding the delivery SDK types; running `make codegen` refreshed the generated artifacts and unblocked verification.

## Ready for Next Run
- Task 06 implementation, required coverage, integration tests, and `make verify` are complete; next follow-on work is task 07 daemon composition and lifecycle wiring against the new broker/runtime seams.
