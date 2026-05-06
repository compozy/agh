---
provider: manual
pr:
round: 1
round_created_at: 2026-05-06T02:21:18Z
status: resolved
file: internal/memory/recall/recall.go
line: 530
severity: high
author: claude-code
provider_ref:
---

# Issue 001: Bounded SignalRecorder queue (B2/NB2) not implemented

## Review Comment

The TechSpec is explicit that recall-signal updates must run through a per-workspace bounded channel with overflow telemetry (peer-review B2/NB2):

> "After `Recaller.Recall()` returns a non-empty `Packaged`, surfaced chunks are enqueued onto a per-workspace bounded channel (`SignalRecorder`) — capacity `[memory.recall.signals] queue_capacity` (default 256). A worker goroutine drains the queue and updates rows … Queue overflow drops the oldest entry, increments `memory_recall_signal_updates_total{status="dropped"}`, and emits a canonical event `memory.recall.signal_dropped` to `memory_events`. … Recall surface latency is preserved (failures do not bubble to the caller)."
> (`.compozy/tasks/mem-v2/_techspec.md` §`memory_recall_signals` write path)

The current implementation does the opposite. `Recaller.Recall` calls `r.recordSignals(ctx, query, signalsForRanked(...))` synchronously at `internal/memory/recall/recall.go:191`, and `recordSignals` calls the source directly at line 530:

```go
if err := r.source.RecordRecall(ctx, signals); err != nil {
    r.warn("memory recall: record recall signal failed", "error", err)
    if eventErr := r.source.RecordRecallSignalFailed(ctx, query, err); eventErr != nil { ... }
}
```

`Store.RecordRecall` (`internal/memory/recall_source.go:284`) opens a `withCatalogWriteTx` with a `BEGIN IMMEDIATE` transaction, runs `INSERT … ON CONFLICT DO UPDATE` per signal, and waits for commit before returning. Two consequences:

1. **Recall surface latency now includes catalog disk I/O.** Every Recall call blocks on a write transaction against the workspace `agh.db`. Under contention this competes with controller decision writes (`memory_decisions`) and extractor consumer writes, exactly the contention the bounded queue was supposed to absorb.
2. **The `memory.recall.signal_dropped` event is dead code.** `memoryEventRecallSignalDropped = "memory.recall.signal_dropped"` is declared at `internal/memory/catalog.go:43` and listed in the allowlist at line 138, but no caller ever emits it. The schema CHECK enum (`internal/store/globaldb/global_db.go:1106`) and config keys (`[memory.recall.signals] queue_capacity`, `worker_retry_max`, `metrics_enabled`) all promise observability that never lights up. Operators who alert on `dropped + failed / ok > 0.01` (per Monitoring section) will never see drops, even when the workspace is saturated.

Suggested fix:

- Introduce a daemon-owned `SignalRecorder` worker keyed by `workspaceID` with a bounded channel sized from `[memory.recall.signals] queue_capacity` (default 256), drained by a single goroutine that calls `Store.RecordRecall` in batches.
- On overflow, drop the oldest queued signal, increment a `dropped` metric, and call `Store.RecordRecallSignalDropped` (new method) to write a `memory.recall.signal_dropped` event row.
- In `Recaller.recordSignals`, replace the synchronous `r.source.RecordRecall(...)` call with `recorder.Submit(workspaceID, signals)` (non-blocking; logs and counts on overflow).
- Surface the queue depth via `memory_recall_signal_queue_depth{workspace_id}` (Monitoring §Metrics promises this gauge but it is also missing).
- Add tests `TestSignalRecorder_QueueOverflowDropsOldest`, `TestSignalRecorder_FailureEmitsEventAndMetric`, `TestSignalRecorder_SuccessIncrementsOkMetric` (already enumerated in §Test Plan but currently absent — the only related test is `TestStore.RecordRecallSignalFailed` in `internal/memory/recall_test.go:271`).

## Triage

- Decision: `VALID`
- Root cause: recall signal persistence was still coupled to the recall request path. `Recaller.recordSignals` synchronously called the catalog-backed `Source.RecordRecall`, so disk writes and write-lock contention could affect recall latency and the configured queue/overflow observability was unreachable.
- Fix approach: add a per-workspace bounded `SignalRecorder` owned by `memory.Store`, submit recall signals non-blockingly from `Recaller`, emit `memory.recall.signal_dropped` on overflow, close recorders during daemon shutdown, and cover async success, retry failure, and overflow behavior with focused tests.

## Resolution

- Implemented a bounded asynchronous `SignalRecorder` with per-workspace registry ownership in `memory.Store`, non-blocking recall submission, oldest-drop overflow behavior, failure/dropped event emission, stats, and daemon shutdown cleanup.
- Added focused recorder tests for async success, retry failure telemetry, and oldest-drop overflow.
- Verification: `go test ./internal/memory/recall ./internal/memory ./internal/memory/extractor -count=1` passed; `go test -race ./internal/memory/recall ./internal/memory ./internal/memory/extractor ./internal/api/core ./internal/daemon ./internal/tools ./internal/cli -count=1` passed; `make verify` passed with Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, and boundaries OK.
