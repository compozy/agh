---
status: resolved
file: internal/hooks/dispatch_async.go
line: 26
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrXs,comment:PRRC_kwDOR5y4QM62twdJ
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Async execution is detached from parent cancellation**

At Lines 24-26, `baseCtx` is rooted in `poolCtx`, so cancellation of `parent` does not stop hook execution unless a timeout is set. This can leave orphaned async work after request/session cancellation. As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via context.Context cancellation".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/hooks/dispatch_async.go` around lines 24 - 26, The async hook's
baseCtx is currently rooted in poolCtx so it ignores parent cancellation; change
the base context to be derived from the parent's context (e.g., use
parent.Context() or the parent context variable) when building baseCtx before
calling h.enterDispatch(asyncHook.Event) so that cancellation of parent cancels
the async hook goroutine; update the lines creating baseCtx and the subsequent
WithValue call (which reference dispatchDepthContextKey{} and
dispatchChainContextKey{} and currentDispatchChain(parent)) to use the parent's
context as the root instead of poolCtx so enterDispatch and any spawned
goroutines inherit parent cancellation.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: Async hook execution currently roots its derived context in the pool context only, so canceling the parent dispatch does not stop the hook unless a hook timeout is configured. That violates the expectation that request/session cancellation propagates into async work.
- Fix approach: Build the async execution context so it is canceled by both the parent dispatch context and the pool lifecycle, then add regression coverage for parent cancellation.
