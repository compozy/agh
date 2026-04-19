---
status: resolved
file: internal/daemon/prompt_input_composite_integration_test.go
line: 130
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM2,comment:PRRC_kwDOR5y4QM65IPEK
---

# Issue 018: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap this scenario in a `Should...` subtest.**

This new integration case is still a bare top-level test, so it misses the repo’s required test-case structure. As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/prompt_input_composite_integration_test.go` around lines 18 -
130, The test function
TestPromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns
is a top-level test and must wrap its scenario in a t.Run("Should...") subtest;
refactor the body of
TestPromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns
so that the existing logic is moved into a single t.Run call (e.g.,
t.Run("Should preserve stored messages across user and network turns", func(t
*testing.T) { ...existing body... })) while keeping the function name and all
referenced symbols unchanged (compositeResolver,
newPromptInputCompositeAugmenter, manager.Prompt, manager.PromptNetwork,
loadStoredPromptMessages, etc.).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This is another style-only request to wrap a test body in a `Should...` subtest.
  - The current integration test already exercises the intended behavior, and the loaded repo instructions do not require that exact naming convention for every case.
  - I will close this item as non-actionable because it does not correspond to a functional defect or missing guard.
