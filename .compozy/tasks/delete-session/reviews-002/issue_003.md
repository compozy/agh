---
status: resolved
file: internal/daemon/daemon_test.go
line: 4163
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:1d95a04a1c96
review_hash: 1d95a04a1c96
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 003: Make Delete able to model not-found behavior in tests.
## Review Comment

This fake currently returns `nil` even when the session ID does not exist, which can mask negative-path behavior in callers.

## Triage

- Decision: `valid`
- Notes:
  - The `fakeSessionManager.Delete` implementation in `internal/daemon/daemon_test.go` currently returns `nil` even when no matching session exists, unlike the real session manager which reports `session.ErrSessionNotFound`.
  - That divergence can hide negative-path behavior in daemon tests that depend on delete semantics.
  - Planned fix: make the fake return `session.ErrSessionNotFound` when no in-memory session matches the requested id.

## Resolution

- Updated `fakeSessionManager.Delete` to return `session.ErrSessionNotFound` when the requested session ID is absent from the fake's in-memory state.
- The fake now matches real delete semantics on the negative path, which prevents daemon tests from silently masking not-found behavior.
- Verified with `make verify` (exit `0`).
