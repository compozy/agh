---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1033
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:a77a821ada7e
review_hash: a77a821ada7e
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 007: Split the run-lifecycle matrix into subtests.
## Review Comment

This single test covers three independent flows (`complete`, `attach-session/fail`, `cancel`). When one step breaks, the rest of the route coverage disappears behind the first failure.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (t.Run) as default in Go tests.

## Triage

- Decision: `valid`
- Root cause: `TestHTTPTaskRunLifecycleRoutesRoundTrip` bundles three independent lifecycle flows into one long sequence, so the first failure hides the rest of the route coverage.
- Fix approach: split the complete, attach/fail, and cancel flows into isolated `Should...` subtests that each assert one lifecycle path end to end.

## Resolution

- Split the HTTP task-run lifecycle integration coverage into isolated `Should...` subtests for complete, attach/fail, and cancel flows.
- Verified in the final `make verify` run.
