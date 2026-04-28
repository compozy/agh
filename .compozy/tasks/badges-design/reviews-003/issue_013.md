---
status: pending
file: web/src/systems/session/adapters/session-api.test.ts
line: 336
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:3a83f40579c6
review_hash: 3a83f40579c6
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 013: Make the query assertion order-agnostic.
## Review Comment

This assertion is coupled to parameter order. If serialization order changes, the test can fail even when behavior is correct.

## Triage

- Decision: `UNREVIEWED`
- Notes:
