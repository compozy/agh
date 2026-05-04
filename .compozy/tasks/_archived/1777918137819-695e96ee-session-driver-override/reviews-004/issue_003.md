---
status: resolved
file: internal/config/provider_test.go
line: 316
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167458928,nitpick_hash:ad57e7ebbfbf
review_hash: ad57e7ebbfbf
source_review_id: "4167458928"
source_review_submitted_at: "2026-04-24T02:26:11Z"
---

# Issue 003: Consider table-driving the ResolveSessionAgent subtests.
## Review Comment

The scenarios are well chosen, but setup is repeated across subtests. A table-driven case list would reduce duplication and make it easier to add new provider/override combinations.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

## Triage

- Decision: `invalid`
- Notes:
- `TestResolveSessionAgent` already uses descriptive parallel subtests for the three materially different provider-resolution scenarios in this file.
- Recasting those branches into a table would not increase coverage or fix incorrect behavior here; it would only compress already-clear case-specific setup into shared closure fields.
