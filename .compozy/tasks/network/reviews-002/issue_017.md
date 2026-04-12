---
status: resolved
file: internal/network/manager_test.go
line: 59
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56T_fK,comment:PRRC_kwDOR5y4QM624toZ
---

# Issue 017: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the specific validation failure for each constructor case.**

This test currently passes on any non-nil error, so a regression to the wrong failure branch would still look green. Please assert the expected error reason per case instead of only checking `err != nil`.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager_test.go` around lines 44 - 59, Update
TestNewManagerRequiresEnabledConfigAndPrompter to assert specific validation
failures instead of any non-nil error: call NewManager for each case (disabled
config via testManagerConfig() with cfg.Enabled=false, nil prompter via passing
nil for newFakeDeliveryPrompter(), and nil context via nilTestContext()) and
replace the generic err==nil checks with assertions that inspect the error
content/type (use testing helpers like ErrorContains or ErrorAs) to verify the
correct validation reason is returned for each invocation of NewManager.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestNewManagerRequiresEnabledConfigAndPrompter` currently treats any non-nil error as success, so regressions to the wrong validation path would still pass.
- Fix plan: Assert the specific validation failure for each constructor case instead of only checking `err != nil`.
