---
status: resolved
file: internal/automation/trigger_test.go
line: 445
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:222ec5823828
review_hash: 222ec5823828
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 018: Use named subtests for these table cases.
## Review Comment

Both loops stop on the first failure and make it harder to see which scenario broke. Wrapping each case in `t.Run("Should...")` and calling `t.Parallel()` inside independent subtests would make the failures much easier to triage.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "Use `t.Parallel()` for independent subtests".

Also applies to: 479-510

## Triage

- Decision: `valid`
- Notes: The current trigger helper loops report only the first failure and do not isolate scenarios. I will convert the affected tables to named `t.Run("Should...")` subtests and mark independent cases parallel.
