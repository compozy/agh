---
status: resolved
file: internal/session/manager_prompt.go
line: 311
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyyr,comment:PRRC_kwDOR5y4QM654Npk
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't fail late/no-op prompt cancels.**

Line 311 still returns `cancelErr`, so a prompt that finishes just before `driver.Cancel` can surface an error even though the process is already done. The same race can also make `processHandle()` nil right after the `IsPrompting()` check, which should be treated as a no-op instead of a failure.



<details>
<summary>Suggested fix</summary>

```diff
 	proc := session.processHandle()
 	if proc == nil {
-		return errors.New("session: agent process is not available")
+		if !session.IsPrompting() {
+			return nil
+		}
+		return errors.New("session: agent process is not available")
 	}
 
 	cancelErr := m.driver.Cancel(ctx, proc)
-	if cancelErr != nil && !isProcessDone(proc) {
-		return fmt.Errorf("session: cancel prompt for %q: %w", target, cancelErr)
+	if cancelErr != nil {
+		if isProcessDone(proc) {
+			return nil
+		}
+		return fmt.Errorf("session: cancel prompt for %q: %w", target, cancelErr)
 	}
-	return cancelErr
+	return nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_prompt.go` around lines 298 - 311, The cancel path
currently can return a spurious error when the agent finishes racing with
Cancel; update the logic in the block around session.IsPrompting(),
session.processHandle(), m.driver.Cancel and isProcessDone so that if
processHandle() returns nil after the IsPrompting check you treat it as a no-op
and return nil, and if m.driver.Cancel returns an error you only propagate it
when isProcessDone(proc) is false; otherwise drop the cancelErr and return nil.
Ensure references to IsPrompting(), processHandle(), m.driver.Cancel and
isProcessDone are used to gate the no-op behavior and to avoid returning
cancelErr for already-finished processes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `CancelPrompt()` currently propagates `cancelErr` unless the process is already known done, and it treats a nil `processHandle()` after the prompting check as a hard failure. That makes a prompt/exit race surface spurious cancel failures even when the turn already finished.
  I will treat a missing process handle after the prompting check as a no-op, and I will swallow cancel errors when the process is already done so late cancels do not fail spuriously.
  Fixed and verified with targeted package tests plus `make verify`.
