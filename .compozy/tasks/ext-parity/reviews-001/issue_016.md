---
status: resolved
file: internal/api/httpapi/helpers_test.go
line: 125
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:e9cf6e39d046
review_hash: e9cf6e39d046
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 016: Consider extracting shared resource-handler test config
## Review Comment

Both constructors duplicate nearly the full `handlerConfig` setup; a small internal helper would reduce maintenance drift.

Also applies to: 157-177

## Triage

- Decision: `INVALID`
- Notes: This is a deduplication suggestion only. The duplicated test setup is local to two helpers, does not create an observable bug, and extracting a shared constructor would be unrelated refactoring outside the substantive review findings.
