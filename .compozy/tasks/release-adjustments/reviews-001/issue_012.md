---
status: resolved
file: internal/network/manager_test.go
line: 563
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1m,comment:PRRC_kwDOR5y4QM67HMWz
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this scenario in a `Should...` subtest.**

The coverage looks good, but this new case skips the repo’s required `t.Run("Should...")` pattern that the rest of this file already uses.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager_test.go` around lines 489 - 563, Wrap the body of
TestManagerAuditsBusyQueueOverflowAsRejected in a t.Run subtest whose name
begins with "Should" (e.g. t.Run("Should audit busy queue overflow as rejected",
func(t *testing.T) { ... })). Move the existing test logic (context setup,
cfg/MAXQueueDepth, newFakeDeliveryPrompter, recordingAuditWriter, NewManager,
JoinChannel, Send calls, waitForCondition, auditor.rejectedForMessage and
manager.Status assertions) into that subtest, call t.Parallel() inside the
subtest (not just at the top-level), and keep existing cleanup/Shutdown logic
intact so behavior of NewManager, Send, JoinChannel, waitForCondition and
manager.Status is unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `TestManagerAuditsBusyQueueOverflowAsRejected` runs directly in the top-level body while surrounding tests use `Should...` scenario wrappers.
  - The fix is to move the existing setup, send calls, audit assertions, and shutdown cleanup into a `Should...` subtest with `t.Parallel()`.
