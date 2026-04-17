---
status: resolved
file: internal/automation/resource.go
line: 43
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:d48f9da21de7
review_hash: d48f9da21de7
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 022: Wrap validation errors with resource-specific context.
## Review Comment

These paths currently return raw scope/binding/spec errors, which makes job-vs-trigger failures harder to diagnose once they bubble out of the codec. Wrap each failing step before returning it.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

Also applies to: 59-72

## Triage

- Decision: `VALID`
- Notes: `validateJobResourceSpec` and `validateTriggerResourceSpec` still return raw scope-binding and spec-validation errors, so callers lose whether the failure happened while validating scope, binding resource scope, or validating the job/trigger payload itself. Wrapping each failure with resource-specific context improves diagnosis without changing validation behavior. The needed codec assertions live in the minimal out-of-scope file `internal/automation/resource_test.go`, which is not part of the scoped list.
