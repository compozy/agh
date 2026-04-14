---
status: resolved
file: internal/config/config_test.go
line: 733
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4108663639,nitpick_hash:01d5a4d1a8fc
review_hash: 01d5a4d1a8fc
source_review_id: "4108663639"
source_review_submitted_at: "2026-04-14T19:43:56Z"
---

# Issue 005: Use table-driven pattern for the validation cases; keep the HTTP warning subtest separate.
## Review Comment

This test repeats setup/assertion patterns across four cases (valid config, empty config, invalid base URL, unknown registry). Consolidate these into a table-driven loop per coding guidelines. However, keep the "ShouldWarnForHTTPBaseURL" subtest outside the loop since it manipulates `slog.Default()` and must run sequentially to avoid cross-test interference.

## Triage

- Decision: `valid`
- Root cause: `TestExtensionsConfigValidateMarketplaceConfig` repeats the same arrange/act/assert pattern across four cases instead of using the repo’s default table-driven style.
- Evidence: [`internal/config/config_test.go`](internal/config/config_test.go) lines 733-808 duplicate setup and validation logic for the non-logging cases.
- Fix plan: fold the pure validation cases into a table-driven loop and keep the `slog.Default()` warning subtest separate and sequential.
- Resolution: Refactored the repeated validation cases into a table-driven loop while keeping the logging subtest isolated. Verified with package tests and `make verify`.
