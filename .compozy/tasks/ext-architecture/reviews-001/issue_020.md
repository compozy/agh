---
status: resolved
file: internal/extension/host_api.go
line: 408
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaY,comment:PRRC_kwDOR5y4QM62zlsj
---

# Issue 020: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard the remaining session handlers against a nil session manager.**

`handleSessionsList` and `handleSessionsCreate` validate `h.sessions`, but `handleSessionsStop`, `handleSessionsStatus`, and `handleSessionsEvents` call through it unconditionally. A partially configured handler will panic on these methods instead of returning a clean RPC error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 349 - 408, Add a nil-check for
h.sessions at the start of handleSessionsStop, handleSessionsStatus, and
handleSessionsEvents and return a proper RPC error instead of dereferencing a
nil pointer; e.g., if h.sessions == nil { return nil,
internalRPCError(errors.New("sessions manager is not configured")) } so these
handlers fail cleanly when the session manager is not initialized.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `handleSessionsStop`, `handleSessionsStatus`, and `handleSessionsEvents` dereference `h.sessions` without the same guard already present in `handleSessionsList`, `handleSessionsCreate`, and `submitPrompt`. A partially configured handler would panic instead of returning a clean error.
  Fix approach: add the missing `h.sessions == nil` guard in all three handlers and cover the behavior with Host API handler tests.
