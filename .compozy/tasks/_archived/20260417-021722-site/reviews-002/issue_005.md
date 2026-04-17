---
status: resolved
file: internal/acp/handlers_test.go
line: 691
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123260080,nitpick_hash:f1c83fc82c1c
review_hash: f1c83fc82c1c
source_review_id: "4123260080"
source_review_submitted_at: "2026-04-16T17:55:39Z"
---

# Issue 005: Missing t.Parallel() in test function.
## Review Comment

`TestNetworkTurnTerminalOwnershipGuards` does not call `t.Parallel()`, unlike most other tests in this file. This may be intentional due to environment manipulation (`t.Setenv`), but if so, a comment explaining why would be helpful.

## Triage

- Decision: `VALID`
- Root cause: `TestNetworkTurnTerminalOwnershipGuards` intentionally manipulates process-wide environment via `t.Setenv("PATH", ...)`, so leaving out `t.Parallel()` is correct, but the reason is currently implicit.
- Fix approach: Add a short comment documenting why the test must stay non-parallel rather than adding `t.Parallel()` unsafely.

## Resolution

- Added a comment in [internal/acp/handlers_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/acp/handlers_test.go) explaining that the test must remain process-serial because it mutates `PATH` with `t.Setenv(...)`.
