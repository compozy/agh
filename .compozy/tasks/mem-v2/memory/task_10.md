# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implemented the Slice 1 post-response extraction substrate:
  - Typed async hook `session.message_persisted` in `internal/hooks` with payload cloning, matching, introspection, no-hook smoke coverage, and daemon bridge forwarding.
  - Session durable boundary dispatch after `SessionDB.RecordPersisted` assigns the committed transcript sequence; transient deltas do not dispatch the persisted-message hook.
  - `internal/memory/extractor` runtime with one in-flight plus one queued request per session, bounded queue coalescing, hard-cap drop telemetry, graceful `Drain`/`Close`, inbox production, FIFO consumption, controller proposal handoff, and DLQ under `_system/extractor/failures`.
  - `memory.Store.ProposeCandidate` and `Store.RecordExtractorEvent` so extractor outputs enter the controller/WAL path and telemetry stays in canonical `memory_events`.

## Important Decisions

- The hook taxonomy stores the event as `session.message_persisted` (repo taxonomy omits the `hook.` prefix) and marks it async-only.
- Session recording now uses optional `RecordPersisted(ctx, event)` on recorders; existing `EventRecorder` implementations keep working through the fallback `Record` path, while `SessionDB` returns committed `id`/`sequence`.
- Extractor runtime skips root-to-subagent pollution by ignoring persisted-message payloads with `parent_session_id` or `actor_kind=agent_subagent`; root sessions enqueue post-message turns.
- Inbox files are daemon-owned JSONL under `<memory-root>/_inbox/<session_id>/`; failed decode/controller handoffs move the processing file into `<memory-root>/_system/extractor/failures/`.
- Consumers call `ProposalSink.ProposeCandidate`; no extractor code writes memory files directly.
- Queue semantics match ADR-010: one in-flight request plus one queued request per session, queued ranges merge, and the oldest queued range is dropped after the configured coalescing cap.

## Learnings

- `internal/session` has timing-sensitive async hook tests that can fail under heavy parallel package load; targeted reruns passed without code changes after a full affected-package run timed out in existing streaming-hook tests.
- Package coverage for `internal/memory` remains exactly at the 80% floor after adding tests for `ProposeCandidate` and `RecordExtractorEvent`; future memory slices should add behavior coverage with every new branch.

## Files / Surfaces

- Hook taxonomy/dispatch: `internal/hooks/*`, `internal/session/hooks.go`, `internal/session/manager_hooks.go`, `internal/session/manager_prompt.go`, `internal/daemon/hooks_bridge.go`.
- Durable recorder seam: `internal/store/sessiondb/session_db.go`, `internal/store/sessiondb/session_db_extra_test.go`.
- Extractor runtime/inbox: `internal/memory/extractor/events.go`, `internal/memory/extractor/inbox.go`, `internal/memory/extractor/runtime.go`, `internal/memory/extractor/runtime_test.go`.
- Controller/event seams: `internal/memory/decision.go`, `internal/memory/extractor_events.go`, `internal/memory/extractor_events_test.go`.

## Errors / Corrections

- `make lint` initially failed on `WithCoalesceMax(max int)` because `max` redefined a built-in; renamed the parameter to `limit`.
- Staticcheck also rejected direct nil context literals in tests; replaced with a `missingContext()` helper so nil-context validation remains covered without linter conflicts.
- An affected-package run failed once in `TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail` and once in `TestMessageDeltaAsyncHooksDoNotBlockPromptStreaming`; both passed on targeted/full `internal/session` reruns, indicating timing flake under package-load pressure rather than a persisted-message regression.

## Ready for Next Run

- `task_11` can consume controller-applied extracted memories and `memory_events` extractor telemetry; it should not add a parallel extractor log.
- `task_14`/`task_16` can expose inbox/DLQ state through public contracts using `_system/` as the failure authority.
- `task_19` should instantiate and own the extractor runtime in daemon composition, reusing `Runtime.Drain`/`Runtime.Close` for shutdown and wiring the real extractor implementation to `session.message_persisted`.
