---
status: resolved
file: internal/store/globaldb/global_db_extra_test.go
line: 793
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:d711b2c188f0
review_hash: d711b2c188f0
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 016: Consider logging rollback errors in test cleanup.
## Review Comment

Line 797 ignores the rollback error with `_`. While this is common in test cleanup and unlikely to cause issues, the coding guidelines suggest handling all errors.

## Triage

- Decision: `valid`
- Root cause: the transactional cleanup currently discards rollback errors, which can hide unexpected cleanup failures during migration-copy coverage.
- Fix plan: keep the deferred rollback but report any non-nil rollback error through the test instead of swallowing it.
