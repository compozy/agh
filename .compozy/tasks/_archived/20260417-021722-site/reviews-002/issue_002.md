---
status: resolved
file: extensions/bridges/teams/provider_test.go
line: 1255
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:bcca49a3ba22
review_hash: bcca49a3ba22
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 002: Wait for ready state in the polling predicate to reduce flakiness.
## Review Comment

This currently waits for “any state” and asserts readiness afterward. If an intermediate non-ready state is written first, the test can fail intermittently.

## Triage

- Decision: `VALID`
- Root cause: The test polls only for `runtime.configForInstance("brg-1")` to succeed, then immediately dereferences `runtime.currentSession()`. Initialization is asynchronous, so the config can be installed before the session pointer is published, which makes the follow-on assertion racy.
- Fix approach: Tighten the polling predicate so it waits for the readiness condition the test actually depends on, not just partial initialization.

## Resolution

- Updated [extensions/bridges/teams/provider_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/extensions/bridges/teams/provider_test.go) so the readiness poll waits for both bridge config registration and a non-nil current session before asserting downstream behavior.
