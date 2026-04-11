---
status: resolved
file: internal/skills/bundled/bundled_test.go
line: 189
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZr,comment:PRRC_kwDOR5y4QM623eZ-
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Refactor new tests to required `t.Run("Should...")` table-driven form and stronger error assertions.**

The new test cases are not using the required subtest naming/pattern, and the error checks rely on `strings.Contains(err.Error(), ...)` instead of explicit error assertion helpers.


As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests`, `MUST use t.Run("Should...") pattern for ALL test cases`, and `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/bundled/bundled_test.go` around lines 107 - 189, Refactor the
four tests (TestBundledRegistryLoadsAghNetworkSkill,
TestBundledAghNetworkSkillMatchesSupportedCLICommands,
TestBundledLoadContentRejectsEmptySkillName,
TestBundledLoadContentRejectsMissingSkill) into table-driven subtests using
t.Run with the "Should..." naming pattern for each scenario, converting inline
assertions into subtests where appropriate; replace loose error checks that use
strings.Contains(err.Error(), ...) with explicit error assertion helpers (e.g.,
ErrorContains or ErrorAs) and update the expectations to assert the exact error
semantics, and for the content/flag checks move repeated cases into table loops
inside t.Run subtests so each case is reported as "Should ..." with clear
failure messages.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The new bundled-skill tests are passing but they do not follow the repo’s required `t.Run("Should...")` structure and they use loose `strings.Contains(err.Error(), ...)` checks for failures that can be asserted more precisely. I will refactor them into table-driven subtests and strengthen the error checks using explicit error semantics.
  Resolved by refactoring the bundled network tests in `internal/skills/bundled/bundled_test.go` into `Should...` subtests and replacing loose error matching with explicit `errors.Is` checks. Verified with package tests and a clean `make verify`.
