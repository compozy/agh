---
status: resolved
file: internal/automation/validate_test.go
line: 460
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:40a86a91c825
review_hash: 40a86a91c825
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 019: Assert which validation failed, not just that something failed.
## Review Comment

These cases only check `err != nil`, so they still pass if validation breaks for an unrelated field or rule. Please carry a `wantErr`/field path per case and assert it, like the earlier table-driven sections in this file already do.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

Also applies to: 621-732, 734-787

## Triage

- Decision: `valid`
- Notes: Several validation tables only assert that an error occurred, not which rule failed, so unrelated regressions can still pass. I will add explicit expected substrings/paths for the affected job, trigger, run, and envelope validation cases.
