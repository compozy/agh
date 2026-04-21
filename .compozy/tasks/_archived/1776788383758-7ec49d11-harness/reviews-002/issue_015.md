---
status: resolved
file: internal/task/manager.go
line: 2586
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUO,comment:PRRC_kwDOR5y4QM65IlPU
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Protect the best-effort observer from panics.**

`OnTaskEvent` runs inline after the durable write. If the injected observer panics, this method unwinds after the event is already committed and skips the live fanout, which breaks the "best-effort" contract for the hook.  


<details>
<summary>Suggested hardening</summary>

```diff
 	if m.eventObserver != nil {
-		m.eventObserver.OnTaskEvent(postCommitCtx, record)
+		func() {
+			defer func() {
+				if recover() != nil {
+					// Keep observer failures best-effort; log via the existing structured logger if available.
+				}
+			}()
+			m.eventObserver.OnTaskEvent(postCommitCtx, record)
+		}()
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if m.eventObserver != nil {
		func() {
			defer func() {
				if recover() != nil {
					// Keep observer failures best-effort; log via the existing structured logger if available.
				}
			}()
			m.eventObserver.OnTaskEvent(postCommitCtx, record)
		}()
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 2584 - 2586, Wrap the injected
observer call so panics cannot escape: protect the
m.eventObserver.OnTaskEvent(postCommitCtx, record) invocation with a recover()
boundary (e.g., defer func(){ if r := recover(); r != nil { /* log the panic and
continue */ } }()), ensuring the call still occurs inline but any panic is
caught and logged and does not prevent subsequent live fanout; update the call
site that references m.eventObserver, postCommitCtx, and record accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `recordTaskEvent` writes the durable event first, then calls the injected `eventObserver` inline before live fanout.
  - If the observer panics, the panic escapes after the event is already committed and prevents `emitTaskLiveRecordBestEffort` from running.
  - Root cause: the best-effort observer hook is not isolated from the rest of the post-commit notification path.
  - Fix approach: add a recovery boundary around `m.eventObserver.OnTaskEvent(...)` so observer panics do not prevent live fanout, and add regression coverage for the panic path.
  - Resolved by hardening `internal/task/manager.go` and adding regression coverage in `internal/task/manager_test.go`.
