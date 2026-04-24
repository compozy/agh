---
status: resolved
file: internal/session/provider_lifecycle_integration_test.go
line: 68
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:613efb7b4e4b
review_hash: 613efb7b4e4b
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 008: Guard startCalls length before indexing.
## Review Comment

These assertions can panic on regressions (`index out of range`) and hide the real failure cause.

Also applies to: 133-134

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The tests index `h.driver.startCalls[1]` without proving that two start calls were recorded, so a regression would panic before reporting the real mismatch. I will add explicit length guards before indexing in both affected assertions.
