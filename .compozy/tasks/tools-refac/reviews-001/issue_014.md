---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 354
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:a437bae88376
review_hash: a437bae88376
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 014: Avoid pinning this test to JSON field order.
## Review Comment

This exact-string comparison is brittle to harmless marshaling changes. Decode `created.Record.Spec` and assert the projected fields instead of the serialized byte layout.

## Triage

- Decision: `VALID`
- Notes: `TestUDSToolResourceCRUDRoundTripTriggersProjection` compares `created.Record.Spec` as one exact JSON string. That couples the test to marshal field order. Decode the spec JSON and assert the projected fields directly.
