---
status: resolved
file: internal/registry/installer_test.go
line: 223
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106850065,nitpick_hash:943104bf6461
review_hash: 943104bf6461
source_review_id: "4106850065"
source_review_submitted_at: "2026-04-14T14:43:27Z"
---

# Issue 011: Replace the sleep-based cancellation trigger with explicit synchronization.
## Review Comment

Lines 230-233 rely on `time.Sleep(50 * time.Millisecond)` to guess when the install path has started reading. That makes this test timing-sensitive and prone to flakes under CI load. It would be more stable to have `blockingReadCloser` signal when `Read` is entered and cancel from that signal instead.

As per coding guidelines, Never use `time.Sleep()` in orchestration — use proper synchronization primitives.

## Triage

- Decision: `valid`
- Notes: `TestInstallerInstallWithContextCancellationClosesReaderAndCleansUp()` uses `time.Sleep()` to guess when the read path has started, which makes the test timing-sensitive and violates the repository rule against sleep-based orchestration. I will add explicit read-start synchronization to the stub reader and remove the sleep from `internal/registry/installer_test.go`.
- Resolution: Reworked `blockingReadCloser` in `internal/registry/installer_test.go` to signal when `Read()` starts and updated the cancellation test to synchronize on that signal instead of sleeping.
- Verification: `go test ./internal/registry/...`; `make verify`
