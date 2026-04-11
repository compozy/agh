---
status: resolved
file: internal/cli/helpers_test.go
line: 19
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:f109374d0bad
review_hash: f109374d0bad
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 006: Add compile-time interface verification for stubClient.
## Review Comment

Given this stub now tracks more `DaemonClient` methods, add an explicit compile-time assertion so future interface changes fail fast in tests.

As per coding guidelines, "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

Also applies to: 67-100

## Triage

- Decision: `valid`
- Root cause: `stubClient` intentionally mirrors `DaemonClient`, but there is no compile-time assertion to catch interface drift when new methods are added.
- Fix approach: add a `var _ DaemonClient = stubClient{}` assertion in the test helper.
