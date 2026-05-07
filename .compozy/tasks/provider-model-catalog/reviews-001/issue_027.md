---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/settings/service_test.go
line: 846
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:f715dde343d5
review_hash: f715dde343d5
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 027: Assert that the empty providers.custom.models table is removed too.
## Review Comment

This only proves the nested keys disappeared. If the writer still leaves `[providers.custom.models]` behind, the clear-path regression still passes while an empty overlay remains in the file. Add a negative check for the table header itself.

As per coding guidelines, "Check tests can fail when business logic changes".

## Triage

- Decision: `UNREVIEWED`
- Notes:
