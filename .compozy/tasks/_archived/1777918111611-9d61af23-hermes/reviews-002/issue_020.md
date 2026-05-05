---
status: resolved
file: internal/session/crash_bundle_test.go
line: 27
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4175824057,nitpick_hash:a8e2bd0fad1b
review_hash: a8e2bd0fad1b
source_review_id: "4175824057"
source_review_submitted_at: "2026-04-25T16:07:33Z"
---

# Issue 020: Use timestamp-derived assertions instead of hardcoded "101"/"202" substrings.
## Review Comment

These checks are a bit brittle and less explicit. Derive expected timestamp tokens from `first`/`second` so the assertion tracks the actual test inputs.

As per coding guidelines, "MUST test meaningful business logic, not trivial operations".

## Triage

- Decision: `valid`
- Root cause: the crash-bundle filename test checks hardcoded `"101"` and `"202"` substrings instead of deriving expected timestamp tokens from the test inputs.
- Fix approach: compute expected suffix tokens from `first.UnixNano()` and `second.UnixNano()` so the assertion tracks the actual timestamps under test.
