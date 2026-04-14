---
status: resolved
file: internal/extension/host_api_test.go
line: 2398
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565HzR,comment:PRRC_kwDOR5y4QM63qGap
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Rename the nested suite labels to `Should...`.**

`MissingManager` and `InvalidInputs` still break the required `t.Run("Should...")` convention for Go tests. As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 2288 - 2398, Rename the two
outer test suite labels that violate the t.Run("Should...") convention: change
the t.Run call using the literal "MissingManager" to a Should-prefixed label
(e.g., "ShouldRejectWhenManagerMissing") and change the t.Run call using
"InvalidInputs" to a Should-prefixed label (e.g., "ShouldRejectInvalidInputs");
locate the offending calls by finding the t.Run invocations that wrap the
MissingManager and InvalidInputs blocks in internal/extension/host_api_test.go
and update only the string names to follow the "Should..." pattern.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The outer suite labels `MissingManager` and `InvalidInputs` violate the repo’s `Should...` test naming convention. I will rename those two wrapper labels only.
  Resolution: Renamed the two outer Host API task-validation suites to `Should...` labels only.
