---
status: resolved
file: internal/testutil/e2e/bridges_extensions.go
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:13103d23848c
review_hash: 13103d23848c
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 024: Consider validating empty bridgeID before path construction.
## Review Comment

The `bridgePath` helper trims whitespace but doesn't return an error for empty IDs. While callers like `GetBridge` may rely on the server to reject empty paths, explicit validation would fail faster and provide clearer error messages.

## Triage

- Decision: `valid`
- Notes:
  `bridgePath("")` currently produces `/api/bridges/`, which can silently route
  bridge-instance helpers to the collection endpoint and hide a caller bug. The
  helper should validate trimmed bridge IDs and return a clear error before any
  request is issued.

## Resolution

- Added blank-ID validation in the bridge path helper and covered the guard with
  harness helper tests.
