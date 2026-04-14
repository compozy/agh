---
status: resolved
file: internal/automation/validate_test.go
line: 832
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107556463,nitpick_hash:30095d4b6862
review_hash: 30095d4b6862
source_review_id: "4107556463"
source_review_submitted_at: "2026-04-14T16:23:06Z"
---

# Issue 007: Cover the mirrored delegated validation case too.
## Review Comment

This only locks down the missing `task_id` branch. A regression in the `task_run_id` requirement for delegated runs would still slip through untested.

As per coding guidelines, `**/*_test.go`: Focus on critical paths: workflow execution, state management, error handling.

## Triage

- Decision: `valid`
- Notes:
  The new delegated-run validation regression covers the missing `task_id` branch but does not cover the mirrored missing `task_run_id` requirement for `RunDelegated`.
  Root cause: the regression test only locked down one half of the delegated-run invariant.
  Planned fix: add the complementary `task_run_id` failure assertion in `TestRunAndEnvelopeValidate`.

## Resolution

- Added the missing delegated-run regression in `internal/automation/validate_test.go` so both required delegated identifiers, `task_id` and `task_run_id`, are enforced by tests.
