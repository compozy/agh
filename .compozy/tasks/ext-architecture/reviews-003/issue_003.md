---
status: resolved
file: internal/acp/client_test.go
line: 524
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q1zm,comment:PRRC_kwDOR5y4QM620rah
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use `Should...` phrasing for subtest names to satisfy test policy.**

The subtest cases in this block currently use labels like `"resource missing request error"` rather than the required `Should...` format.



As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client_test.go` around lines 521 - 524, The subtest names must
follow the "Should..." pattern: update the t.Run invocation inside the tests
loop (the block that declares tc := tc and calls t.Run(name, func(t *testing.T)
{ t.Parallel() ... })) to use a "Should ..." phrasing (e.g., t.Run("Should
"+name, ...) or map each test key to a "Should ..." string) so all subtest names
conform to the required pattern; ensure you still capture tc with tc := tc and
keep t.Parallel() unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The subtests in `TestIsLoadSessionResourceMissing` currently use free-form labels instead of the repo’s dominant `Should...` subtest naming pattern.
  - Root cause: this older test block predates the naming cleanup used elsewhere in the package.
  - Fix plan: rename the subtests to `Should...` phrasing without changing the test table, captured variables, or parallel execution.
  - Implemented: renamed the ACP subtest cases to `Should...` names and kept the existing table, captured loop variable, and parallel execution unchanged.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.
