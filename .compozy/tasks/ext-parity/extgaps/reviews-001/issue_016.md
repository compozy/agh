---
status: resolved
file: internal/extension/install_managed_test.go
line: 128
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:bbe93d9eb934
review_hash: bbe93d9eb934
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 016: Add a compile-time assertion for the test double.

## Review Comment

`recordingManagedInstallRegistry` is clearly meant to implement `managedInstallRegistry`, but that contract is only checked indirectly right now. A compile-time assertion here will fail fast if the interface changes.

As per coding guidelines, `**/*.go`: `Use compile-time interface verification: var _ Interface = (*Type)(nil)`.

## Triage

- Decision: `valid`
- Root cause: `recordingManagedInstallRegistry` is intended to satisfy `managedInstallRegistry`, but the file does not declare that contract explicitly. Interface drift would only surface indirectly at call sites instead of failing fast in the test package.
- Fix plan: add a compile-time interface assertion for the test double.
- Resolution: added a compile-time interface assertion for `recordingManagedInstallRegistry`.
- Verification: updated `internal/extension/install_managed_test.go` and passed `go test ./internal/extension` plus `make verify`.
