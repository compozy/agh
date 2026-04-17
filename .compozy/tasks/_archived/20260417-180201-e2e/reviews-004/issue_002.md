---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 199
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM570BHi,comment:PRRC_kwDOR5y4QM646Fv6
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Adopt `t.Run("Should...")` subtests for the three test scenarios.**

Lines 34, 101, and 153 define standalone test cases; this file should structure cases with `t.Run("Should...")` (and preferably table-driven form) to match repo test requirements.



As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default in Go tests."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/transport_parity_integration_test.go` around lines 34 -
199, Replace the three standalone tests
(TestUDSTransportApprovalRouteDocumentsNotImplementedGap,
TestUDSTransportProjectionParityMatchesHTTPAndCLI,
TestUDSTransportPromptFailureProjectionUsesSharedRuntimeHarness) with a single
test that uses t.Run subtests (or a table-driven loop calling t.Run) named like
t.Run("Should ...") for each scenario; move each test body into its own subtest,
keep unique setup calls (e.g., acpmock.RequireDriver,
e2etest.StartRuntimeHarness, runtimeHarness.CreateSession,
seedTransportWebhookTrigger) inside the corresponding subtest, and call
t.Parallel() inside each subtest if parallelism is desired, ensuring you
preserve the original function/variable names (transportUDSApprovalAgent,
transportUDSAutomationAgent, transportUDSFaultyAgent, mustUnixRequest,
waitForHTTPAutomationRun, waitForUDSAutomationRun, waitForCLIAutomationRun,
e2etest.ValidateWebhookRunProjection, etc.) so the test logic and assertions
remain identical but conform to the repo's t.Run("Should...") pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This is a style preference, not a correctness defect in the current file.
  - The repo guidance requires subtests by default, but no repo-wide rule was found that every independent test must be collapsed into a single `t.Run("Should...")` wrapper; the batch comment is the only place that states that requirement.
  - These three integration scenarios have distinct harness setup, fixtures, and failure modes. Keeping them as separate top-level tests preserves focused `go test -run` targeting and does not reduce coverage or correctness.
  - No code change is warranted for this batch.
