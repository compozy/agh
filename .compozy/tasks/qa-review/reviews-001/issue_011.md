---
status: resolved
file: internal/store/schema_test.go
line: 65
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQu,comment:PRRC_kwDOR5y4QM67VX7N
---

# Issue 011: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use the repo’s required `t.Run("Should...")` structure for these new scenarios.**

Both added migration-table cases are introduced as top-level tests instead of the required subtest pattern. As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.



Also applies to: 116-117

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/schema_test.go` around lines 64 - 65, The two new test cases
were added as top-level tests instead of using the repo’s required
t.Run("Should...") subtest pattern; update
TestRunMigrationsUsesIndependentMigrationTables (and the other top-level cases
around the 116-117 area) to wrap each scenario in t.Run("Should ...", func(t
*testing.T) { ... }) subtests (preserve t.Parallel where appropriate inside each
subtest) so every test case follows the "Should..." naming convention and uses
t.Run rather than being standalone top-level tests.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The migration-table namespace regressions were added as standalone top-level tests instead of named subtests.
  - Root cause: the new scenarios were appended as separate test functions instead of being wrapped in the repo's required `t.Run("Should...")` structure.
  - Fix plan: wrap the affected migration scenarios in named subtests and keep their current migration assertions unchanged.
