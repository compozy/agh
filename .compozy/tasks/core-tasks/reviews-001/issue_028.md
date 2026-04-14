---
status: resolved
file: internal/observe/tasks_test.go
line: 396
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aoy,comment:PRRC_kwDOR5y4QM63mgSM
---

# Issue 028: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the validation error shape, not only that it is non-nil.**

These checks stay green for any failure path. If `Validate()` starts rejecting the wrong field, this test still passes.



As per coding guidelines, `**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/tasks_test.go` around lines 388 - 396, The tests currently
only assert that Validate() returns non-nil, which is too weak; update the three
assertions to check the error shape/content for the specific invalid field. For
the TaskSummaryQuery cases (TaskSummaryQuery{Scope: taskpkg.Scope("bogus")} and
TaskSummaryQuery{OwnerKind: taskpkg.OwnerKind("bogus")}) and the
TaskMetricsQuery case (TaskMetricsQuery{OriginKind:
taskpkg.OriginKind("bogus")}), replace the plain nil checks with assertions that
the returned error either matches the expected validation error type via
errors.As or contains an expected substring (e.g., "scope", "owner kind",
"origin kind") using ErrorContains / strings.Contains so the test fails if
Validate() rejects a different field.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The current assertions only check that validation returns a non-nil error. That allows the tests to stay green if validation fails for the wrong field or with the wrong error class.
  I will strengthen the assertions to verify both the validation error type and field-specific error text so the tests fail if `Validate()` starts rejecting the wrong input.
  Resolution: Tightened the validation tests to assert `task.ErrValidation` and field-specific error text for invalid `scope`, `owner_kind`, and `origin_kind` cases.
