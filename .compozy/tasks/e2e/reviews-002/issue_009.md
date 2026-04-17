---
status: resolved
file: internal/testutil/acpmock/registration.go
line: 47
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130477247,nitpick_hash:2ecf143d824b
review_hash: 2ecf143d824b
source_review_id: "4130477247"
source_review_submitted_at: "2026-04-17T16:34:12Z"
---

# Issue 009: Wrap pass-through errors in Register with local operation context.
## Review Comment

A few returns pass errors through directly, which makes root-cause tracing harder from this call site.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

Also applies to: 59-62, 69-77

## Triage

- Decision: `valid`
- Root cause: several `Register()` failure paths pass helper errors through unchanged. When registration fails inside an E2E harness, the missing local operation context makes it harder to see whether the failure came from fixture loading, fixture-agent lookup, driver resolution, or diagnostics path setup.
- Fix plan: wrap pass-through errors in `Register()` with operation-specific context while preserving the original error via `%w`.
- Test impact: requires a small assertion update in `internal/testutil/acpmock/fixture_test.go`.
- Resolution: implemented. `Register()` now wraps fixture load, fixture-agent lookup, driver resolution, and diagnostics path failures with local operation context, and the acpmock tests assert the new context on representative failures.
- Verification: `go test ./internal/testutil/acpmock`, `make verify`.
