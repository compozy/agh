---
status: resolved
file: internal/cli/daemon_test.go
line: 1551
severity: medium
author: claude-reviewer
---

# Issue 108: fakeDaemonClient in tests is not safe for concurrent use



## Review Comment

The `fakeDaemonClient` struct at line 1551 has many mutable fields that are written to during method calls (e.g., `c.spawnSessionRef = sessionRef`, `c.statusRef = ref`, `c.readStatusCalls++`). These fields have no synchronization (no mutex). While each test currently creates its own `fakeDaemonClient` instance, several tests use `t.Parallel()` at the subtest level with a shared `deps` struct that references the same client.

For example, in the messaging and state tests, the parent test sets up `deps` and then spawns parallel subtests that create their own clients. This pattern is currently safe because each subtest creates a new `fakeDaemonClient`. However, it is fragile -- if any future test shares a `fakeDaemonClient` across goroutines, the race detector will catch it.

The tests run with `-race`, so if any future test shares a `fakeDaemonClient` across goroutines, the race detector will catch it. But as a defensive measure, consider documenting that `fakeDaemonClient` is not goroutine-safe, or adding a mutex to the shared state fields.

## Triage

- Decision: `invalid`
- Notes: The review describes a hypothetical future race, not a current defect. The existing tests instantiate separate `fakeDaemonClient` values per parallel subtest; I did not find a concurrent shared use of one client instance in the current suite. Adding locking purely for speculative future misuse would be a defensive redesign, not a root-cause fix for a reproduced bug in this batch.
