---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: pending
file: internal/testutil/acpmock/cmd/acpmock-driver/main.go
line: 39
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYarH,comment:PRRC_kwDOR5y4QM6-7HZx
---

# Issue 032: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>❓ Verification inconclusive</summary>

In the ACP protocol, are session config option values scoped to an individual session or shared across all sessions of one agent process?

---

In the Agent Client Protocol (ACP), session config option values are scoped to an individual session, not shared across all sessions of one agent process.[1][2][3]

Citations:

- 1: https://agentclientprotocol.com/protocol/session-config-options
- 2: https://crystaldoc.info/github/hahwul/acp.cr/main/ACP/Session.html
- 3: https://crystaldoc.info/github/hahwul/acp.cr/main/ACP/Protocol/SessionSetConfigOptionParams.html

---



</details>

**Move `configOptions` into session state; scope config per session not agent.**

The ACP protocol scopes session config option values to individual sessions.[1] Storing `configOptions` on `mockAgent` and mutating it via `SetSessionConfigOption` (line 208–223) without checking session ID causes config changes from one session to leak into all subsequent sessions created by this agent. `ResumeSession` cannot surface the current config because it returns an empty response (line 169–183). Multi-session tests will pass or fail for the wrong reason. Create a sessionState struct or map (keyed by session ID) to store config per session, and populate it in `NewSession` / `LoadSession` (lines 154–166) / `ResumeSession`.

[1]: https://agentclientprotocol.com/protocol/session-config-options

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/testutil/acpmock/cmd/acpmock-driver/main.go` around lines 35 - 39,
The code stores configOptions on mockAgent causing session config to leak across
sessions; create a per-session state (e.g., sessionState struct with
configOptions field) and a map on mockAgent keyed by sessionID to hold these
states, update NewSession, LoadSession and ResumeSession to allocate or load the
sessionState (populate configOptions from the session data), and change
SetSessionConfigOption to look up and mutate the configOptions inside the
correct sessionState using the provided session ID rather than mutating
mockAgent.configOptions so each session keeps its own config.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
