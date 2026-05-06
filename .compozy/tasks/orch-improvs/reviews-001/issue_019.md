---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/session/query_test.go
line: 595
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:9472c0edc9ae
review_hash: 9472c0edc9ae
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 019: Make the finalization wait assertion deterministic.
## Review Comment

The `time.After(50 * time.Millisecond)` window can pass even if `openQueryRecorder` never waits; the goroutine may simply not be scheduled before the timeout. The final assertion also only compares event counts. Add a handshake that confirms the goroutine has entered the call before unblocking finalization, then compare event sequences/IDs against `activeEvents` so this test fails on premature returns instead of scheduler variance.

Also applies to: 631-636

## Triage

- Decision: `valid`
- Notes:
  - The current finalization-wait test still uses `time.After(50 * time.Millisecond)` as a proxy for “goroutine reached the wait,” which is scheduler-dependent.
  - The final assertion only compares counts, so a premature return with the wrong event sequence can still pass.
  - Planned fix: add a deterministic handshake around the recorder open/query path and assert the returned event sequence/IDs against the active recorder data.
  - Resolved: the test now uses a deterministic start handshake before unblocking finalization and compares returned event IDs/sequences instead of only counting events.
