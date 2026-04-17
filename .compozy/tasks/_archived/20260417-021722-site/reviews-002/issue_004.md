---
status: resolved
file: internal/acp/client.go
line: 181
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:a89043defd8e
review_hash: a89043defd8e
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 004: Wrap launcher failures with agent/process context.
## Review Comment

This returns the launcher error verbatim, which drops the agent and command that failed to start. Wrapping it here would make startup failures much easier to triage.

As per coding guidelines, `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

## Triage

- Decision: `VALID`
- Root cause: `spawnProcess` wraps launcher failures with the raw command string only. When startup fails, the error does not include the ACP agent name, which makes multi-agent startup failures harder to identify.
- Fix approach: Wrap the launch error with both agent and command context while preserving the original cause.

## Resolution

- Wrapped ACP launcher failures in [internal/acp/client.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/acp/client.go) with agent and command context.
- Added a regression test in [internal/acp/client_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/acp/client_test.go) to assert the enriched launch error.
