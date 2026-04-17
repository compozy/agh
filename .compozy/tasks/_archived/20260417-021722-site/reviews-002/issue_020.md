---
status: resolved
file: internal/bridges/managed_sync.go
line: 168
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123260080,nitpick_hash:553c424a1e0d
review_hash: 553c424a1e0d
source_review_id: "4123260080"
source_review_submitted_at: "2026-04-16T17:55:39Z"
---

# Issue 020: Wrap source validation errors with method context.
## Review Comment

Line 169 returns `normalizedSource.Validate()` directly, which loses caller context in logs and error chains.

As per coding guidelines, "`**/*.go`: Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `VALID`
- Root cause: `validateSyncInputs` returns `normalizedSource.Validate()` directly in [internal/bridges/managed_sync.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/bridges/managed_sync.go#L124), which drops the method context from the eventual error chain.
- Fix approach: Wrap source-validation failures with `validateSyncInputs` context while preserving the original error.

## Resolution

- Wrapped source-validation failures with caller context in [internal/bridges/managed_sync.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/bridges/managed_sync.go) and added regression coverage in [internal/bridges/managed_sync_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/bridges/managed_sync_test.go).
- Full-repository verification also exposed a flaky ack snapshot assertion in [internal/bridges/delivery_broker_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/bridges/delivery_broker_test.go); that test now waits for the broker snapshot to reflect the second ack before asserting, which stabilized the required `make verify` gate without changing production behavior.
