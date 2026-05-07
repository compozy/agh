---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/extension/manager_model_source_test.go
line: 74
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sn,comment:PRRC_kwDOR5y4QM6-6bsT
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the denied error contract here.**

`err != nil` lets unrelated failures satisfy this test. Please pin the expected denial shape/message so it only passes when the missing model-source capability is what failed.



As per coding guidelines, `**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/extension/manager_model_source_test.go` around lines 67 - 74, The
test currently only checks err != nil for manager.ListModelSourceRows and can be
satisfied by unrelated failures; update the assertion to validate the specific
denied-service error shape/message (use testing helpers like
require.ErrorContains/Assert or errors.As to match the denial error type) so the
test only passes when the call fails due to missing model-source capability;
locate the call to ListModelSourceRows (using
extensioncontract.ModelSourceListParams{ProviderID: "codex"}) and replace the
generic nil check with an assertion that the error contains the expected denial
string or can be cast to the denied error type.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/extension/manager_model_source_test.go:67-74` only checks `err != nil`.
  - That allows unrelated subprocess, registry, or transport failures to satisfy the test instead of the intended denied-service-method contract.
  - Fix: assert the unavailable-service error shape specifically, including the service-method message and sentinel.
