---
status: resolved
file: internal/bridges/resource_test.go
line: 84
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:320f6c1aa739
review_hash: 320f6c1aa739
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 022: Strengthen these negative-path assertions.
## Review Comment

These cases mostly pass on “any non-nil error”, so the tests stay green even if the wrong validation branch fails. Please assert the expected error type/message per case so the suite actually pins the business rule being exercised.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

Also applies to: 187-193, 492-533

## Triage

- Decision: `INVALID`
- Notes:
  - The reviewed file `internal/bridges/resource_test.go` is not present in this checkout.
  - The current equivalent negative-path coverage in `internal/bridges/managed_sync_test.go` already asserts expected substrings instead of only checking for any non-nil error.
  - No weaker assertion matching this comment remains in the live managed-sync tests, so this batch item is stale after the rebase/file move.
  - Result: resolved as stale after current-tree inspection; no code change required.
