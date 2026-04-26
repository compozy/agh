---
status: resolved
file: internal/transcript/transcript_test.go
line: 680
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQz,comment:PRRC_kwDOR5y4QM67VX7S
---

# Issue 014: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap these scenarios in `t.Run("Should...")` subtests.**

These updated cases are still declared as standalone tests, but the repo test rules require the `t.Run("Should...")` pattern for each scenario. As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.



Also applies to: 754-755

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/transcript/transcript_test.go` around lines 679 - 680, The test
function TestMarshalAgentEventExtractsToolResultShapeWithoutPersistingRaw
currently contains standalone scenarios; wrap each scenario in a
t.Run("Should...") subtest (e.g., t.Run("Should extract tool result shape
without persisting raw", func(t *testing.T){...})) so it follows the repo rule;
apply the same change to the other failing test cases referenced in the file
(the other test functions that currently have standalone scenarios around those
lines) by converting their scenario blocks into t.Run("Should...") subtests with
descriptive names and moving assertions inside each subtest closure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The new transcript serialization regressions are still declared as standalone top-level tests.
  - Root cause: scenario-style tests were added without the file's required `t.Run("Should...")` wrapper structure.
  - Fix plan: wrap the tool-result extraction and adjacent round-trip assertions in named subtests without changing the canonical transcript expectations.
