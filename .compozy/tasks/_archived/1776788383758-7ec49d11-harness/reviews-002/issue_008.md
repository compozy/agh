---
status: resolved
file: internal/daemon/harness_reentry_bridge.go
line: 157
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUH,comment:PRRC_kwDOR5y4QM65IlPJ
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Track bridge goroutines during shutdown.**

`newHarnessReentryBridge`/`enqueueWake` start background goroutines (`run`, `drainWakeQueue`, `awaitSyntheticWake`), but `shutdown()` only cancels the context and returns. The daemon can tear down `sessions` or `store` while those goroutines are still calling `PromptSynthetic`, `Events`, and `UpdateTaskRun`, which makes shutdown nondeterministic.

<details>
<summary>💡 Suggested change</summary>

```diff
 type harnessReentryBridge struct {
 	ctx      context.Context
 	cancel   context.CancelFunc
+	workers  sync.WaitGroup
 	resolver *HarnessContextResolver
 	recorder *harnessLifecycleRecorder
 	store    harnessReentryStore
 	sessions harnessReentrySessionManager
 	logger   *slog.Logger
...
-	go bridge.run()
+	bridge.workers.Add(1)
+	go func() {
+		defer bridge.workers.Done()
+		bridge.run()
+	}()
 	return bridge, nil
 }
 
 func (b *harnessReentryBridge) shutdown() {
 	if b == nil || b.cancel == nil {
 		return
 	}
 	b.cancel()
+	b.workers.Wait()
 }
...
 	if shouldStart {
-		go b.drainWakeQueue(item.targetSessionID)
+		b.workers.Add(1)
+		go func(sessionID string) {
+			defer b.workers.Done()
+			b.drainWakeQueue(sessionID)
+		}(item.targetSessionID)
 	}
...
-	go b.awaitSyntheticWake(item, eventsCh)
+	b.workers.Add(1)
+	go func() {
+		defer b.workers.Done()
+		b.awaitSyntheticWake(item, eventsCh)
+	}()
```
</details>


As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via context.Context cancellation" and "No fire-and-forget goroutines — track with sync.WaitGroup or equivalent".


Also applies to: 160-165, 613-719

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_reentry_bridge.go` around lines 142 - 157, The bridge
spawns background goroutines in newHarnessReentryBridge/enqueueWake (run,
drainWakeQueue, awaitSyntheticWake) but shutdown only cancels the context,
letting those goroutines race against sessions/store during teardown; fix by
adding explicit goroutine ownership (e.g., a sync.WaitGroup field on
harnessReentryBridge), incrementing the WaitGroup when
run/drainWakeQueue/awaitSyntheticWake start and deferring Done when they exit,
ensure all goroutines select on bridge.ctx.Done() for prompt termination, and
modify shutdown to call cancel() and then wg.Wait() (and close any channels like
events/rescan only after Wait returns) so no fire-and-forget goroutines remain
during shutdown.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The bridge started `run`, `drainWakeQueue`, and `awaitSyntheticWake` goroutines but `shutdown()` only canceled context and returned immediately. That left live workers able to keep touching the session manager and store during teardown, which violates the repo rule that goroutines must have explicit ownership and shutdown. I added worker tracking, made shutdown wait for spawned bridge goroutines, and stopped wake-queue draining after cancellation. Verified with `go test ./internal/daemon -count=1` and `make verify`.
