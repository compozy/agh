---
status: resolved
file: internal/procutil/process_group_windows.go
line: 10
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:f634e0c5e6a9
review_hash: f634e0c5e6a9
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 023: Windows implementation violates the documented blocking contract for process group shutdown.
## Review Comment

`WaitForProcessGroupIDExit` returns success immediately without waiting, and `KillProcessGroupIDAndWait` only signals without waiting for the process group to exit. The Unix implementation polls until the group is gone or the deadline is exceeded. Callers expect blocking semantics after `KillProcessGroupIDAndWait` completes—they proceed with cleanup, restart, and port reuse with the assumption that the process tree has been torn down. This implementation allows child processes to remain running while callers proceed, creating races. Return an explicit unsupported error or implement actual process group termination and wait behavior instead of returning success prematurely.

## Triage

- Decision: `valid`
- Root cause: the Windows process-group implementation reports success from `WaitForProcessGroupIDExit` without observing process exit, and `KillProcessGroupIDAndWait` returns after signaling a single process. That violates the blocking contract used by shutdown/restart cleanup code.
- Fix approach: because real Windows process-group parity is not implemented in this file, return an explicit unsupported process-group error from the Windows group wait/kill paths instead of reporting successful teardown. This preserves the contract by preventing callers from assuming descendants are gone.
