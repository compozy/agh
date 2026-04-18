---
status: resolved
file: internal/acp/types_test.go
line: 229
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMh,comment:PRRC_kwDOR5y4QM65IPD0
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Align new tests with mandatory `t.Run("Should...")` convention.**

The added tests validate the right behavior, but their shape/naming does not meet the enforced Go test format requirements.


As per coding guidelines: "Use table-driven tests with subtests (t.Run) as default in Go tests" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/types_test.go` around lines 139 - 229, Rename or wrap each
top-level test into a t.Run subtest using the "Should ..." pattern and keep
t.Parallel() inside the subtest: for
TestPromptMetaValidateSyntheticRequiresWakeupReason, wrap the existing
assertions in t.Run("Should require a wakeup reason for synthetic turns", func(t
*testing.T){ t.Parallel(); ... }); for
TestPromptMetaValidateRejectsSyntheticFieldsOnUserAndNetworkTurns, convert the
table loop to call t.Run("Should reject synthetic fields on <case>", func(t
*testing.T){ t.Parallel(); ... }) for each case (keeping the existing tc.name
for clarity); and for TestPromptSyntheticMetaNormalizeAndValidate wrap the
validation steps in t.Run("Should normalize and validate synthetic meta", func(t
*testing.T){ t.Parallel(); ... }); ensure test names match the "Should..."
convention and keep all existing assertions and calls to PromptMeta.Validate,
PromptSyntheticMeta.Normalize, IsZero, and Validate unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The repo instructions loaded for this run require table-driven tests with subtests as a default, but they do not define a universal `t.Run("Should...")` naming mandate for every Go test.
  - This file and the wider repo already contain many top-level `Test...` functions without `Should...` subtests, so this comment does not identify a correctness regression or a violated enforced rule.
  - No code change is warranted for this review item; I will close it as analysis-only in the resolution pass.
