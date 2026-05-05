---
status: resolved
file: internal/codegen/openapits/generate.go
line: 49
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:3cbe119d01c8
review_hash: 3cbe119d01c8
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 007: Handle temporary-file cleanup errors in Check.
## Review Comment

Line 49 ignores `os.Remove` errors entirely, which can silently leak temp files and violates the project’s error-handling rule.

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

## Triage

- Decision: `valid`
- Root cause: `Check` defers `os.Remove(file.Name())` and discards any cleanup failure, which violates the repo error-handling rule and can hide leaked temp artifacts.
- Fix plan: switch `Check` to a named return and join any deferred cleanup error onto the function result without masking the primary failure.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
