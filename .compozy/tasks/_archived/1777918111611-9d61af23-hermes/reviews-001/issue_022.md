---
status: resolved
file: internal/observe/health.go
line: 140
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:277b97af3896
review_hash: 277b97af3896
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 022: Include failure-health degradation in the top-level status.
## Review Comment

`collectFailureHealth` can mark the failures section as degraded, but the overall `Health.Status` ignores it. That means the daemon can report top-level `"ok"` while simultaneously surfacing persisted lifecycle failures.

## Triage

- Decision: `valid`
- Root cause: `collectFailureHealth` correctly marks `FailureHealth.Status` as `degraded` when persisted lifecycle failures exist, but `Observer.Health` only derives the top-level status from persistence and agent probe health. A daemon with stored failures and no unhealthy probes can therefore report `"ok"` at the top level while the failures section is degraded.
- Fix approach: include `failureHealth.Status` in the top-level `observeHealthStatus(...)` aggregation and add coverage for the failure-only path so the top-level status degrades even without probe failures.
