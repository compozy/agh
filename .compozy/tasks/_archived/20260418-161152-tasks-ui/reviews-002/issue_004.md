---
status: resolved
file: internal/api/core/tasks_surface_internal_test.go
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUW,comment:PRRC_kwDOR5y4QM65ChGl
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use the required `Should...` subtest naming pattern.**

Subtest names like `"task list"` and `"workspace lookup failure"` don’t follow the required convention. Please rename these to `t.Run("Should ...")` style for consistency and policy compliance.



As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.


Also applies to: 76-76, 133-133, 210-210, 240-240, 279-279, 304-304

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_internal_test.go` at line 30, Update all
t.Run subtest names in the tasks_surface_internal_test.go tests to follow the
required "Should ..." pattern; specifically replace occurrences like t.Run("task
list") and t.Run("workspace lookup failure") (and the other reported occurrences
at the same t.Run call sites) with descriptive phrases beginning with "Should"
(e.g., t.Run("Should list tasks" or t.Run("Should fail workspace lookup")),
keeping the rest of the test bodies unchanged so the unique test behavior
(inside the same t.Run closures) remains intact.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `internal/api/core/tasks_surface_internal_test.go` uses subtest names like `"task list"` and `"workspace lookup failure"` instead of the repo-standard `t.Run("Should ...")` naming convention.
- Root cause analysis: The file predates or skipped the enforced test naming standard.
- Intended fix: Rename every subtest in this file to descriptive `Should ...` names without changing the test logic.
- Resolution: Renamed the scoped subtests to `Should ...` names and added regression coverage for the bounded draft-compensation helper in the same file.
- Verification:
  - `go test ./internal/api/contract ./internal/api/core ./internal/daemon`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
