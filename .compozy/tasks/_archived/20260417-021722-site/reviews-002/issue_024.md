---
status: resolved
file: internal/bundles/resource_test.go
line: 71
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:17d9e9346374
review_hash: 17d9e9346374
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 024: Make the wrong-plan failure assertion specific.
## Review Comment

`err != nil` will pass for any unrelated failure inside `Apply`, so this does not prove the type-check branch is exercised. Please assert the expected error type or at least an identifying substring.

As per coding guidelines, "`**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs)`."

## Triage

- Decision: `INVALID`
- Notes:
  - The reviewed file `internal/bundles/resource_test.go` does not exist in this checkout.
  - The current bundle tests in `internal/bundles/service_test.go` do not have a wrong-plan `Apply` case matching the comment.
  - No live assertion matches the reported failure mode, so this item is stale.
  - Result: resolved as stale after current-tree inspection; no code change required.
