---
status: resolved
file: internal/api/core/tasks_internal_test.go
line: 366
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsL,comment:PRRC_kwDOR5y4QM67YHCl
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this new case in a `t.Run("Should...")` subtest.**

The assertions are good, but this new test case skips the repo’s required `Should...` subtest pattern. Please nest it in a named subtest and keep `t.Parallel()` inside that block.



As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_internal_test.go` around lines 307 - 366, Wrap the
test body of TestTaskRunPayloadFromRunExposesLeaseStateWithoutRawClaimToken in a
t.Run subtest named with the "Should..." pattern (e.g. t.Run("Should not expose
raw claim tokens and expose lease state", func(t *testing.T) { ... })), move the
existing t.Parallel() call inside that subtest function, and keep all current
assertions and use of TaskRunPayloadFromRun unchanged so the behavior and checks
remain the same.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestTaskRunPayloadFromRunExposesLeaseStateWithoutRawClaimToken` is a standalone behavior test with `t.Parallel()` at the top level instead of a named `Should...` subtest. Fix by nesting the current assertions in a descriptive subtest and moving `t.Parallel()` into that subtest.
