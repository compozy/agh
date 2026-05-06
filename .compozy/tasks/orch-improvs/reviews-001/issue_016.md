---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/task_event_bridge_notifier.go
line: 144
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:f97df580b9b5
review_hash: f97df580b9b5
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 016: Avoid doing bridge delivery inline on the task event callback path.
## Review Comment

`DeliverDue` performs store reads and outbound bridge delivery, and `taskEventObserverFanout` calls observers serially. A slow or unavailable bridge can therefore hold the task event publisher for up to the timeout on every wake event. This should be offloaded to a bounded worker/queue instead of running inline.

## Triage

- Decision: `valid`
- Notes:
  - `taskEventObserverFanout` still invokes observers serially, and the bridge observer performs store reads plus outbound delivery inline on that path.
  - A slow/unavailable bridge therefore stalls task event publication for the full delivery timeout on each wake event.
  - Planned fix: offload bridge delivery to bounded daemon-owned async processing with explicit shutdown, while preserving wake coalescing semantics and adding targeted tests.
  - Resolved: bridge delivery now runs through a bounded async observer/worker with explicit shutdown and wake coalescing, and tests verify the callback returns without waiting on slow delivery.
