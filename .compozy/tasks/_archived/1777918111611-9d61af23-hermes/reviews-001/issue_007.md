---
status: resolved
file: internal/cli/lifecycle.go
line: 201
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:331e552db8ec
review_hash: 331e552db8ec
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 007: Treat “already exited” as success during uninstall.
## Review Comment

There’s a race between `daemonInfo(...running)` and `signalProcess`. If the daemon exits in that window, `SIGTERM` can return `ESRCH`/`os.ErrProcessDone` and uninstall aborts even though the target state is already satisfied. This should continue to artifact cleanup instead of failing the command.

## Triage

- Decision: `VALID`
- Notes: `stopDaemonForUninstall` treats every `signalProcess` error as fatal after `daemonInfo` reports a running process. If the daemon exits between those calls, `os.ErrProcessDone` or `ESRCH` means the desired stopped state is already reached and uninstall should continue artifact cleanup.
