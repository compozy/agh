---
status: resolved
file: internal/acp/client.go
line: 239
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:08a6c5454fc0
review_hash: 08a6c5454fc0
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 001: Cancel the per-process context when registry setup fails.
## Review Comment

If `Register` fails here, the subprocess is stopped, but the `procCtx` passed into `newLocalToolHostFromPolicy` stays live. Any tool-host or terminal goroutines bound to that context can outlive the failed start. Call `cancelProcess()` before returning from this cleanup path.

## Triage

- Decision: `VALID`
- Notes: `startAgentProcess` creates `procCtx` before local tool-host setup and passes it into the `AgentProcess`. If `registerAgentProcess` fails, the cleanup path stops the process handle but never cancels `procCtx`, so goroutines tied to that process context can outlive the failed start. Fix by canceling the process context on the registration-failure cleanup path before returning.
