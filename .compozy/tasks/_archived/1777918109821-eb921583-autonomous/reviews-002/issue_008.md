---
status: resolved
file: internal/api/udsapi/agent_identity_test.go
line: 173
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tm,comment:PRRC_kwDOR5y4QM67YhqN
---

# Issue 008: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use the required `t.Run("Should...")` pattern consistently.**

The first table uses free-form subtest names, and `TestAgentMeReturnsValidatedCallerIdentity` is still a bare top-level case. Please wrap/rename these so every case follows the repository's required `Should...` subtest pattern.



As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases and `Table-driven tests with subtests (t.Run) as default.`

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/agent_identity_test.go` around lines 16 - 173, The tests
violate the required "Should..." t.Run pattern: update the table-driven names in
TestAgentMeRejectsInvalidCallerIdentity so each test case's name begins with
"Should ..." and is invoked via t.Run(tt.name, func(t *testing.T) { ... })
(preserve t.Parallel() inside each subtest), and wrap the standalone
TestAgentMeReturnsValidatedCallerIdentity body inside a t.Run("Should return
validated caller identity", func(t *testing.T) { ... }) (keeping the existing
t.Parallel and assertions); reference the test functions
TestAgentMeRejectsInvalidCallerIdentity and
TestAgentMeReturnsValidatedCallerIdentity and the table loop that calls t.Run
for the fix.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestAgentMeRejectsInvalidCallerIdentity` uses free-form table names and `TestAgentMeReturnsValidatedCallerIdentity` has assertions directly in the top-level test body. Both violate the AGH test convention that every case is a `t.Run("Should ...")` subtest. The fix is to rename table cases and wrap the standalone test body in a `Should...` subtest.
- Resolution: Renamed/wrapped the UDS agent identity tests with `Should...` subtests and verified with focused tests plus full `make verify`.
