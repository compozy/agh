---
status: resolved
file: internal/api/core/tasks_test.go
line: 918
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:5d026632dc4f
review_hash: 5d026632dc4f
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 001: Consider adding delete to the remaining error-path matrices for parity.
## Review Comment

Nice addition in the actor-resolver error table. To fully cover this new critical endpoint, mirror `DELETE /tasks/:id` in the service-unavailable and manager-error request matrices in this file as well.

As per coding guidelines, "Must Check: Focus on critical paths: workflow execution, state management, error handling".

## Triage

- Decision: `valid`
- Notes:
  The actor-resolver matrix already includes `DELETE /tasks/:id`, but the service-unavailable and manager-error matrices still omit it. That leaves the new delete endpoint without parity coverage in two of the three error-path tables. I will add delete coverage to both matrices and use a delete-specific manager error case that exercises the runtime `400` mapping for delete validation failures.
