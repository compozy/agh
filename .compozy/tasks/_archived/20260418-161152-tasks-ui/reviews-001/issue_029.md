---
status: resolved
file: internal/task/live.go
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb8,comment:PRRC_kwDOR5y4QM65B8fb
---

# Issue 029: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Don't close `deliver` while `enqueue` can still send.**

`emitTaskLiveEventBestEffort` calls `subscriber.enqueue` after copying subscriber pointers out of the map, while other paths can concurrently run `subscriber.stop()` and close `deliver`. A send on a closed channel will panic here, so a disconnect or overflow can take down the process. Keep the channel open and signal shutdown separately, or guard send/close with the same synchronization primitive.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/live.go` around lines 43 - 55, The code currently closes the
deliver channel in taskStreamSubscriber.stop(), which can race with
taskStreamSubscriber.enqueue() (called from emitTaskLiveEventBestEffort) and
cause a panic; change the shutdown signalling to avoid closing deliver: add a
separate done chan struct{} (or an atomic/locked closed flag) on
taskStreamSubscriber, have stop() close(done) via closeOnce, and remove
close(s.deliver); update enqueue(event StreamEvent) to first select on <-s.done
to return false if closed, otherwise attempt to send to s.deliver (preserving
the non-blocking default path), e.g., select { case <-s.done: return false; case
s.deliver <- event: return true; default: return false }, and ensure any other
code that iterates subscribers uses the done signal rather than relying on
closed deliver.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `taskStreamSubscriber.stop()` closes `deliver` while `emitTaskLiveEventBestEffort()` can still call `enqueue()` on the copied subscriber pointer outside the manager lock. A concurrent send on a closed channel can panic the process.
- Fix approach: stop using channel close as the subscriber shutdown signal. Introduce a separate closed signal/flag so `enqueue()` can reject stopped subscribers without panicking, while the stream goroutine owns the `deliver` reader lifecycle.

## Resolution

- Reworked `taskStreamSubscriber` shutdown in `internal/task/live.go` to use a dedicated `done` channel instead of closing `deliver`.
- `enqueue()` now rejects stopped subscribers without panicking, and the stream loop exits on `done` or context cancellation while leaving `deliver` owned by the subscriber goroutine.
- Verification: `go test ./internal/task` and `go test -tags integration ./internal/task`
