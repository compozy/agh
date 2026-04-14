---
status: resolved
file: internal/daemon/task_runtime_test.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107909499,nitpick_hash:c00b2ac040ed
review_hash: c00b2ac040ed
source_review_id: "4107909499"
source_review_submitted_at: "2026-04-14T17:26:14Z"
---

# Issue 008: Test case names should use "Should" prefix pattern.
## Review Comment

Per coding guidelines, test cases should use `t.Run("Should...")` pattern. Current names like `"workspace task uses workspace id"` and `"claimed without session requeues"` are descriptive but don't follow the prescribed format.

As per coding guidelines, "MUST use t.Run('Should...') pattern for ALL test cases".

Also applies to: 167-172

## Triage

- Decision: `valid`
- Notes:
  Several task-runtime test tables still use descriptive names that do not follow the enforced `Should...` subtest convention. I will rename the affected table entries only.
  Resolution: Renamed the affected task-runtime table entries to `Should ...` labels only.
