---
status: resolved
file: internal/scheduler/scheduler_channel_test.go
line: 15
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:4b4b45f7e73c
review_hash: 4b4b45f7e73c
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 017: Add an empty-channel regression case for channel-bound runs.
## Review Comment

Current tests validate matching and mismatching channels, but they don’t guard the critical edge case where a session has `Channel == ""` while the run is channel-bound.

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling" and tests should ensure behavior regressions are caught.

## Triage

- Decision: `VALID`
- Notes: Existing scheduler channel tests cover matching and wrong-channel sessions but not the critical edge case where the session channel is empty and the run is channel-bound. Add a `Should ...` regression subtest for that no-match case.
