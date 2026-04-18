---
status: resolved
file: internal/store/globaldb/global_db_task.go
line: 252
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:64712bdb2ad3
review_hash: 64712bdb2ad3
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 024: Case-insensitive search prevents index usage.
## Review Comment

Using `LOWER(title) LIKE ?` and `LOWER(COALESCE(identifier, ''))` prevents SQLite from using indexes on these columns. For small-to-medium datasets this is fine. If search performance becomes problematic, consider:
- Adding computed columns or expression indexes (SQLite 3.9+)
- Storing lowercase versions of title/identifier
- Using FTS5 for full-text search

## Triage

- Decision: `invalid`
- Reasoning: this is another advisory optimization note rather than a correctness bug. The current case-insensitive search behavior is intentional and already covered by persistence/search tests.
- Reasoning: changing the schema/search strategy to expression indexes, lowercase shadow columns, or FTS would be a separate search-architecture task, not a review-fix required to keep this batch correct.

## Resolution

- Closed as `invalid`.
- No code change was made because the review describes a search-index optimization tradeoff rather than a defect in this batch.
