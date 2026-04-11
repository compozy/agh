---
status: resolved
file: internal/automation/trigger_integration_test.go
line: 94
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:ea8085e6ed08
review_hash: ea8085e6ed08
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 017: Guard promptCalls() before indexing it.
## Review Comment

Both assertions read `[0]` without first checking the slice length. If dispatch creates the session but never reaches `Prompt`, the test fails with an index panic instead of a clear assertion.

Also applies to: 160-165

## Triage

- Decision: `valid`
- Notes: These integration assertions index `promptCalls()[0]` without proving a prompt call exists first, which turns a missing dispatch into a panic instead of a clear failure. I will guard the slice length before indexing in both affected tests.
