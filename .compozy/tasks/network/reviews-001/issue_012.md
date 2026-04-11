---
status: resolved
file: internal/daemon/daemon_test.go
line: 281
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:9b163ed6b08a
review_hash: 9b163ed6b08a
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 012: Avoid matching the boot failure by substring.
## Review Comment

Line 282 couples this test to error wording rather than the actual contract. Please prefer a sentinel/typed boot error and assert it with `errors.Is`/`errors.As`; otherwise harmless message rewording will break the test.

As per coding guidelines, "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings".

## Triage

- Decision: `valid`
- Root cause: the test currently matches the missing binding-surface failure by substring because the production path returns an untyped error.
- Fix approach: introduce a stable sentinel for the missing network binding surface and assert it with `errors.Is` in the test.
- Scope note: this required one minimal production edit in `internal/daemon/boot.go` because the boot-time error is emitted from `bootNetwork`, not from `daemon.go`.
