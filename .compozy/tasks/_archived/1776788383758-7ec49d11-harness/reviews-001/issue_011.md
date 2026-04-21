---
status: resolved
file: internal/daemon/daemon_test.go
line: 3634
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:d9482f1ac45f
review_hash: d9482f1ac45f
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 011: Reuse one turn ID/timestamp for the synthetic event record.
## Review Comment

`recordSyntheticEvent` writes different `TurnID`/`Timestamp` values into the JSON payload and the outer `store.SessionEvent`. Any test that decodes `Content` can now observe an impossible event shape for a single row, which makes the fake less trustworthy for transcript/harness assertions.

## Triage

- Decision: `valid`
- Notes:
  - `recordSyntheticEvent` currently marshals an inner synthetic event with one `TurnID`/`Timestamp` pair and then writes a different `TurnID`/`Timestamp` on the outer `store.SessionEvent`.
  - Any test that decodes `Content` can therefore observe an impossible row where the payload and wrapper disagree about the same event.
  - I will generate one shared synthetic turn identifier and timestamp per fake event record and reuse them in both places.
