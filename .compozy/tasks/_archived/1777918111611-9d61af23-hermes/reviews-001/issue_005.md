---
status: resolved
file: internal/bridgesdk/errors_test.go
line: 229
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:c757015290bc
review_hash: c757015290bc
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 005: Refactor added cases into t.Run("Should...") table-driven subtests.
## Review Comment

The new scenarios are bundled as inline assertions rather than explicit `Should...` subtests, which makes case isolation/reporting weaker and conflicts with the enforced test pattern.

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "`MUST use t.Run(\"Should...\") pattern for ALL test cases`".

Also applies to: 291-331

## Triage

- Decision: `VALID`
- Notes: Several new bridge SDK retry/error-helper assertions are grouped in monolithic test bodies. Refactor the affected tests into `t.Run("Should...")` subtests and table-driven cases so individual scenarios report independently.
