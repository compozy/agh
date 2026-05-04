---
status: resolved
file: internal/cli/client_test.go
line: 428
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tp,comment:PRRC_kwDOR5y4QM67YhqQ
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap these new test bodies in `t.Run("Should...")` subtests.**

This change adds several standalone tests with direct assertions (`TestUnixSocketClientAgentMeSendsIdentityHeaders`, `TestUnixSocketClientAgentTaskErrorsRedactClaimTokens`, etc.) instead of the repo’s required `Should...` subtest pattern. Please nest each case in a named subtest and keep `t.Parallel()` inside those blocks.


As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client_test.go` around lines 29 - 428, The top-level test
functions (e.g., TestUnixSocketClientAgentMeSendsIdentityHeaders and
TestUnixSocketClientAgentTaskErrorsRedactClaimTokens) must be converted to use
the repository's required subtest pattern: wrap each test body in a
t.Run("Should ...", func(t *testing.T) { t.Parallel(); ... }) block so the
assertions run as a named "Should..." subtest, keeping t.Parallel() inside the
subtest closure and preserving existing logic (use the same client setup and
assertions inside the new subtest); update any other standalone tests in this
file to the same pattern so all cases follow t.Run("Should...").
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestUnixSocketClientAgentMeSendsIdentityHeaders` and `TestUnixSocketClientAgentTaskErrorsRedactClaimTokens` contain direct assertions in the top-level test body. They need `t.Run("Should ...")` wrappers to match AGH test conventions; the existing client logic and assertions can remain inside those subtests.
- Resolution: Wrapped the affected CLI client tests with `Should...` subtests and verified with focused tests plus full `make verify`.
