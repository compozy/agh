---
status: resolved
file: internal/extension/registry.go
line: 117
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106850065,nitpick_hash:47a5979963f0
review_hash: 47a5979963f0
source_review_id: "4106850065"
source_review_submitted_at: "2026-04-14T14:43:27Z"
---

# Issue 005: Redundant options processing across Install and installWithSource.
## Review Comment

Options are applied twice: first in `Install()` (lines 121-125), then again in `installWithSource()` (lines 299-303). While this is functionally correct since options are idempotent setters, it's slightly inefficient. Consider passing the already-built config to `installWithSource` instead of re-processing opts.

## Triage

- Decision: `valid`
- Notes: `Registry.Install()` builds `installConfig` and then `installWithSource()` rebuilds the same config from the same option set. The setters are idempotent, so behavior is correct today, but the duplication is unnecessary and keeps source normalization split across two functions. I will pass the resolved config through once in `internal/extension/registry.go` while preserving current behavior.
- Resolution: `Registry.Install()` now resolves options once and forwards the built config through `installWithConfig(...)`, while preserving the old `installWithSource(...)` helper shape for package-internal tests.
- Verification: `go test ./internal/extension`; `make verify`
