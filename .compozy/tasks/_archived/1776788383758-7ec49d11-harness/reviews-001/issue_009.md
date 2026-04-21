---
status: resolved
file: internal/daemon/composed_assembler_test.go
line: 391
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMs,comment:PRRC_kwDOR5y4QM65IPEA
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Convert the new top-level tests to `t.Run("Should...")` cases.**

The added startup-assembly scenarios are valid, but they should follow the repository’s mandatory test-case pattern.


As per coding guidelines `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/composed_assembler_test.go` around lines 211 - 391, Multiple
new top-level test functions violate the repo pattern that requires test cases
to be subtests using t.Run("Should..."); locate the functions
TestComposedAssemblerAssembleStartupUsesEligibleSectionOrdering,
TestComposedAssemblerAppliesBudgetPolicies,
TestComposedAssemblerDeduplicatesEligibleSectionNames, and
TestComposedAssemblerAssembleStartupLoadsBundledNetworkSectionDescriptor and
convert them into t.Run subtests (e.g., a single TestComposedAssembler* parent
or an existing parent test) by moving each test body into t.Run("Should <short
description>", func(t *testing.T){ ... }) and keep t.Parallel() inside the
subtest; preserve the existing logic, assertions, and references to
NewComposedAssembler, WithPromptSectionDescriptors,
assembleStartupPrompt/assemblePrompt, NewHarnessContextResolver, and
defaultStartupPromptSectionDescriptors when relocating the code.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The cited tests are already behaviorally sound and cover real startup-section logic; the review comment is only about a preferred `Should...` subtest wrapper.
  - The loaded repo instructions do not establish that wrapper as a required invariant, and the surrounding daemon tests do not follow it consistently.
  - Because there is no demonstrated bug or missing verification signal, this item is not actionable.
