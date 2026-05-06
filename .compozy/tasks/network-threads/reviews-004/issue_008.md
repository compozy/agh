---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/spec/spec.go
line: 1275
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:a5ce6235de1c
review_hash: a5ce6235de1c
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 008: Fix the after cursor description for thread listing.
## Review Comment

This endpoint paginates threads, but the parameter text says “message id”. That will mislead clients reading the generated OpenAPI.

## Triage

- Decision: `valid`
- Notes: The `GET /api/network/channels/{channel}/threads` spec describes the `after` cursor as a "message id" even though the endpoint paginates thread summaries. That description is inaccurate and should be corrected in `internal/api/spec/spec.go`. Regenerating `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` is also required so the checked-in contract artifacts stay synchronized with the source spec.
