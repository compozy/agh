---
status: resolved
file: internal/session/manager_workspace.go
line: 156
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11e,comment:PRRC_kwDOR5y4QM663fBv
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Preserve the nil-input guards in the new session-agent resolver.**

The old `resolveWorkspaceAgent` path returned an error when `resolvedWorkspace` was nil. In this refactor, the `agentResolver != nil` branch skips that check and then dereferences `resolvedWorkspace.Config` on Line 152, which will panic instead of returning a normal error. The wrapper also dereferences `m.agentResolver` without guarding a nil receiver.

<details>
<summary>Suggested change</summary>

```diff
 func (m *Manager) resolveWorkspaceSessionAgent(
 	agentName string,
 	provider string,
 	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
 ) (aghconfig.ResolvedAgent, error) {
-	return resolveWorkspaceSessionAgent(agentName, provider, resolvedWorkspace, m.agentResolver)
+	var resolver AgentResolver
+	if m != nil {
+		resolver = m.agentResolver
+	}
+	return resolveWorkspaceSessionAgent(agentName, provider, resolvedWorkspace, resolver)
 }
@@
 func resolveWorkspaceSessionAgent(
 	agentName string,
 	provider string,
 	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
 	agentResolver AgentResolver,
 ) (aghconfig.ResolvedAgent, error) {
+	if resolvedWorkspace == nil {
+		return aghconfig.ResolvedAgent{}, errors.New("session: resolved workspace is required")
+	}
+
 	var (
 		agentDef aghconfig.AgentDef
 		err      error
 	)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (m *Manager) resolveWorkspaceSessionAgent(
	agentName string,
	provider string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) (aghconfig.ResolvedAgent, error) {
	var resolver AgentResolver
	if m != nil {
		resolver = m.agentResolver
	}
	return resolveWorkspaceSessionAgent(agentName, provider, resolvedWorkspace, resolver)
}

func resolveWorkspaceSessionAgent(
	agentName string,
	provider string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	agentResolver AgentResolver,
) (aghconfig.ResolvedAgent, error) {
	if resolvedWorkspace == nil {
		return aghconfig.ResolvedAgent{}, errors.New("session: resolved workspace is required")
	}

	var (
		agentDef aghconfig.AgentDef
		err      error
	)
	if agentResolver != nil {
		agentDef, err = agentResolver.ResolveAgent(agentName, resolvedWorkspace)
	} else {
		agentDef, err = resolveWorkspaceAgent(agentName, resolvedWorkspace)
	}
	if err != nil {
		return aghconfig.ResolvedAgent{}, err
	}

	resolved, err := resolvedWorkspace.Config.ResolveSessionAgent(agentDef, provider)
	if err != nil {
		return aghconfig.ResolvedAgent{}, err
	}
	return resolved, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_workspace.go` around lines 125 - 156, The new
resolveWorkspaceSessionAgent wrapper can panic when resolvedWorkspace or
m.agentResolver is nil: add explicit nil-input guards so both
Manager.resolveWorkspaceSessionAgent and resolveWorkspaceSessionAgent validate
resolvedWorkspace != nil and return a sensible error instead of dereferencing,
and ensure the call that uses m.agentResolver first checks m != nil and
m.agentResolver != nil before invoking AgentResolver.ResolveAgent; keep using
resolveWorkspaceAgent when agentResolver is nil but preserve its existing
nil-check behavior and return the same error paths.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `Manager.resolveWorkspaceSessionAgent` forwards `m.agentResolver` unguarded, so calling the method on a nil receiver panics before any validation runs.
- `resolveWorkspaceSessionAgent` also dereferences `resolvedWorkspace.Config` without first rejecting a nil workspace, so the old safe error path was lost in the refactor and needs to be restored with regression coverage.
- Resolved by restoring the nil-workspace error path, guarding nil manager receivers before reading `m.agentResolver`, and adding regression coverage for both cases in `internal/session/manager_integration_test.go`.
