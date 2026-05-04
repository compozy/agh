---
status: resolved
file: internal/api/core/automation_test.go
line: 781
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:059add0f5ed9
review_hash: 059add0f5ed9
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 003: Refactor into subtests using t.Run("Should...") pattern.
## Review Comment

This test validates three distinct functions (AutomationHealthPayloadFromStatus, JobPayloadsFromJobs, RunPayloadFromRun) with sequential assertions, but lacks the required `t.Run` subtests. Split into named subtests to match the repository's mandatory test pattern:

```
t.Run("Should expose scheduler state in health payload", ...)
t.Run("Should include scheduler state in job payload", ...)
t.Run("Should expose delivery error in run payload", ...)
```

## Triage

- Decision: `VALID`
- Notes: `TestAutomationPayloadsExposeSchedulerStateAndDeliveryErrors` verifies health, job, and run payload conversion in one sequential body. The assertions are distinct behaviors and should be isolated with `t.Run("Should...")` subtests without weakening coverage.
