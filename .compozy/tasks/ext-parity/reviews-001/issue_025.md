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

# Issue 025: Strengthen these negative-path assertions.
## Review Comment

These cases mostly pass on “any non-nil error”, so the tests stay green even if the wrong validation branch fails. Please assert the expected error type/message per case so the suite actually pins the business rule being exercised.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

Also applies to: 187-193, 492-533

## Triage

- Decision: `VALID`
- Notes: The invalid bridge codec tests still accept any non-nil error for several negative paths, so they do not prove that scope binding, malformed JSON, DM policy, delivery defaults, or manifest metadata validation failed for the expected reason. The fix is to assert the relevant sentinel or identifying error text for each case so the suite pins the intended branch.
