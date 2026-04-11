---
status: resolved
file: internal/extension/host_api.go
line: 745
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbs,comment:PRRC_kwDOR5y4QM624L_b
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't fail a successful prompt on the first post-submit read.**

This path polls later for seed events, but `TurnID` is still discovered with one immediate `Events` query. If the prompt is accepted before the user-message event is persisted, this returns `"prompt turn id not found"` for a valid submission. Fold turn-id discovery into the same polling loop, or wait until the first post-submit user-message arrives before failing.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 725 - 745, The current code
finds turnID immediately from a single h.sessions.Events call and errors if not
found; instead, fold turnID discovery into the existing post-submit polling loop
so we wait for the first post-submit user-message to appear before failing.
Replace the one-off Events lookup and the immediate error return with logic
inside the polling loop that repeatedly calls h.sessions.Events(ctx, sessionID,
store.EventQuery{AfterSequence: lastSequence}) and scans returned events for
strings.TrimSpace(event.Type)==acp.EventTypeUserMessage and a non-empty
strings.TrimSpace(event.TurnID); set turnID when found and only return
hostAPIPromptSubmission{} with errors.New("extension: prompt turn id not found
after prompt submission") after the polling timeout/limit elapses. Ensure you
still break out early when turnID is discovered and reuse lastSequence correctly
so you don’t miss events.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Invalid`
- Notes:
  `session.Manager.Prompt` persists the `user_message` event, including the turn ID, before it calls the driver and before it returns the event channel. That means the immediate `Events(... AfterSequence:lastSequence)` lookup in `submitPrompt` cannot miss the turn ID under the actual session-manager contract used here.
  Polling for turn discovery would duplicate an existing guarantee rather than fixing a real race. Closed with no code change after tracing the prompt path in `internal/session/manager_prompt.go`.
