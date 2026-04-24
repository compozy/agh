---
status: resolved
file: internal/testutil/e2e/runtime_harness.go
line: 419
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:9a33cc5c3935
review_hash: 9a33cc5c3935
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 017: Consider consolidating the "address already in use" detection.
## Review Comment

The HTTP port retry check uses `"address already in use"` which could match both TCP and UDS errors on some systems. The UDS check is more specific with `"listen unix"` + `"bind: file exists"`. This works but the overlap could cause both flags to be true for certain edge cases.

## Triage

- Decision: `VALID`
- Notes:
  - `readinessFailureRetryReasons` treats any process log containing `address already in use` as an HTTP port conflict, even when the log could describe a Unix socket bind failure.
  - The fix is to split HTTP and UDS conflict detection into protocol-specific predicates so TCP and Unix listener conflicts cannot both be inferred from the same generic text.
