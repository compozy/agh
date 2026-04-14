---
status: resolved
file: internal/config/config_test.go
line: 753
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106850065,nitpick_hash:4b6213b167a1
review_hash: 4b6213b167a1
source_review_id: "4106850065"
source_review_submitted_at: "2026-04-14T14:43:27Z"
---

# Issue 004: Global slog state mutation prevents parallel execution.
## Review Comment

The `ShouldWarnForHTTPBaseURL` subtest manipulates `slog.Default()`, which is global state. This is intentional for testing warning output but means the parent test cannot use `t.Parallel()`. Consider adding a comment explaining why parallel execution is not used here.

## Triage

- Decision: `valid`
- Notes: The `ShouldWarnForHTTPBaseURL` subtest intentionally mutates `slog.Default()`, which is process-global state. The parent test therefore must remain non-parallel, and the lack of an inline note makes accidental `t.Parallel()` refactors more likely. I will add a short explanatory comment in `internal/config/config_test.go`.
- Resolution: Added an inline comment in `internal/config/config_test.go` documenting why the parent test must stay non-parallel.
- Verification: `go test ./internal/config`; `make verify`
