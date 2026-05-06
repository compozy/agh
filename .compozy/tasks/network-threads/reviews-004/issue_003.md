---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/core/agent_channels.go
line: 860
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:444412e0b640
review_hash: 444412e0b640
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 003: Hardcoded query key limits reusability.
## Review Comment

The `parsePositiveIntQuery` function now hardcodes the key as `"limit"`, which reduces its flexibility compared to the previous signature that accepted a key parameter. If only the `limit` query parameter needs this parsing, consider renaming to `parseLimitQuery` for clarity.

## Triage

- Decision: `invalid`
- Notes: `parsePositiveIntQuery` is no longer a generic helper; all current call sites in `internal/api/core` parse only the `limit` query parameter. The hardcoded key matches actual usage, so there is no correctness or maintainability bug to fix in this batch.
