---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 290
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMP,comment:PRRC_kwDOR5y4QM645au8
---

# Issue 006: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap these top-level tests in `t.Run("Should...")` cases.**

These functions still execute scenarios directly in the test body, and a few of them already bundle multiple behaviors into one case. Split them into named subtests so each behavior is isolated and matches the required test shape. As per coding guidelines, "MUST use `t.Run("Should...")` pattern for ALL test cases".



Also applies to: 292-316, 318-384, 386-412, 414-455, 457-500, 502-595

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 248 - 290, The test
function TestToolMCPSourceSyncerHandlesNilReceiverAndTriggerFailures must be
converted to use a named subtest; wrap the existing test body inside
t.Run("Should handle nil receiver and trigger failures", func(t *testing.T) {
... }) so the scenario is isolated and follows the required t.Run("Should...")
pattern; do the same pattern for the other top-level tests noted (the functions
covering lines 292-316, 318-384, 386-412, 414-455, 457-500, 502-595) by wrapping
each distinct scenario in a t.Run call with a "Should..." descriptive name,
keeping the existing setup (e.g., nilSyncer, newToolMCPSourceSyncer, the trigger
failure closure, and assertions on syncer.Sync) inside their respective subtest
closures.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The affected functions are already single-behavior top-level tests. Wrapping each whole function body in a single `t.Run("Should...")` block would not increase isolation or change assertion granularity.
  - The repo instructions available in this workspace do not impose a universal `Should...` subtest wrapper for every test function, and the same file already mixes plain top-level tests with subtest-heavy cases.
  - This is stylistic churn rather than a correctness, determinism, or coverage defect, so it is not actionable in this remediation batch.
