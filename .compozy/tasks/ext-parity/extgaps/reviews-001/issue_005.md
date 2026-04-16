---
status: resolved
file: internal/bridges/managed_sync_test.go
line: 103
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:24a0b5f303c5
review_hash: 24a0b5f303c5
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 005: Assert the preserved timestamp exactly.

## Review Comment

`IsZero()` still passes if the syncer overwrites `CreatedAt` with a new non-zero value. Compare it to the original persisted timestamp so this test actually pins the preservation contract it describes.

As per coding guidelines, `**/*_test.go`: Must Check: Focus on critical paths: workflow execution, state management, error handling.

## Triage

- Decision: `valid`
- Root cause: the test only checks `updated[0].CreatedAt.IsZero()`, which would still pass if the syncer overwrote `CreatedAt` with a different non-zero timestamp instead of preserving the original persisted value.
- Fix plan: assert equality with the original stored `CreatedAt` timestamp so the test pins the preservation contract exactly.
- Resolution: tightened the managed-sync update assertion to compare against the exact persisted `CreatedAt` value.
- Verification: updated `internal/bridges/managed_sync_test.go` and passed `go test ./internal/bridges` plus `make verify`.
