---
status: resolved
file: internal/agentidentity/identity.go
line: 285
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tZ,comment:PRRC_kwDOR5y4QM67Yhp9
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Populate `Model` when converting `session.Info` to `SessionSnapshot`.**

`SessionSnapshotFromInfo` never copies `info.Model`, but `internal/api/core/agent_identity.go` builds `/api/agent/me` from `caller.Session.Model`. Any caller resolved through this helper will therefore report an empty model even when the backing session has one.

<details>
<summary>Suggested fix</summary>

```diff
 	return SessionSnapshot{
 		ID:            info.ID,
 		Name:          info.Name,
 		AgentName:     info.AgentName,
 		Provider:      info.Provider,
+		Model:         info.Model,
 		WorkspaceID:   info.WorkspaceID,
 		WorkspacePath: info.Workspace,
 		Channel:       info.Channel,
 		Type:          info.Type,
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/agentidentity/identity.go` around lines 272 - 285,
SessionSnapshotFromInfo currently omits copying info.Model into the returned
SessionSnapshot, causing callers (e.g., internal/api/core/agent_identity.go that
reads caller.Session.Model) to see an empty model; update the constructor to set
the SessionSnapshot's Model field from info.Model (or clone it if a deep copy is
required) so the returned SessionSnapshot includes the session's model
information (reference: SessionSnapshot, session.Info, info.Model,
SessionSnapshotFromInfo).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `SessionSnapshotFromInfo` omits `info.Model` even though `/api/agent/me` surfaces `caller.Session.Model`. Sessions resolved through this helper can therefore return an empty model despite a populated daemon session record. Focused verification also showed the backing `session.Info`/store metadata path did not carry model data, so the root fix must propagate `Model` through the session/store read model as well as copy it into `SessionSnapshot.Model`.
- Resolution: Added model propagation through session/store metadata and identity snapshot conversion, then covered it through the validated identity response test and full `make verify`.
