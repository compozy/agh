---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/acp/client_test.go
line: 424
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:93bf2224ad10
review_hash: 93bf2224ad10
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 002: Cover the new routing fields in this metadata round-trip.
## Review Comment

This fixture now sets `Surface` and `DirectID`, but the test never asserts them. A regression that drops the new conversation-routing metadata would still pass here.

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling" and "Check tests can fail when business logic changes."

Also applies to: 462-467

## Triage

- Decision: `valid`
- Notes: `TestPromptTransmitsStructuredMetadata` seeds `Surface` and `DirectID` in `PromptNetworkMeta` but only asserts `MessageID`, `WorkID`, and `Trust`. A regression that strips the routing metadata would still pass. Add assertions for the new routing fields in the decoded payload.
