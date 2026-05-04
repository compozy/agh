---
status: resolved
file: internal/acp/client.go
line: 250
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiP,comment:PRRC_kwDOR5y4QM67SmDS
---

# Issue 001: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`Does Go's context.WithoutCancel preserve parent deadlines or cancellation, or does it only retain context values?`

💡 Result:

No, Go's context.WithoutCancel does not preserve parent deadlines or cancellation. It returns a derived context that inherits context values from the parent but ignores the parent's cancellation signal, deadline, and error state. Specifically: - It does not propagate parent's cancellation: the returned context is never canceled when the parent is. - Deadline: always returns (time.Time{}, false) — no deadline. - Done: returns nil. - Err: returns nil. It only retains the parent's context values (via Value(key)), allowing access to request-scoped data like those set by middleware, without the cancellation/deadline behavior. This is useful for background tasks after an HTTP response, where you want values but not the request's cancellation. To preserve deadline while ignoring cancellation, a common workaround is to use context.WithDeadline(context.WithoutCancel(parent), parent.Deadline), though this creates a new timer. Official source code confirms: withoutCancelCtx implements Deadline/Done/Err to return no deadline/nil/nil, but delegates Value to parent.

Citations:

- 1: https://go.dev/src/context/context.go?m=text
- 2: https://pkg.go.dev/context@go1.21.4
- 3: https://pkg.go.dev/context

---



</details>

**Remove deadline stripping from process-registry operations.**

`context.WithoutCancel` removes both cancellation and deadlines, causing `Checkpoint` and `Complete` calls to become unbounded. This can block `Stop` before signaling the subprocess, and can indefinitely stall `p.done` closure if record completion hangs. Use the caller context or a derived context with a timeout for these registry operations.

Also applies to: 649–655, 875–876

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 249 - 250, Replace the use of
context.WithoutCancel (e.g. where waitCtx := context.WithoutCancel(ctx) and go
process.waitForExit(waitCtx)) with the original caller context or a derived
context that preserves deadlines (e.g. pass ctx directly or use
context.WithTimeout(ctx, <reasonable-duration>)) so registry operations like
Checkpoint and Complete aren't made unbounded; update the same pattern at the
other occurrences (the other context.WithoutCancel uses noted in the comment)
and ensure Stop and p.done are not blocked by operations that lack deadlines.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Start`, `Stop`, and `waitForExit` use `context.WithoutCancel` directly for process-registry `Checkpoint`/`Complete` operations. That detached context has no cancellation or deadline, so a stalled registry write can block stop-before-signal handling or delay `p.done` closure indefinitely.
- Fix approach: add a bounded process-registry operation context on the ACP driver and use it for process record registration, checkpoint, and completion paths so cleanup can outlive caller cancellation without becoming unbounded.
