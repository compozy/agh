---
status: resolved
file: internal/daemon/daemon.go
line: 522
severity: medium
author: claude-code
provider_ref:
---

# Issue 014: notifierFanout mutated after being passed to session manager

## Review Comment

The boot sequence creates an empty `notifierFanout{}` at line 522, passes a pointer to it as the notifier to `newSessionManager`, then mutates `fanout.notifiers` by appending the observer at line 541 *after* the session manager is created. If the session manager calls any notifier method during construction (e.g., reconciliation), events would be silently dropped.

Additionally, `notifierFanout.notifiers` is written in `boot()` and read from notifier methods that can be called from different goroutines, with no synchronization on the slice. In practice, boot completes before sessions are created, but this is a latent data race.

**Suggested fix:** Construct the fanout with all notifiers known upfront by restructuring boot order (create observer before session manager), or protect `notifierFanout.notifiers` with a mutex.

## Triage

- Decision: `invalid`
- Notes: `fanout.notifiers` is populated during boot before the session manager is exposed to external callers, and `session.NewManager()` does not emit notifications during construction. That means the reported event-loss race is not reachable in the current boot sequence. The point is architectural hygiene, not a present defect.
