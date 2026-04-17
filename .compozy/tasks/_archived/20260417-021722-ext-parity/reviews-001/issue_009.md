---
status: resolved
file: internal/acp/launcher.go
line: 73
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQF,comment:PRRC_kwDOR5y4QM64dqGM
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Honor the caller context when spawning the subprocess.**

`Launch` takes a `ctx` but then starts the child with `context.Background()`, so canceled or timed-out starts can still spawn and hang. Pass the incoming context into `subprocess.Launch` instead.


As per coding guidelines, `Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/launcher.go` around lines 56 - 73, The Launch method on
localLauncher is ignoring its incoming ctx and uses context.Background() when
calling subprocess.Launch, so cancellation/timeouts from the caller are lost;
update the subprocess.Launch call to pass the provided ctx (the first parameter
of Launch) instead of context.Background(), preserving the rest of the
LaunchConfig fields and behavior; ensure you reference the Launch method on
localLauncher and the subprocess.Launch invocation when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `localLauncher.Launch` ignores its `ctx` and always calls `subprocess.Launch(context.Background(), ...)`, so caller cancellation cannot stop process startup. The fix is to pass the inbound context through and add a regression test for cancellation.
