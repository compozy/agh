---
status: resolved
file: internal/store/globaldb/global_db_test.go
line: 991
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:587a1484a8c9
review_hash: 587a1484a8c9
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 030: Round-trip the other new liveness fields too.
## Review Comment

This fixture now populates `LastUpdateAt` and `StallState`, but the assertions only verify `SubprocessPID` and `StallReason`. A regression in the `last_update_at` or `stall_state` read/write path would still pass this test.

Also applies to: 1038-1046

## Triage

- Decision: `valid`
- Notes:
  The session registration round-trip test writes `LastUpdateAt` and `StallState`, but only checks `SubprocessPID` and `StallReason`. That leaves part of the new liveness persistence contract unverified.
  I will assert the persisted `LastUpdateAt` timestamp and `StallState` value alongside the existing liveness checks.
  Fixed and verified with targeted package tests plus `make verify`.
