---
status: resolved
file: internal/session/prompt_activity_test.go
line: 214
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4175824057,nitpick_hash:fd35c0b0c1d3
review_hash: fd35c0b0c1d3
source_review_id: "4175824057"
source_review_submitted_at: "2026-04-25T16:07:33Z"
---

# Issue 021: Expand this test to lock in all timeoutStopDeadline() branches.
## Review Comment

Line 225 currently validates only the configured positive-grace path. The method also guarantees fallback behavior for nil supervisor and non-positive grace; adding those as table-driven subtests will make regressions much harder.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests, and MUST test meaningful business logic.

## Triage

- Decision: `valid`
- Root cause: `TestPromptActivitySupervisorTimeoutStopDeadline` only covers the positive configured grace branch and does not lock in nil-supervisor or non-positive-grace fallback behavior.
- Fix approach: convert the test into table-driven `Should...` subtests that cover nil supervisor, zero grace, negative grace, and positive configured grace.
