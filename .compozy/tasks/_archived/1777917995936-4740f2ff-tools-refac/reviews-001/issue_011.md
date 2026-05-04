---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/spec/spec_test.go
line: 310
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:15438cad277e
review_hash: 15438cad277e
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 011: Add schema coverage for the tool search operations too.
## Review Comment

This block covers list/invoke/approvals/toolsets, but the dedicated tool search endpoints are still unasserted. A missing operation or wrong request/response schema there will slip through this suite.

As per coding guidelines, "Must Check: Focus on critical paths: workflow execution, state management, error handling".

## Triage

- Decision: `VALID`
- Notes: The OpenAPI spec test verifies tool list/invoke/approval/toolset list routes but does not assert the schemas for `/api/tools/search`, `/api/sessions/{id}/tools/search`, or `/api/toolsets/{id}`. Add schema coverage for these operations to catch missing routes or request/response drift.
