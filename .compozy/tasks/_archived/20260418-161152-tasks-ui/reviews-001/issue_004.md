---
status: resolved
file: internal/api/core/parsers.go
line: 185
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:790fc71cebd6
review_hash: 790fc71cebd6
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 004: Validate enum-like query params in the exported parse helpers.
## Review Comment

`ParseTaskListQuery`, `ParseTaskDashboardQuery`, and `ParseTaskInboxQuery` normalize fields like `scope`, `status`, `owner_kind`, `origin_kind`, and `lane`, but they still return invalid values without error. Right now that only gets caught later in the domain-query builders, so any caller that reuses these exported parsers directly gets inconsistent validation behavior.

## Triage

- Decision: `valid`
- Notes: The exported parse helpers normalize enum-like values but currently return unsupported values without error. Validation only happens later in domain-query builders, which means callers that reuse `ParseTaskListQuery`, `ParseTaskDashboardQuery`, or `ParseTaskInboxQuery` directly get inconsistent error timing. I’ll add enum validation in the parse helpers while leaving workspace binding to the domain conversion layer.
