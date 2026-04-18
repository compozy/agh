---
status: resolved
file: internal/config/persistence_integration_test.go
line: 163
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRw,comment:PRRC_kwDOR5y4QM65B60a
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use the repo’s required `t.Run("Should...")` structure for these cases.**

These new integration cases are all top-level tests, which drifts from the test shape required elsewhere in the repo. As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (t.Run) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/persistence_integration_test.go` around lines 13 - 163, These
tests violate the repo rule requiring subtests with t.Run; wrap each existing
test body inside a t.Run call with a descriptive "Should ..." name and move
assertions into the t.Run closure (keep the top-level test functions
TestEditConfigOverlayGlobalWritePreservesStructureOnDisk,
TestEditConfigOverlayWorkspaceWriteLeavesGlobalConfigUntouched, and
TestPutMCPSidecarServerWritesAndPreservesUnaffectedEntries but replace their
current bodies with a single t.Run(...) that contains the current logic),
ensuring any t.TempDir or local variables are created inside the subtest closure
so they use the subtest's *testing.T context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The scoped integration tests are currently top-level bodies without a named `t.Run("Should ...")` wrapper. I will wrap each case in a subtest and keep all temp-dir setup inside the subtest closure so the test structure matches the project’s required pattern without changing behavior.
