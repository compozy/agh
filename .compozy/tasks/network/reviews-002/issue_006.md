---
status: resolved
file: internal/api/core/network_test.go
line: 379
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TblV,comment:PRRC_kwDOR5y4QM624BCT
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Break disabled/error scenarios into `Should...` subtests and strengthen error assertions.**

This block tests many distinct scenarios in one flow. A single early failure masks later regressions, and error-path checks mostly validate status code only. Add subtests per scenario and assert error payload content where applicable.

As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases` and `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_test.go` around lines 299 - 379, The
TestBaseHandlersNetworkErrorsAndDisabledMode currently bundles many scenarios in
one flow; split it into t.Run subtests with descriptive "Should..." names for
each scenario (e.g., "ShouldReturnDisabledStatus",
"ShouldReturnServiceUnavailableWhenPeersDisabled",
"ShouldMapNetworkStatusErrorTo500", "ShouldMapListSpacesErrorTo400",
"ShouldReturnBadRequestOnSendDecode", "ShouldMapSendTargetNotFoundTo404",
"ShouldReturnBadRequestWhenInboxMissing", "ShouldMapInboxInvalidFieldTo400"),
moving the relevant setup (fixture, toggling
fixture.Handlers.Config.Network.Enabled, and swapping fixture.Handlers.Network
stub implementations) into each subtest; within each subtest replace bare
status-code checks with stronger assertions that decode the JSON error payload
(using testutil.DecodeJSONResponse) and assert contents via ErrorContains or
errors.As where appropriate, and keep the final assertions on
core.StatusForNetworkError(validationErr) intact as their own subtest too—locate
and modify the TestBaseHandlersNetworkErrorsAndDisabledMode function and the
calls to performRequest, fixture.Handlers.Network, and
core.StatusForNetworkError to implement these changes.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestBaseHandlersNetworkErrorsAndDisabledMode` bundles disabled-mode behavior, transport availability, validation failures, and domain error mapping into one linear flow with mostly status-only checks.
- Fix plan: Break the scenarios into `Should...` subtests and assert decoded `contract.ErrorPayload` contents where an HTTP error body is part of the behavior.
