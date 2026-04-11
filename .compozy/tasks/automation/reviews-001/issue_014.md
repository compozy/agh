---
status: resolved
file: internal/automation/manager_test.go
line: 1024
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:a34415f6d11e
review_hash: a34415f6d11e
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 014: Make the nil-context assertions specific.
## Review Comment

Right now any error satisfies these branches, so a validation or persistence regression would still look like a passing nil-context test. Please match the sentinel or message the manager is expected to return for nil contexts.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Notes: The nil-context assertions in `manager_test.go` only require a non-nil error, so unrelated regressions can satisfy them. I will assert the specific context-required error text for each affected manager method.
