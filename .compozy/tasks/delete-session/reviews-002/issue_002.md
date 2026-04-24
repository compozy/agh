---
status: resolved
file: internal/daemon/daemon_test.go
line: 4034
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:5ab3004a8600
review_hash: 5ab3004a8600
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 002: Add compile-time interface verification for fakeSessionManager.
## Review Comment

Since this fake now tracks the new delete surface, add an interface assertion to prevent drift as the `SessionManager` contract evolves.

As per coding guidelines, "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

---

## Triage

- Decision: `valid`
- Notes:
  - `fakeSessionManager` in `internal/daemon/daemon_test.go` is used as a transport-facing `SessionManager` fake and now carries the delete surface, but it has no compile-time assertion against the interface it is expected to satisfy.
  - That leaves the test fake vulnerable to silent interface drift as the session contract evolves.
  - Planned fix: add a compile-time interface assertion near the fake type definition.

## Resolution

- Added the compile-time assertion `var _ SessionManager = (*fakeSessionManager)(nil)` next to the fake definition in `internal/daemon/daemon_test.go`.
- This now forces the test double to stay aligned with the daemon-facing `SessionManager` contract as the interface evolves.
- Verified with `make verify` (exit `0`).
