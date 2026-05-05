---
status: resolved
file: internal/daemon/task_runtime.go
line: 517
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:04d09de0ab19
review_hash: 04d09de0ab19
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 012: Don't classify orphaned/stalled sessions from PID liveness alone.
## Review Comment

These branches only check `procutil.Alive(liveness.SubprocessPID)`. After a reboot, PID reuse can make an unrelated process look like the old session subprocess, so the new recovery metadata can report `orphaned` or `stalled` for the wrong process. You already persist `SubprocessStartedAt`; use it to confirm process identity before emitting those classifications.

## Triage

- Decision: `valid`
- Root cause: `classifyRecoveredTaskSession` treats `procutil.Alive(pid)` as proof that the stored subprocess still exists. That is not sufficient after PID reuse because the code never checks whether the live PID has the recorded `SubprocessStartedAt`.
- Fix plan: add a minimal process-start-time helper in `internal/procutil` and use it here to confirm `(pid, started_at)` identity before emitting `orphaned` or `stalled`. Add regression coverage in `internal/daemon/task_runtime_test.go` for mismatched start times.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
