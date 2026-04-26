---
status: resolved
file: internal/api/udsapi/agent_channels_test.go
line: 116
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vG,comment:PRRC_kwDOR5y4QM67Z0NA
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap this test case in `t.Run("Should ...")` to match required test structure.**

This new test is currently a direct top-level body without the required subtest naming pattern.


As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/agent_channels_test.go` around lines 72 - 116, Wrap the
existing TestAgentCoordinatorConfigRouteReturnsResolvedPayload body in a subtest
using t.Run("Should return resolved workspace coordinator payload", func(t
*testing.T) { ... }) so the test follows the required t.Run("Should ...")
pattern; locate the TestAgentCoordinatorConfigRouteReturnsResolvedPayload
function and move its current contents into a t.Run call while keeping all setup
(manager := activeAgentSessionManager, handlers := newTestHandlers,
handlers.CoordinatorConfig = agentCoordinatorConfigResolverFunc, engine :=
newTestRouter, performAgentKernelRequest, decodeJSONResponse and assertions)
unchanged inside the subtest body.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestAgentCoordinatorConfigRouteReturnsResolvedPayload` contains one direct top-level test body while this repo requires each case to run under a `t.Run("Should ...")` subtest. Fix by moving the existing setup, request, and assertions into a named `Should return resolved workspace coordinator payload` subtest.
