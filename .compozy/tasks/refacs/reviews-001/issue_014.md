---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/network_stub.go
line: 89
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:f2fdd2285b94
review_hash: f2fdd2285b94
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 014: Hoist the default WaitInbox error into a sentinel.
## Review Comment

Allocating a fresh `errors.New(...)` here prevents callers from using `errors.Is()` for error comparison. This violates the guideline to use `errors.Is` and `errors.As` only for error type checking. The codebase establishes sentinel error pattern (e.g., `var errXxx = errors.New(...)`) across all packages — apply the same pattern here.

## Triage

- Decision: `VALID`
- Notes:
  The default `WaitInbox` path allocates a new error every call, which blocks `errors.Is` matching and is inconsistent with repo sentinel-error discipline. Hoisting the default error to a package sentinel is the correct fix.
