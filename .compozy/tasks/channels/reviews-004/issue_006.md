---
status: resolved
file: internal/daemon/daemon_test.go
line: 3195
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093927386,nitpick_hash:7214aa9dc03b
review_hash: 7214aa9dc03b
source_review_id: "4093927386"
source_review_submitted_at: "2026-04-11T15:47:00Z"
---

# Issue 006: Wrap marker-recording failures with operation context.
## Review Comment

These helpers currently bubble raw marshal / append errors. When the helper process exits, that leaves stderr without enough context to tell whether initialize encoding, delivery encoding, or file append failed.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `valid`
- Notes:
  - `recordInitialize`, `recordDelivery`, and `appendMarkerLine` currently return raw marshal/open/append failures without operation context.
  - When the helper subprocess exits on one of those errors, stderr does not say whether initialize recording, delivery recording, or marker-file append failed.
  - Fix approach: wrap these failures with method-specific context and add direct tests that assert the contextual error messages.

## Resolution

- Wrapped initialize-marker, delivery-marker, and marker-file append failures with operation-specific context.
- Added regression tests that force marker append failures and assert the contextual error strings.
- Verified with `go test ./internal/daemon` and `make verify`.
