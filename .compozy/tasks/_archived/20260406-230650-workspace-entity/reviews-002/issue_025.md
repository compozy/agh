---
status: resolved
file: internal/udsapi/handlers.go
line: 98
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCz,comment:PRRC_kwDOR5y4QM61T6IU
---

# Issue 025: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`workspace_id` / `workspace_path` are never populated on session events.**

`sessionEventPayloadFromEvent()` still only copies the persisted event fields, and the synthetic `session_stopped` event in `streamSession()` also leaves these new fields empty. As written, `/sessions/:id/events`, `/history`, and the SSE stream won't actually expose the workspace context this struct advertises.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/udsapi/handlers.go` around lines 87 - 98, sessionEventPayload's
WorkspaceID and WorkspacePath are never set because sessionEventPayloadFromEvent
currently only copies persisted event fields and streamSession's synthetic
"session_stopped" event doesn't populate them; update
sessionEventPayloadFromEvent to copy WorkspaceID and WorkspacePath from the
source event (or its associated session/context) into the sessionEventPayload
struct, and ensure streamSession sets WorkspaceID and WorkspacePath on the
synthetic session_stopped payload using the same session/context values so SSE
and /sessions/:id/events include workspace context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `sessionEventPayload` exposes `workspace_id` and `workspace_path`, but the
  handlers only copy persisted event fields and the synthetic
  `session_stopped` SSE payload also omits workspace context. `store.SessionEvent`
  does not carry workspace metadata, so the handler layer must attach it from
  `SessionInfo`. Plan: load `SessionInfo` once per request/stream and populate
  workspace fields on event/history/SSE payloads.
