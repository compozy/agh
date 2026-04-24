---
status: resolved
file: internal/daemon/restart.go
line: 819
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1X,comment:PRRC_kwDOR5y4QM67HMWg
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Preserve cancellation cause instead of reporting timeout on canceled readiness waits.**

When `waitCtx.Done()` is triggered by cancellation, this path still performs timeout-style drain handling and can persist a timeout-oriented failure reason after an unnecessary grace delay.


<details>
<summary>🔧 Proposed fix</summary>

```diff
         case <-waitCtx.Done():
+            if errors.Is(waitCtx.Err(), context.Canceled) {
+                return h.fail(
+                    store,
+                    operationID,
+                    fmt.Errorf("daemon: replacement daemon readiness canceled: %w", waitCtx.Err()),
+                )
+            }
             if exited, err := waitForProcessExitAfterReadyTimeout(
+                ctx,
                 processErrCh,
                 h.cfg.ExitDrainWait,
             ); exited {
                 if err != nil {
                     return h.fail(
@@
-func waitForProcessExitAfterReadyTimeout(processErrCh <-chan error, grace time.Duration) (bool, error) {
+func waitForProcessExitAfterReadyTimeout(
+    ctx context.Context,
+    processErrCh <-chan error,
+    grace time.Duration,
+) (bool, error) {
     select {
     case err := <-processErrCh:
         return true, err
     default:
     }
@@
     select {
     case err := <-processErrCh:
         return true, err
     case <-timer.C:
         return false, nil
+    case <-ctx.Done():
+        return false, nil
     }
 }
```
</details>

As per coding guidelines, `Use select with ctx.Done() in all long-running goroutine loops` and `context.Context as first argument to functions crossing runtime boundaries`.


Also applies to: 844-863

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/restart.go` around lines 803 - 819, The readiness-wait path
in restart.go is converting context cancellations into timeout-style replacement
failures; update the logic around waitForProcessExitAfterReadyTimeout and the
surrounding readiness wait to detect when waitCtx.Done() is triggered and
preserve/return that cancellation error (or propagate the original
context.Err()) instead of treating it as a timeout and calling h.fail with
errReplacementDaemonExitedBeforeReady; specifically, change the post-wait
handling in the block that calls waitForProcessExitAfterReadyTimeout (and the
similar block around the 844-863 range) to check for context cancellation
(ctx.Err() or the cancellation cause) and return/propagate that immediately, and
ensure any long-running loops use select { case <-waitCtx.Done(): ... } so the
cancellation cause is not lost when invoking h.fail(store, operationID, ...).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `waitForReady` treats `waitCtx.Done()` as a readiness timeout even when the parent context was canceled.
  - The fix is to detect cancellation separately, preserve the cancellation cause, and make the post-timeout process-exit drain context-aware.
