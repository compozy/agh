# TC-INT-005: Extractor, Inbox, Dreaming, DLQ, And Shutdown Behavior

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify Memory v2 background workers use durable boundaries and are safe under queue pressure, failures, retries, and daemon shutdown.

## Preconditions

- [ ] Isolated daemon has extractor and dreaming enabled.
- [ ] Scenario root session can persist assistant messages.
- [ ] Fixtures can force extractor/dreaming success and failure paths.

## Test Steps

1. **Run focused worker tests**
   - Input: `go test ./internal/memory/extractor ./internal/memory/consolidation ./internal/memory ./internal/daemon -count=1`
   - **Expected:** Queue, coalescing, DLQ, dreaming gate, and daemon lifecycle tests pass.

2. **Trigger root-session extraction**
   - Input: persist assistant messages in a root session.
   - **Expected:** `hook.session.message_persisted` triggers extractor; `_inbox/` receives JSONL; controller applies candidates.

3. **Verify sub-agent skip**
   - Input: persist sub-agent messages.
   - **Expected:** Extractor no-ops or records skipped behavior; sub-agent cannot write directly.

4. **Queue pressure probe**
   - Input: generate more messages than queue capacity and coalesce threshold.
   - **Expected:** Coalescing and bounded drop events are emitted according to config; daemon remains responsive.

5. **DLQ failure and replay**
   - Input: force extractor decode/apply failure, then replay from DLQ.
   - **Expected:** Failure file lands under `_system/extractor/failures`; replay is idempotent and produces expected events.

6. **Dreaming gate and retry**
   - Input: seed recall signals above threshold, trigger dream, then force one failure and retry.
   - **Expected:** Successful run writes `_system/dreaming`, marks signals promoted; failure writes `_system/dream/failures`; retry is idempotent.

7. **Shutdown drain**
   - Input: stop daemon while extractor/dreaming work is in flight.
   - **Expected:** drain respects configured timeout; no goroutine leak or lost durable payload.

## Evidence To Capture

- Go test logs.
- `_inbox/`, extractor failure, dream output, and dream failure paths.
- `memory_events`, `memory_recall_signals`, and `memory_consolidations` rows.
- Daemon shutdown logs.

