---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1034
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107556463,nitpick_hash:047960890d8a
review_hash: 047960890d8a
source_review_id: "4107556463"
source_review_submitted_at: "2026-04-14T16:23:06Z"
---

# Issue 003: Mark these new subtests as parallel.
## Review Comment

Each subtest spins up its own isolated runtime, so they look independent. Adding `t.Parallel()` here keeps the integration suite aligned with the repo test rules and makes race coverage more useful.

As per coding guidelines, `**/*_test.go`: Use `t.Parallel()` for independent subtests in Go tests.

## Triage

- Decision: `valid`
- Notes:
  The three new subtests in `TestHTTPTaskRunLifecycleRoutesRoundTrip` each create their own isolated integration runtime and do not share mutable test state. They should participate in parallel execution like the rest of the suite.
  Root cause: the new subtests were added without `t.Parallel()`.
  Planned fix: mark each new subtest as parallel.

## Resolution

- Added `t.Parallel()` to each independent subtest in `TestHTTPTaskRunLifecycleRoutesRoundTrip` so the new task lifecycle integration cases follow the repo’s parallel-test convention.
