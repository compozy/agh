---
status: pending
file: internal/cli/client.go
line: 64
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:cb616e548bf2
review_hash: cb616e548bf2
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 003: Add a compile-time assertion for unixSocketClient.
## Review Comment

`DaemonClient` gained another method, but no `var _ DaemonClient = (*unixSocketClient)(nil)` guard exists. This assertion catches interface drift at compile time instead of later through call-site failures.

## Triage

- Decision: `UNREVIEWED`
- Notes:
