---
status: resolved
file: internal/cli/client_test.go
line: 411
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:7206e46da4a3
review_hash: 7206e46da4a3
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 023: Split the automation client E2E test into focused t.Run("Should...") cases.
## Review Comment

This test exercises a large slice of the client surface in one flow, so the first failure hides the rest of the regressions and makes the failing capability harder to isolate. Breaking it up by operation (`Should list jobs`, `Should create job`, `Should list runs`, etc.) will keep failures actionable and aligns with the repo's test conventions.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use `t.Run("Should...")` pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes: `TestUnixSocketClientAutomationMethods` exercises many client operations in one flow, which hides follow-on failures once the first assertion trips. I will split it into focused `Should...` subtests while preserving the shared request/response coverage.
