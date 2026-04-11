---
status: resolved
file: internal/session/stop_reason.go
line: 96
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZq,comment:PRRC_kwDOR5y4QM623eZ8
---

# Issue 029: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Avoid waiting for `proc.Done()` after a failed driver stop.**

When `m.driver.Stop(ctx, proc)` fails and the process is still running, this can block until `ctx.Done()` and hang stop flows for long-lived contexts.

<details>
<summary>💡 Proposed fix</summary>

```diff
+import "fmt"
...
 	stopErr := m.driver.Stop(ctx, proc)
-	if !isProcessDone(proc) {
+	if stopErr == nil && !isProcessDone(proc) {
 		select {
 		case <-proc.Done():
 		case <-ctx.Done():
-			return errors.Join(stopErr, ctx.Err())
+			return fmt.Errorf("wait for process stop completion: %w", ctx.Err())
 		}
 	}
+	if stopErr != nil && !isProcessDone(proc) {
+		return fmt.Errorf("stop session process: %w", stopErr)
+	}
 
 	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
```
</details>

As per coding guidelines, `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	stopErr := m.driver.Stop(ctx, proc)
	if stopErr == nil && !isProcessDone(proc) {
		select {
		case <-proc.Done():
		case <-ctx.Done():
			return fmt.Errorf("wait for process stop completion: %w", ctx.Err())
		}
	}
	if stopErr != nil && !isProcessDone(proc) {
		return fmt.Errorf("stop session process: %w", stopErr)
	}

	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/stop_reason.go` around lines 89 - 96, The code currently
calls m.driver.Stop(ctx, proc) and then waits on proc.Done() even when stopErr
!= nil, which can block; change the control flow so that if stopErr is non-nil
you return it immediately wrapped with context (e.g., fmt.Errorf("driver stop:
%w", stopErr)), and only enter the select/wait for proc.Done() when stopErr ==
nil and isProcessDone(proc) is false; if the wait returns due to ctx.Done(),
wrap and return ctx.Err() with fmt.Errorf("context: %w", ctx.Err()). Ensure you
update the logic around stopErr, m.driver.Stop, isProcessDone, and proc.Done
accordingly.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `StopWithCause` currently waits on `proc.Done()` whenever the process is still running, even if `m.driver.Stop()` already returned an error. That can block the stop path until `ctx.Done()` for long-lived contexts. The fix is to wait only after a successful driver stop; if stopping fails and the process is still alive, return the wrapped stop error immediately.
  Resolved by changing `internal/session/stop_reason.go` to skip the `proc.Done()` wait when `driver.Stop` fails and the process is still running, plus a regression test in `internal/session/manager_hooks_test.go`. Verified with targeted race coverage and a clean `make verify`.
