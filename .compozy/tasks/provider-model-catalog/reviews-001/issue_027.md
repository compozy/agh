---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
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

- Decision: `valid`
- Notes:
  - The clear-models assertion in `internal/settings/service_test.go` currently proves the nested model keys are removed but does not prove the `[providers.custom.models]` table header itself is gone.
  - A regression could still leave an empty overlay table in the persisted config and this test would miss it.
  - Fix approach: add a negative assertion for the `[providers.custom.models]` header alongside the existing key checks.
  - Resolved in `internal/settings/service_test.go`; verified with focused package tests and full `make verify`.
