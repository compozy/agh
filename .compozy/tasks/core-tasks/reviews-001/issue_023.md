---
status: resolved
file: internal/extension/manager_test.go
line: 788
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aoe,comment:PRRC_kwDOR5y4QM63mgRz
---

# Issue 023: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**The “restart” case can pass without proving that anything was reloaded.**

The success path starts a manager against an empty registry and only checks that `Reload()` returns `nil`, so a no-op implementation would still satisfy this test. Please seed at least one extension and assert an observable reload side effect, and split the scenarios into `t.Run("Should...")` cases so each branch stands on its own.
 

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases", "Focus on critical paths: workflow execution, state management, error handling", and "Ensure tests verify behavior outcomes, not just function calls".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manager_test.go` around lines 753 - 788,
TestManagerReloadValidatesAndRestarts currently only verifies Reload() returns
nil and can be satisfied by a no-op; update the test to seed the registry with
at least one extension (use newRegistryTestEnv / env.registry or helper that
registers an extension) before starting the manager (NewManager /
startedManager), then call startedManager.Reload and assert an observable side
effect such as the extension's Start/Reload handler being invoked, a changed
Manager state, or a registry reload counter; also split the existing checks into
separate t.Run("Should ...") subtests for the nil manager case, canceled context
case, missing registry case, and the successful reload case so each scenario is
isolated and verifies behavior, not just return values.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The current success case starts a manager against an empty registry and only checks that `Reload()` returns `nil`. A no-op implementation would satisfy that branch, so the test does not prove restart behavior.
  The fix is to split the scenarios into explicit `t.Run("Should ...")` subtests, seed the registry with a real extension, and assert an observable reload side effect through the fake launcher/process state so the test proves `Reload()` actually stops and starts managed extensions.
  Resolution: Split the test into `Should ...` subtests, installed a real registry fixture, and asserted restart behavior via a second process launch, first-process shutdown, and updated runtime PID after `Reload()`.
