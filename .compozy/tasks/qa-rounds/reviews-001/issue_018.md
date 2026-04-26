---
status: resolved
file: internal/session/query.go
line: 279
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:43a18e2d5bd1
review_hash: 43a18e2d5bd1
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 018: Consider normalizing meta.Model during hydration for consistency.
## Review Comment

Optional: trim at read time too, so older/manual metadata with stray whitespace doesn’t leak into API/session snapshots.

## Triage

- Decision: `VALID`
- Notes: `sessionInfoFromMeta` copies `meta.Model` directly while session creation paths trim resolved model names. Manual or older metadata with whitespace can leak inconsistent snapshots. Fix by trimming `meta.Model` during hydration. A focused assertion in `internal/session/query_test.go` is needed even though that test file is outside the batch list because it validates the hydration behavior.
