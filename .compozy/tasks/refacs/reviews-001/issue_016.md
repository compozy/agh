---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/sse.go
line: 33
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:abecf6ea2d26
review_hash: abecf6ea2d26
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 016: Avoid emitting empty SSE records and accept valid field:value lines.
## Review Comment

Line 33 appends a record for every blank separator, even when `current` is empty, which can produce spurious entries. Also, Lines 40-48 only parse `"id: "`, `"event: "`, `"data: "` (with space), so valid SSE `field:value` lines are silently ignored.

## Triage

- Decision: `VALID`
- Notes:
  `ParseSSE` appends an empty record for every blank separator, even when no fields were accumulated, and it only accepts `field: value` with a space. Valid SSE also allows `field:value`, so the parser should accept both forms and skip empty frames.
