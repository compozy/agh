---
status: resolved
file: cmd/agh-codegen/main.go
line: 51
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:e83f8666f561
review_hash: e83f8666f561
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 001: Wrap errors with context for better diagnostics.
## Review Comment

Errors returned from `sdkts.Generate()`, `os.MkdirAll()`, and `os.WriteFile()` lack context, making debugging harder when codegen fails.

As per coding guidelines: "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `valid`
- Notes: `writeSDKContracts` currently returns raw errors from `sdkts.Generate`, `os.MkdirAll`, and `os.WriteFile`, which drops the failing operation and path from diagnostics. I will wrap each failure with operation-specific context so codegen failures are actionable.
