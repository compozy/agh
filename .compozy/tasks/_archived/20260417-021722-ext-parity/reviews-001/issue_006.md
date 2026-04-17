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

# Issue 006: Wrap launcher failures with agent/process context.
## Review Comment

This returns the launcher error verbatim, which drops the agent and command that failed to start. Wrapping it here would make startup failures much easier to triage.

As per coding guidelines, `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

## Triage

- Decision: `VALID`
- Notes: `launchAgentProcess` currently returns `launcher.Launch(...)` errors unchanged, so startup failures lose the agent/command context that is needed to diagnose which subprocess failed. The fix is to wrap the launcher error at the call site and add regression coverage.
