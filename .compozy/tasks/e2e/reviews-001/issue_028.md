---
status: resolved
file: internal/testutil/e2e/runtime_harness_helpers_test.go
line: 591
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:d7ccf0e9c421
review_hash: d7ccf0e9c421
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 028: Assert the error cause, not just “non-nil”.
## Review Comment

Every branch here passes on any failure path, including unrelated request-shape bugs. This helper is supposed to prove transport/decode errors are surfaced, so the assertions should check the distinguishing status/decoding context instead of only `err != nil`. As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Notes:
  The transport-error test currently passes on any non-nil error, including
  unrelated failures. These assertions should check the distinguishing status
  or decode context so the test proves the harness surfaces the intended
  transport/decode errors rather than just "something failed".

## Resolution

- Tightened the transport-error assertions to check concrete error substrings
  instead of accepting any non-nil failure.
