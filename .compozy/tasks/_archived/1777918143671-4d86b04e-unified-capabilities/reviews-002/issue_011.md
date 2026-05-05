---
status: resolved
file: internal/daemon/daemon_test.go
line: 4196
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:aabb9109459a
review_hash: aabb9109459a
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 011: ClearConversation fallback is currently unreachable for missing sessions.
## Review Comment

At Line 4201-Line 4203, any `Status` error returns immediately, so the fallback at Line 4205 never handles not-found cases. This can break tests expecting clear-conversation to succeed for absent sessions.

As per coding guidelines, "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings".

## Triage

- Decision: `valid`
- Root cause: `fakeSessionManager.ClearConversation` returns immediately on any `Status` error, so its nil-info fallback can never handle `session.ErrSessionNotFound`.
- Fix plan: treat `session.ErrSessionNotFound` as the missing-session case in the fake and add a regression test in `internal/daemon/daemon_test.go` so daemon-layer tests keep the fake aligned with the real clear-conversation contract.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
