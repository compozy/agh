---
status: resolved
file: internal/api/core/network_test.go
line: 119
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TblS,comment:PRRC_kwDOR5y4QM624BCQ
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Split this into `t.Run("Should...")` subtests.**

This test currently bundles multiple independent behaviors (status mapping, send-request conversion, envelope conversion) into one case, which makes failures harder to localize and violates the project’s test-case structure requirement.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_test.go` around lines 18 - 119, The test
TestNetworkConversionHelpersPreserveMetadata bundles three independent checks;
split it into t.Run subtests named like t.Run("Should map status metadata"),
t.Run("Should convert NetworkSendRequest preserving metadata"), and
t.Run("Should convert Envelope preserving metadata") so failures are isolated.
For each subtest, move only the relevant setup/assertions into that t.Run body
(you can keep shared fixtures like deadline/status/payload/envelope definitions
at the top of the test or re-create minimal needed values inside each subtest),
and call the existing helper functions NetworkStatusPayloadFromStatus,
NetworkSendRequestFromPayload, and NetworkEnvelopePayloadFromEnvelope inside
their respective subtests, preserving the same assertions and error handling.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestNetworkConversionHelpersPreserveMetadata` currently combines status, send-request, and envelope conversions in one body, so one failure masks the others.
- Fix plan: Split the test into focused `t.Run("Should...")` subtests while preserving the existing assertions and fixtures.
