---
status: resolved
file: internal/observe/observer.go
line: 362
severity: medium
author: claude-code
provider_ref:
---

# Issue 006: Observer resolves global agent def, ignoring workspace overrides

## Review Comment

`defaultPermissionModeResolver` at line 362 loads the agent definition using `aghconfig.LoadAgentDef(agentName, homePaths)`, which reads from the global `~/.agh/agents/` directory. However, session creation and resume resolve agents from `ResolvedWorkspace.Agents`, which merges workspace-local agents (local wins over global). If a workspace-local agent overrides the global agent's permission field, the observer's `permission_log.policy_used` will record the global permission mode, not the effective one that was actually applied at session startup.

This creates audit inconsistency: the permission log says one policy was used, but the session actually ran with a different one.

**Suggested fix:** Resolve the agent from the workspace's agent list instead of the global path. Since `defaultPermissionModeResolver` already has access to the workspace resolver (and resolves the workspace to get config), it can also use `ResolvedWorkspace.Agents` to find the correct agent definition:

```go
resolved, err := resolver.Resolve(ctx, workspaceID)
// ... use resolved.Agents to find the agent by name
// then cfg.ResolveAgent(workspaceAgentDef)
```

## Triage

- Decision: `valid`
- Root cause: `defaultPermissionModeResolver()` resolves workspace config from the workspace root, but then reloads the agent definition only from the global home directory. That bypasses the already-merged `ResolvedWorkspace.Agents` set, so workspace-local agent overrides are ignored in permission audit records.
- Fix plan: when a workspace ID is present, resolve the workspace once and select the effective agent definition from `ResolvedWorkspace.Agents`; only the no-workspace case should continue using the global home lookup.

## Resolution

- Updated the observer permission resolver to source agent definitions from `ResolvedWorkspace.Agents` whenever a workspace ID is present.
- Added regression coverage proving workspace-local agent definitions win over the global home copy for permission logging.
- Verified with targeted package tests and `make verify`.
