---
status: resolved
file: internal/task/hooks_test.go
line: 181
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vS,comment:PRRC_kwDOR5y4QM67Z0NN
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `t.Run("Should...")` subtests for the newly added hook-context cases.**

The coverage is valuable, but these two cases should be structured as `t.Run("Should...")` subtests to match the repository’s mandatory test pattern.



As per coding guidelines, "**MUST use t.Run("Should...") pattern for ALL test cases**."


Also applies to: 183-224

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/hooks_test.go` around lines 136 - 181, The test
TestTaskRunObservationHooksDetachFromCallerCancellation must be converted to use
t.Run subtests for the two hook-context assertions: wrap the
enqueued-cancel/assertion logic into a t.Run("Should keep enqueued hook context
active") subtest and the post-claim-cancel/assertion into a t.Run("Should keep
post-claim hook context active") subtest; keep the setup (store, manager with
WithTaskRunHooks, CreateTask, EnqueueRun/ClaimRun and their cancels) but move
the specific cancel+assertContextStillActive(enqueuedCtx, t, "enqueued") and
cancel+assertContextStillActive(postClaimCtx, t, "post-claim") calls into their
respective t.Run blocks so the file follows the required t.Run("Should...")
pattern while still referencing enqueuedCtx, postClaimCtx,
manager.EnqueueRun/ClaimRun and assertContextStillActive.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `TestTaskRunObservationHooksDetachFromCallerCancellation` currently performs the enqueued and post-claim hook context assertions inline in the top-level test body. The repository's Go test convention requires explicit `t.Run("Should ...")` subtests for each case. The fix is to keep the shared manager/task setup intact and move each cancellation assertion into a named subtest.
