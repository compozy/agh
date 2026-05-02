---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/cli/command_paths_test.go
line: 55
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:85d9d3b43fb1
review_hash: 85d9d3b43fb1
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 019: Assert that the new authored-context commands hit their dedicated client methods.
## Review Comment

Right now this only proves the commands exit cleanly. If one of these subcommands is wired to the wrong API method, the test can still pass on zero-value output. Track call flags for the new soul/heartbeat/session methods and assert them after the loop so the matrix fails on misrouting.

As per coding guidelines "Ensure tests can fail when business logic changes".

Also applies to: 138-155, 203-235

## Triage

- Decision: `valid`
- Notes:
  - `internal/cli/command_paths_test.go` exercises the new authored-context commands but does not prove they hit their dedicated client methods.
  - Because other stubbed methods also return successful data, misrouting could still pass this smoke matrix; I added dedicated call tracking and final assertions in `internal/cli/command_paths_test.go` for the soul, heartbeat, and session authored-context command paths.
  - Verification: `make verify` passed with the stronger command-routing assertions.
