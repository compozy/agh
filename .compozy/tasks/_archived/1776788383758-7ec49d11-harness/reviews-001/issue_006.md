---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 323
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMp,comment:PRRC_kwDOR5y4QM65IPD9
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `t.Run("Should...")` for this new integration scenario.**

The parity test itself is valuable, but it should be structured under the required subtest naming/pattern.


As per coding guidelines `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/transport_parity_integration_test.go` around lines 216 -
323, The test function TestUDSTransportObserveHarnessLifecycleParityMatchesHTTP
does not follow the required subtest naming pattern; wrap the existing test body
inside a t.Run("Should ...") subtest (e.g., t.Run("Should match observe harness
lifecycle parity between UDS and HTTP", func(t *testing.T) { ... })) and move
all current logic into that closure, keeping the function name as the top-level
test to satisfy the test harness while ensuring the subtest uses the "Should..."
pattern; update any t references inside to use the closure parameter (func(t
*testing.T)).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This is the same style-only `Should...` complaint as issue 001, not a behavioral defect.
  - The repo does not define an enforced rule that every top-level Go test must wrap its body in a `t.Run("Should...")` subtest, and this file already follows the prevailing local style.
  - No functional or test-quality regression was demonstrated, so there is nothing to remediate in code.
