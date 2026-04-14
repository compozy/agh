---
status: resolved
file: internal/cli/task_test.go
line: 51
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565HzJ,comment:PRRC_kwDOR5y4QM63qGah
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Rename these subtests to the required `Should...` form.**

This table still feeds `t.Run(...)` with names that do not follow the enforced Go test naming convention. As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/task_test.go` around lines 16 - 51, The table-driven tests in
the tests variable use non-conforming names passed through t.Run(tt.name);
rename each tt.name value to the "Should ..." form (e.g. "Should require
workspace for workspace scope", "Should forbid workspace for global scope",
"Should require change flags on update", "Should reject clear owner with owner
mutation") so t.Run receives names that follow the enforced pattern; no
behavioral code changes needed beyond updating the string literals in the tests
slice.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The table-driven CLI test uses non-compliant subtest labels even though the assertions themselves are fine. I will rename the table entries to `Should ...` without changing behavior.
  Resolution: Renamed the CLI table entries to the required `Should ...` form with no behavioral changes.
