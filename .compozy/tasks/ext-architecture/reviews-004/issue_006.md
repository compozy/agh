---
status: resolved
file: internal/api/core/handlers_internal_test.go
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093048586,nitpick_hash:f8290ab22c1d
review_hash: f8290ab22c1d
source_review_id: "4093048586"
source_review_submitted_at: "2026-04-11T01:15:37Z"
---

# Issue 006: Tighten the failure-path assertion and deduplicate these cases.
## Review Comment

The scenarios are good, but the last subtest only checks `err != nil`, so the helper could start returning the wrong wrapped error and this would still pass. Converting this into a small table-driven test and asserting the expected error text/cause in the failing row would make the coverage much more resilient.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Notes:
- `TestResolveUserHomeDir` duplicates setup across three subtests, and its failure-path branch only checks for a non-nil error instead of validating the returned error contract.
- That leaves room for a future regression in the wrapped error text or redaction behavior without a failing test.
- Fix approach: convert the cases into a table-driven test with subtests and assert the expected error content in the failing row.
