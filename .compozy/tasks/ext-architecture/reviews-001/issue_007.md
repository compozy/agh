---
status: resolved
file: internal/api/spec/spec_test.go
line: 9
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:899b76342930
review_hash: 899b76342930
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 007: Consider using subtests for better test isolation and diagnostics.
## Review Comment

The test validates multiple distinct operations (list sessions, create session, approve session, write memory) but lacks `t.Run` subtests. When one assertion fails, subsequent checks are skipped, making it harder to assess overall coverage gaps.

As per coding guidelines: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes: This test exercises several unrelated OpenAPI assertions in one body, which makes failures coarse and obscures which contract regressed. I will split it into focused subtests while preserving the same coverage.
