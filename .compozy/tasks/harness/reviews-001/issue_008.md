---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 173
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMq,comment:PRRC_kwDOR5y4QM65IPD-
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the required `t.Run("Should...")` test-case pattern here.**

This added integration scenario is currently a standalone top-level test and should follow the repository’s mandatory subtest format.


As per coding guidelines `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/udsapi_integration_test.go` around lines 112 - 173, The
test function TestUDSSessionTranscriptEndpointIncludesSyntheticTurns must be
converted to use the required subtest pattern: wrap the existing test logic
inside a t.Run("Should include synthetic turns in transcript", func(t
*testing.T) { ... }) call (keeping the same body and using the inner t
*testing.T), so that TestUDSSessionTranscriptEndpointIncludesSyntheticTurns only
calls t.Run with the described name and the current assertions and helper calls
remain unchanged inside the anonymous func.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This comment only requests a `t.Run("Should...")` wrapper and does not identify a correctness, reliability, or coverage problem.
  - The repository instructions available in this run do not enforce that convention for every Go test, and adjacent files use the same top-level test style.
  - I will close this as a non-actionable style preference rather than change the test structure without a real defect.
