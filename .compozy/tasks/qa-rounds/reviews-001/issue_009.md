---
status: resolved
file: internal/network/manager_test.go
line: 751
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:004e4aeb5977
review_hash: 004e4aeb5977
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 009: These metric expectations are heartbeat-timing sensitive.
## Review Comment

The exact `sent/received` and `KindGreet` counts here assume neither session heartbeat fires before `Status()` runs. Since `JoinChannel()` starts live 1-second heartbeats, slower CI can legitimately observe extra greet traffic and fail this test intermittently. I’d make the test config use a much longer greet interval, or assert only the deltas introduced by the explicit `KindSay`.

Also applies to: 781-793

## Triage

- Decision: `VALID`
- Notes: `TestManagerStatusTracksWorkflowMetricsAndStructuredLogs` uses `testManagerConfig()`, whose one-second greet interval can emit extra heartbeat greets before `Status()` on slow CI. Fix by giving this test a longer `GreetInterval` while preserving the expected initial-greet and explicit `KindSay` metrics.
