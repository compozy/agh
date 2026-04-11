---
status: resolved
file: internal/api/contract/contract_test.go
line: 267
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093889370,nitpick_hash:b2826f6c0380
review_hash: b2826f6c0380
source_review_id: "4093889370"
source_review_submitted_at: "2026-04-11T14:58:56Z"
---

# Issue 001: Expand HasChanges() cases to cover all mutable fields.
## Review Comment

Current cases are valid, but they only sample a subset of mutable fields. Adding one positive case per mutable field would make regressions in `HasChanges()` much harder to miss.

As per coding guidelines, `**/*_test.go`: "MUST test meaningful business logic, not trivial operations" and "Must Check: Focus on critical paths: workflow execution, state management, error handling".

Also applies to: 304-324

## Triage

- Decision: `valid`
- Notes:
- `UpdateJobRequest.HasChanges()` currently checks 8 mutable fields and `UpdateTriggerRequest.HasChanges()` checks 12 mutable fields, but the tests only exercise 2 positive cases for each type.
- That leaves regressions on fields like `AgentName`, `WorkspaceID`, `Prompt`, `Schedule`, `Retry`, `FireLimit`, `Event`, `Filter`, `WebhookID`, and `EndpointSlug` undetected.
- Fix plan: expand the table-driven cases so each mutable field has one positive case and the empty request remains the negative baseline.
